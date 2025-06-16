package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
)

func TestGetBackupPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: filepath.Join(tmpDir, ".gofilesync"),
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	tests := []struct {
		original string
		contains []string
	}{
		{
			filepath.Join(tmpDir, "www", "config.php"),
			[]string{".backups", "config.php"},
		},
		{
			filepath.Join(tmpDir, "app", "data", "test.txt"),
			[]string{".backups", "test.txt"},
		},
	}

	for _, test := range tests {
		result := backup.GetBackupPath(test.original)
		for _, expected := range test.contains {
			if !strings.Contains(result, expected) {
				t.Errorf("Backup path %s should contain %s", result, expected)
			}
		}
	}
}

func TestBackupPathUniqueness(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: filepath.Join(tmpDir, ".gofilesync"),
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	// Same filename, different paths
	path1 := backup.GetBackupPath(filepath.Join(tmpDir, "app", "config.php"))
	path2 := backup.GetBackupPath(filepath.Join(tmpDir, "var", "config.php"))

	if path1 == path2 {
		t.Error("Different source paths should have different backup paths")
	}
}

func TestBackup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	srcFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(srcFile, []byte("original"), 0644)

	err := backup.Backup(srcFile)
	if err != nil {
		t.Fatal(err)
	}

	backupPath := backup.GetBackupPath(srcFile)
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal("Backup file should exist")
	}

	if string(content) != "original" {
		t.Error("Backup content mismatch")
	}
}

func TestBackupSkipExisting(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	srcFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(srcFile, []byte("original"), 0644)

	// First backup
	backup.Backup(srcFile)

	// Modify source
	os.WriteFile(srcFile, []byte("modified"), 0644)

	// Second backup should skip
	backup.Backup(srcFile)

	backupPath := backup.GetBackupPath(srcFile)
	content, _ := os.ReadFile(backupPath)

	if string(content) != "original" {
		t.Error("Existing backup should not be overwritten")
	}
}

func TestBackupDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(true)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	srcFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(srcFile, []byte("content"), 0644)

	err := backup.Backup(srcFile)
	if err != nil {
		t.Fatal(err)
	}

	backupPath := backup.GetBackupPath(srcFile)
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Dry run should not create backup")
	}
}

func TestRestore(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	destFile := filepath.Join(tmpDir, "test.txt")
	backupPath := backup.GetBackupPath(destFile)

	os.MkdirAll(filepath.Dir(backupPath), 0755)
	os.WriteFile(backupPath, []byte("backup content"), 0644)
	os.WriteFile(destFile, []byte("current"), 0644)

	err := backup.Restore(destFile)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "backup content" {
		t.Error("File should be restored from backup")
	}
}

func TestRestoreNoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	destFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(destFile, []byte("to delete"), 0644)

	err := backup.Restore(destFile)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(destFile); !os.IsNotExist(err) {
		t.Error("File should be deleted when no backup exists")
	}
}

func TestRestoreDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(true)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	destFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(destFile, []byte("current"), 0644)

	err := backup.Restore(destFile)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(destFile)
	if string(content) != "current" {
		t.Error("Dry run should not modify file")
	}
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(false)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	// Create some backups
	backupDir := cfg.GetBackupDir()
	testFiles := []string{
		filepath.Join(backupDir, "test1.txt"),
		filepath.Join(backupDir, "subdir", "test2.txt"),
		filepath.Join(backupDir, "subdir", "nested", "test3.txt"),
	}

	for _, file := range testFiles {
		os.MkdirAll(filepath.Dir(file), 0755)
		os.WriteFile(file, []byte("backup"), 0644)
	}

	err := backup.Clear()
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range testFiles {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Errorf("Backup file %s should be deleted", file)
		}
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory should be recreated")
	}
}

func TestClearDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tmpDir,
	}

	fileOps := NewFileOperations(true)
	log, _ := logger.New("", true)
	defer log.Close()

	backup := NewBackupManager(cfg, fileOps, log)

	backupFile := filepath.Join(cfg.GetBackupDir(), "test.txt")
	os.MkdirAll(filepath.Dir(backupFile), 0755)
	os.WriteFile(backupFile, []byte("backup"), 0644)

	err := backup.Clear()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Dry run should not delete backups")
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/var/www/html", "_var_www_html"},
		{"C:\\Users\\test", "C__Users_test"},
		{"/app/data:config", "_app_data_config"},
		{"relative/path", "relative_path"},
	}

	for _, test := range tests {
		result := sanitizePath(test.input)
		if result != test.expected {
			t.Errorf("sanitizePath(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}
