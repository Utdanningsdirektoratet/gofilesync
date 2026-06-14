package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// startPolling compares modification times on an interval — the reliable mode
// for mounted volumes where fsnotify doesn't fire (e.g. Docker on macOS).
func (w *Watcher) startPolling() error {
	lastModTimes := make(map[string]time.Time)
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
			w.runAndReport()
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
		if currentMod := info.ModTime(); !exists || !lastMod.Equal(currentMod) {
			if exists {
				w.logger.Debug("Source modified: " + p)
			} else {
				w.logger.Debug("Source added: " + p)
			}
			modTimes[p] = currentMod
			changed = true
		}
		return nil
	})

	for file := range modTimes {
		if strings.HasPrefix(file, path) {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				w.logger.Debug("Source removed: " + file)
				delete(modTimes, file)
				changed = true
			}
		}
	}

	return changed
}
