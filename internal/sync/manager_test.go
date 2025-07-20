package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
)

func TestManagerCreation(t *testing.T) {
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

	manager, err := NewManager(cfg, log, false)
	if err != nil {
		t.Fatal(err)
	}

	if manager.config != cfg {
		t.Error("Config not set correctly")
	}

	if _, err := os.Stat(cfg.GetStateDir()); os.IsNotExist(err) {
		t.Error("State directory should be created")
	}

	if _, err := os.Stat(cfg.GetBackupDir()); os.IsNotExist(err) {
		t.Error("Backup directory should be created")
	}
}

func TestSyncFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(destDir, 0755)

	srcFile := filepath.Join(srcDir, "test.txt")
	destFile := filepath.Join(destDir, "test.txt")

	os.WriteFile(srcFile, []byte("content"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Test File", Source: srcFile, Destination: destFile},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "content" {
		t.Error("File content mismatch")
	}
}

func TestSyncDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src", "mydir")
	destDir := filepath.Join(tmpDir, "dest", "mydir")

	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file2"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Test Dir", Source: srcDir, Destination: destDir},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Check files
	file1 := filepath.Join(destDir, "file1.txt")
	file2 := filepath.Join(destDir, "subdir", "file2.txt")

	content1, _ := os.ReadFile(file1)
	if string(content1) != "file1" {
		t.Error("file1 content mismatch")
	}

	content2, _ := os.ReadFile(file2)
	if string(content2) != "file2" {
		t.Error("file2 content mismatch")
	}
}

func TestSyncSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(destDir, 0755)

	targetFile := filepath.Join(srcDir, "target.txt")
	linkFile := filepath.Join(srcDir, "link.txt")
	destFile := filepath.Join(destDir, "link.txt")

	os.WriteFile(targetFile, []byte("target content"), 0644)
	os.Symlink(targetFile, linkFile)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Symlink", Source: linkFile, Destination: destFile},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Should copy content, not create symlink
	info, _ := os.Lstat(destFile)
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("Destination should not be a symlink")
	}

	content, _ := os.ReadFile(destFile)
	if string(content) != "target content" {
		t.Error("Should copy symlink target content")
	}
}

func TestSyncWithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src", "test.txt")
	destFile := filepath.Join(tmpDir, "dest", "test.txt")

	os.MkdirAll(filepath.Dir(srcFile), 0755)
	os.MkdirAll(filepath.Dir(destFile), 0755)

	os.WriteFile(destFile, []byte("original"), 0644)
	os.WriteFile(srcFile, []byte("updated"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Update", Source: srcFile, Destination: destFile},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(destFile)
	if string(content) != "updated" {
		t.Error("File should be updated")
	}

	// Check backup exists
	backupPath := manager.backupManager.GetBackupPath(destFile)
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal("Backup should exist")
	}

	if string(backupContent) != "original" {
		t.Error("Backup should contain original content")
	}
}

func TestSyncUnchangedFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src", "test.txt")
	destFile := filepath.Join(tmpDir, "dest", "test.txt")

	os.MkdirAll(filepath.Dir(srcFile), 0755)
	os.MkdirAll(filepath.Dir(destFile), 0755)

	content := []byte("same content")
	os.WriteFile(srcFile, content, 0644)
	os.WriteFile(destFile, content, 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Unchanged", Source: srcFile, Destination: destFile},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	// No backup should be created
	backupPath := manager.backupManager.GetBackupPath(destFile)
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("No backup should be created for unchanged files")
	}
}

