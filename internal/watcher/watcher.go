package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	"github.com/ajxudir/gofilesync/internal/progress"
	syncpkg "github.com/ajxudir/gofilesync/internal/sync"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	config       *config.Config
	syncManager  *syncpkg.Manager
	logger       *logger.Logger
	style        progress.Styler
	pollInterval time.Duration

	// syncMu serializes runs so debounce goroutines never sync concurrently;
	// pending coalesces changes that arrive mid-sync into one follow-up run.
	syncMu  sync.Mutex
	running bool
	pending bool
}

func New(cfg *config.Config, manager *syncpkg.Manager, log *logger.Logger) *Watcher {
	return &Watcher{
		config:      cfg,
		syncManager: manager,
		logger:      log,
		style:       progress.NewStyler(os.Stderr),
	}
}

func (w *Watcher) SetPollInterval(ms int) {
	w.pollInterval = time.Duration(ms) * time.Millisecond
}

func (w *Watcher) runAndReport() {
	w.syncMu.Lock()
	if w.running {
		w.pending = true
		w.syncMu.Unlock()
		return
	}
	w.running = true
	w.syncMu.Unlock()

	for {
		w.syncOnce()

		w.syncMu.Lock()
		if w.pending {
			w.pending = false
			w.syncMu.Unlock()
			continue
		}
		w.running = false
		w.syncMu.Unlock()
		return
	}
}

type changeGroup struct {
	changed  []string
	restored []string
	removed  []string
}

func (w *Watcher) syncOnce() {
	res, err := w.syncManager.Sync()
	if err != nil {
		w.logger.Error("Sync failed: " + err.Error())
		return
	}
	if res.Changes() == 0 {
		return
	}

	var order []string
	groups := map[string]*changeGroup{}
	get := func(title string) *changeGroup {
		g, ok := groups[title]
		if !ok {
			g = &changeGroup{}
			groups[title] = g
			order = append(order, title)
		}
		return g
	}
	for _, c := range res.Synced {
		g := get(c.Title)
		g.changed = append(g.changed, c.Rel)
	}
	for _, c := range res.Restored {
		g := get(c.Title)
		g.restored = append(g.restored, c.Rel)
	}
	for _, c := range res.Removed {
		g := get(c.Title)
		g.removed = append(g.removed, c.Rel)
	}

	ts := time.Now().Format("2006-01-02 15:04:05")
	w.logger.Blank()
	w.logger.Line("  "+w.style.Dim("["+ts+"]")+"  "+w.style.Cyan("INFO"), "  ["+ts+"]  INFO")
	for _, title := range order {
		g := groups[title]
		w.report(title, "changed", g.changed)
		w.report(title, "restored", g.restored)
		w.report(title, "removed", g.removed)
	}
}

func (w *Watcher) report(title, verb string, items []string) {
	var phrase string
	switch n := len(items); {
	case n == 0:
		return
	case n == 1:
		phrase = verb + " " + items[0]
	default:
		phrase = fmt.Sprintf("%d %s", n, verb)
	}
	w.logger.Line(
		"    "+w.style.Accent("▸")+" "+w.style.Bold(title)+"  "+w.style.Dim("("+phrase+")"),
		"    ▸ "+title+"  ("+phrase+")",
	)
}

func (w *Watcher) Start() error {
	if w.pollInterval > 0 {
		return w.startPolling()
	}
	return w.startWatching()
}

func (w *Watcher) startWatching() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, dir := range w.config.Sync {
		if dir.Once {
			continue
		}
		if err := w.addPath(watcher, dir.Source); err != nil {
			w.logger.Error(fmt.Sprintf("Could not watch %q (%s): %v", dir.Title, dir.Source, err))
		}
	}

	var debounceTimer *time.Timer
	var mu sync.Mutex

	runSync := func() {
		w.runAndReport()
	}

	debounce := time.Duration(w.config.DebounceInMs) * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				// Watch newly-created dirs too, else edits inside them are missed.
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						if err := w.addPath(watcher, event.Name); err != nil {
							w.logger.Error(fmt.Sprintf("Could not watch new directory %s: %v", event.Name, err))
						}
					}
				}

				mu.Lock()
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
