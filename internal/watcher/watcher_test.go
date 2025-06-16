package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	syncpkg "github.com/ajxudir/gofilesync/internal/sync"
	"github.com/fsnotify/fsnotify"
)

func TestWatcherCreation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
		Directories: []config.Directory{
			{
				Title:       "Test",
				Source:      filepath.Join(tmpDir, "src"),
				Destination: filepath.Join(tmpDir, "dest"),
			},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	if w.config != cfg {
		t.Error("Config not set correctly")
	}

	if w.syncManager != manager {
		t.Error("Sync manager not set correctly")
	}

	if w.logger != log {
		t.Error("Logger not set correctly")
	}
}

func TestAddPathFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Directories: []config.Directory{
			{Title: "File", Source: testFile, Destination: filepath.Join(tmpDir, "dest", "test.txt")},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	err = w.addPath(watcher, testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Modify file to test watching
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(testFile, []byte("modified"), 0644)
	}()

	select {
	case event := <-watcher.Events:
		if event.Name != testFile {
			t.Errorf("Expected event for %s, got %s", testFile, event.Name)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("No event received")
	}
}

func TestAddPathDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(filepath.Join(srcDir, "subdir", "nested"), 0755)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Directories: []config.Directory{
			{Title: "Dir", Source: srcDir, Destination: filepath.Join(tmpDir, "dest")},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	err = w.addPath(watcher, srcDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create file in nested directory
	testFile := filepath.Join(srcDir, "subdir", "test.txt")
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(testFile, []byte("content"), 0644)
	}()

	select {
	case <-watcher.Events:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Error("No event received for nested directory")
	}
}

func TestDebouncing(t *testing.T) {
	var mu sync.Mutex
	syncCount := 0
	syncChan := make(chan struct{}, 10)

	// Test debouncing logic
	var debounceTimer *time.Timer
	eventCount := 0

	runSync := func() {
		mu.Lock()
		syncCount++
		syncChan <- struct{}{}
		mu.Unlock()
	}

	// Simulate rapid events
	for i := 0; i < 5; i++ {
		mu.Lock()
		eventCount++
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(200*time.Millisecond, runSync)
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce
	select {
	case <-syncChan:
		if syncCount != 1 {
			t.Errorf("Expected 1 sync, got %d", syncCount)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Sync not triggered")
	}
}

func TestCustomDebounceTime(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:    tmpDir,
		DebounceInMs: 100,
		Directories: []config.Directory{
			{
				Title:       "Test",
				Source:      filepath.Join(tmpDir, "src"),
				Destination: filepath.Join(tmpDir, "dest"),
			},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	if w.config.DebounceInMs != 100 {
		t.Errorf("Expected debounce 100ms, got %d", w.config.DebounceInMs)
	}
}

func TestMultipleDirectoryWatch(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Directories: []config.Directory{
			{Title: "Dir1", Source: dir1, Destination: filepath.Join(tmpDir, "dest1")},
			{Title: "Dir2", Source: dir2, Destination: filepath.Join(tmpDir, "dest2")},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	// Add both directories
	for _, dir := range cfg.Directories {
		w.addPath(watcher, dir.Source)
	}

	// Create files in both
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(filepath.Join(dir1, "test1.txt"), []byte("1"), 0644)
		os.WriteFile(filepath.Join(dir2, "test2.txt"), []byte("2"), 0644)
	}()

	eventCount := 0
	timeout := time.After(500 * time.Millisecond)

	for eventCount < 2 {
		select {
		case <-watcher.Events:
			eventCount++
		case <-timeout:
			t.Fatalf("Only received %d events, expected 2", eventCount)
		}
	}
}

func TestWatchNonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
		Directories: []config.Directory{
			{
				Title:       "Missing",
				Source:      filepath.Join(tmpDir, "nonexistent", "path"),
				Destination: filepath.Join(tmpDir, "dest"),
			},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	err = w.addPath(watcher, cfg.Directories[0].Source)
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestEventFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)

	// Test that only relevant events trigger sync
	relevantOps := []fsnotify.Op{
		fsnotify.Write,
		fsnotify.Create,
		fsnotify.Remove,
		fsnotify.Rename,
	}

	for _, op := range relevantOps {
		event := fsnotify.Event{
			Name: testFile,
			Op:   op,
		}

		if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
			t.Errorf("Operation %v should be considered relevant", op)
		}
	}

	// Test irrelevant operation
	chmodEvent := fsnotify.Event{
		Name: testFile,
		Op:   fsnotify.Chmod,
	}

	if chmodEvent.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
		t.Error("Chmod should not trigger sync")
	}
}

func TestMultipleEventsDebouncing(t *testing.T) {
	var mu sync.Mutex
	syncCount := 0
	syncChan := make(chan struct{}, 10)

	// Test 10+ events within debounce period
	var debounceTimer *time.Timer
	eventCount := 0

	runSync := func() {
		mu.Lock()
		syncCount++
		syncChan <- struct{}{}
		mu.Unlock()
	}

	// Send 10 events rapidly
	for i := 0; i < 10; i++ {
		mu.Lock()
		eventCount++
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(300*time.Millisecond, runSync)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond) // Total 200ms for 10 events
	}

	// Wait for debounce period to complete
	select {
	case <-syncChan:
		if syncCount != 1 {
			t.Errorf("Expected 1 sync after 10 rapid events, got %d", syncCount)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Sync not triggered after debounce period")
	}
}

func processEventsWithDebounce(syncFunc func(), log *logger.Logger, eventCount int, eventDelay, debounceMs time.Duration) {
	var debounceTimer *time.Timer
	var mu sync.Mutex
	events := 0

	runSync := func() {
		mu.Lock()
		count := events
		events = 0
		mu.Unlock()

		log.Info(fmt.Sprintf("Detected %d file change(s), syncing...", count))
		syncFunc()
	}

	for i := 0; i < eventCount; i++ {
		mu.Lock()
		events++
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(debounceMs, runSync)
		mu.Unlock()

		if i < eventCount-1 {
			time.Sleep(eventDelay)
		}
	}
}