func TestSyncMissingSourceWithExistingDest(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{
				Title:       "Missing Source",
				Source:      filepath.Join(tmpDir, "nonexistent", "file.txt"),
				Destination: filepath.Join(tmpDir, "dest", "file.txt"),
			},
		},
	}

	// Create destination file and backup
	destFile := filepath.Join(tmpDir, "dest", "file.txt")
	os.MkdirAll(filepath.Dir(destFile), 0755)
	os.WriteFile(destFile, []byte("existing content"), 0644)

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)

	// Create backup manually
	backupPath := manager.backupManager.GetBackupPath(destFile)
	os.MkdirAll(filepath.Dir(backupPath), 0755)
	os.WriteFile(backupPath, []byte("backup content"), 0644)

	// Run sync with missing source
	err := manager.Run()
	if err != nil {
		t.Error("Should not error on missing source")
	}

	// Verify file was restored from backup
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatal("Destination file should exist")
	}

	if string(content) != "backup content" {
		t.Errorf("File should be restored from backup, got: %s", content)
	}
}

func TestMultipleDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple source files
	src1 := filepath.Join(tmpDir, "src1", "file1.txt")
	src2 := filepath.Join(tmpDir, "src2", "file2.txt")
	dest1 := filepath.Join(tmpDir, "dest1", "file1.txt")
	dest2 := filepath.Join(tmpDir, "dest2", "file2.txt")

	os.MkdirAll(filepath.Dir(src1), 0755)
	os.MkdirAll(filepath.Dir(src2), 0755)
	os.WriteFile(src1, []byte("content1"), 0644)
	os.WriteFile(src2, []byte("content2"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "First", Source: src1, Destination: dest1},
			{Title: "Second", Source: src2, Destination: dest2},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	content1, _ := os.ReadFile(dest1)
	if string(content1) != "content1" {
		t.Error("First file not synced correctly")
	}

	content2, _ := os.ReadFile(dest2)
	if string(content2) != "content2" {
		t.Error("Second file not synced correctly")
	}
}

func TestFileRemovedFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src", "test.txt")
	destFile := filepath.Join(tmpDir, "dest", "test.txt")

	os.MkdirAll(filepath.Dir(srcFile), 0755)
	os.MkdirAll(filepath.Dir(destFile), 0755)

	// Initial setup with file in config
	os.WriteFile(srcFile, []byte("source content"), 0644)
	os.WriteFile(destFile, []byte("original content"), 0644)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Test File", Source: srcFile, Destination: destFile},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	// First sync - creates backup
	manager, _ := NewManager(cfg, log, false)
	manager.Run()

	// Verify file was synced
	content, _ := os.ReadFile(destFile)
	if string(content) != "source content" {
		t.Error("File should be synced")
	}

	// Remove file from config
	cfg2 := &config.Config{
		ConfigDir: tmpDir,
		Sync:      []config.Directory{}, // Empty - file removed from config
	}

	// Run sync with new config
	manager2, _ := NewManager(cfg2, log, false)
	manager2.Run()

	// File should still exist (not touched when removed from config)
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("File should not be deleted when removed from config")
	}
}

func TestSyncDirectoryWithSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)

	// Create files and symlinks
	targetFile := filepath.Join(srcDir, "target.txt")
	regularFile := filepath.Join(srcDir, "regular.txt")
	symlinkFile := filepath.Join(srcDir, "subdir", "link.txt")

	os.WriteFile(targetFile, []byte("target"), 0644)
	os.WriteFile(regularFile, []byte("regular"), 0644)
	os.Symlink(filepath.Join("..", "target.txt"), symlinkFile)

	cfg := &config.Config{
		ConfigDir: tmpDir,
		Sync: []config.Directory{
			{Title: "Mixed Dir", Source: srcDir, Destination: destDir},
		},
	}

	log, _ := logger.New("", true)
	defer log.Close()

	manager, _ := NewManager(cfg, log, false)
	err := manager.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Check regular file
	regularDest := filepath.Join(destDir, "regular.txt")
	content, _ := os.ReadFile(regularDest)
	if string(content) != "regular" {
		t.Error("Regular file content mismatch")
	}

	// Check symlink was resolved and content copied
	linkDest := filepath.Join(destDir, "subdir", "link.txt")
	info, _ := os.Lstat(linkDest)
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("Symlink should be resolved to regular file")
	}

	linkContent, _ := os.ReadFile(linkDest)
	if string(linkContent) != "target" {
		t.Error("Symlink target content should be copied")
	}
}
