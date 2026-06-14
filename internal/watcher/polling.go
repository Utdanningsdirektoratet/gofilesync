package watcher

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// startPolling compares modification times on an interval — the reliable mode
// for mounted volumes where fsnotify doesn't fire (e.g. Docker on macOS).
func (w *Watcher) startPolling() error {
	lastModTimes := make(map[string]time.Time)
	for _, dir := range w.config.Sync {
		if dir.Once {
			continue
		}
		for p, mt := range w.snapshot(dir.Source) {
			lastModTimes[p] = mt
		}
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		changed := false
		for _, dir := range w.config.Sync {
			if dir.Once {
				continue
			}
			if w.hasChanges(dir.Source, lastModTimes) {
				changed = true
			}
		}
		if changed {
			w.runAndReport()
		}
	}
	return nil
}

func (w *Watcher) snapshot(path string) map[string]time.Time {
	var paths []string
	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			paths = append(paths, p)
		}
		return nil
	})

	out := make(map[string]time.Time, len(paths))
	var mu sync.Mutex
	workers := w.config.Workers
	if workers < 1 {
		workers = 1
	}
	ch := make(chan string, 256)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range ch {
				if fi, err := os.Stat(p); err == nil {
					mu.Lock()
					out[p] = fi.ModTime()
					mu.Unlock()
				}
			}
		}()
	}
	for _, p := range paths {
		ch <- p
	}
	close(ch)
	wg.Wait()
	return out
}

func (w *Watcher) hasChanges(path string, modTimes map[string]time.Time) bool {
	cur := w.snapshot(path)
	changed := false

	for p, mt := range cur {
		if old, ok := modTimes[p]; !ok || !old.Equal(mt) {
			modTimes[p] = mt
			changed = true
		}
	}
	for p := range modTimes {
		if underPath(p, path) {
			if _, ok := cur[p]; !ok {
				delete(modTimes, p)
				changed = true
			}
		}
	}

	return changed
}

func underPath(p, path string) bool {
	return p == path || strings.HasPrefix(p, path+string(os.PathSeparator))
}
