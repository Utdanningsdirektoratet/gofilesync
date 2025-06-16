package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	syncpkg "github.com/ajxudir/gofilesync/internal/sync"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	config      *config.Config
	syncManager *syncpkg.Manager
	logger      *logger.Logger
}

func New(cfg *config.Config, manager *syncpkg.Manager, log *logger.Logger) *Watcher {
	return &Watcher{
		config:      cfg,
		syncManager: manager,
		logger:      log,
	}
}

func (w *Watcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, dir := range w.config.Directories {
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

	debounceMs := time.Duration(w.config.DebounceInMs) * time.Millisecond

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

				debounceTimer = time.AfterFunc(debounceMs, runSync)
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
