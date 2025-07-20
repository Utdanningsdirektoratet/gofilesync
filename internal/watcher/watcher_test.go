package watcher

import (
	"os"
	"path/filepath"
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
		Sync: []config.Directory{
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

func TestSetPollInterval(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
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

	// Test setting poll interval
	w.SetPollInterval(300)
	if w.pollInterval != 300*time.Millisecond {
		t.Errorf("Expected poll interval 300ms, got %v", w.pollInterval)
	}
}

func TestPollingMode(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(destDir, 0755)

	// Create initial file
	testFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{
				Title:       "Test",
				Source:      srcDir,
				Destination: destDir,
			},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	// Use real sync manager in dry-run mode
	manager, _ := syncpkg.NewManager(cfg, log, true)

	w := New(cfg, manager, log)
	w.SetPollInterval(100)

	// Run polling for a short time
	done := make(chan bool)
	go func() {
		time.Sleep(150 * time.Millisecond)
		// Modify file
		os.WriteFile(testFile, []byte("modified"), 0644)
		time.Sleep(150 * time.Millisecond)
		done <- true
	}()

	// Start polling (will run until we signal done)
	go func() {
		<-done
		// In a real test we'd have a way to stop the watcher
		// For now, the test will end and clean up
	}()

	// Let polling run briefly
	go w.Start()
	<-done
}

func TestScanPath(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)

	// Create test files
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("2"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{
				Title:       "Test",
				Source:      srcDir,
				Destination: filepath.Join(tmpDir, "dest"),
			},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	modTimes := make(map[string]time.Time)
	w.scanPath(srcDir, modTimes)

	if len(modTimes) != 2 {
		t.Errorf("Expected 2 files scanned, got %d", len(modTimes))
	}
}

func TestHasChanges(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	testFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{
				Title:       "Test",
				Source:      srcDir,
				Destination: filepath.Join(tmpDir, "dest"),
			},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := syncpkg.NewManager(cfg, log, true)
	w := New(cfg, manager, log)

	modTimes := make(map[string]time.Time)
	w.scanPath(srcDir, modTimes)

	// No changes yet
	if w.hasChanges(srcDir, modTimes) {
		t.Error("Should not detect changes when nothing changed")
	}

	// Modify file
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	os.WriteFile(testFile, []byte("modified"), 0644)

	// Should detect changes
	if !w.hasChanges(srcDir, modTimes) {
		t.Error("Should detect changes after file modification")
	}

	// No changes after update
	if w.hasChanges(srcDir, modTimes) {
		t.Error("Should not detect changes after updating mod times")
	}
}

func TestAddPathFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
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
		Sync: []config.Directory{
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
