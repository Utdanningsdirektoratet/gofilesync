package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	syncpkg "github.com/ajxudir/gofilesync/internal/sync"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	config       *config.Config
	syncManager  *syncpkg.Manager
	logger       *logger.Logger
	pollInterval time.Duration
}

func New(cfg *config.Config, manager *syncpkg.Manager, log *logger.Logger) *Watcher {
	return &Watcher{
		config:      cfg,
		syncManager: manager,
		logger:      log,
	}
}

func (w *Watcher) SetPollInterval(ms int) {
	w.pollInterval = time.Duration(ms) * time.Millisecond
}

func (w *Watcher) Start() error {
	if w.pollInterval > 0 {
		return w.startPolling()
	}
	return w.startWatching()
}

func (w *Watcher) startPolling() error {
	w.logger.Info(fmt.Sprintf("Starting polling mode (interval: %dms)", w.pollInterval.Milliseconds()))

	lastModTimes := make(map[string]time.Time)

	// Initial scan
	for _, dir := range w.config.Sync {
		w.scanPath(dir.Source, lastModTimes)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		changed := false

		for _, dir := range w.config.Sync {
			if w.hasChanges(dir.Source, lastModTimes) {
				changed = true
			}
		}

		if changed {
			w.logger.Info("Detected file changes via polling, syncing...")
			if err := w.syncManager.Run(); err != nil {
				w.logger.Error("Sync failed: " + err.Error())
			}
		}
	}

	return nil
}

func (w *Watcher) scanPath(path string, modTimes map[string]time.Time) {
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		modTimes[p] = info.ModTime()
		return nil
	})
}

func (w *Watcher) hasChanges(path string, modTimes map[string]time.Time) bool {
	changed := false

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		lastMod, exists := modTimes[p]
		currentMod := info.ModTime()

		if !exists || !lastMod.Equal(currentMod) {
			modTimes[p] = currentMod
			changed = true
		}

		return nil
	})

	// Check for deleted files
	for file := range modTimes {
		if strings.HasPrefix(file, path) {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				delete(modTimes, file)
				changed = true
			}
		}
	}

	return changed
}

func (w *Watcher) startWatching() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, dir := range w.config.Sync {
		if err := w.addPath(watcher, dir.Source); err != nil {
			w.logger.Error(fmt.Sprintf("Failed to watch %s (%s): %v", dir.Title, dir.Source, err))
		}
	}

	var debounceTimer *time.Timer
	var mu sync.Mutex
	eventCount := 0

	runSync := func() {
		mu.Lock()
		count := eventCount
		eventCount = 0
		mu.Unlock()

		w.logger.Info(fmt.Sprintf("Detected %d file change(s), syncing...", count))
		if err := w.syncManager.Run(); err != nil {
			w.logger.Error("Sync failed: " + err.Error())
		}
	}

	debounce := time.Duration(w.config.DebounceInMs) * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				mu.Lock()
				eventCount++

				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounce, runSync)
				mu.Unlock()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			w.logger.Error("Watch error: " + err.Error())
		}
	}
}

func (w *Watcher) addPath(watcher *fsnotify.Watcher, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return watcher.Add(filepath.Dir(path))
	}

	if err := watcher.Add(path); err != nil {
		return err
	}

	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != path {
			watcher.Add(p)
		}
		return nil
	})
}
