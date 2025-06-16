package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
configDir: /custom/state
debounceInMs: 500
directories:
  - title: Config File
    source: /app/config.php
    destination: /var/www/config.php
  - title: Auth Module
    source: /app/auth
    destination: /var/www/auth
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ConfigDir != "/custom/state" {
		t.Errorf("Expected configDir /custom/state, got %s", cfg.ConfigDir)
	}

	if cfg.DebounceInMs != 500 {
		t.Errorf("Expected debounceInMs 500, got %d", cfg.DebounceInMs)
	}

	if len(cfg.Directories) != 2 {
		t.Fatalf("Expected 2 directories, got %d", len(cfg.Directories))
	}

	if cfg.Directories[0].Title != "Config File" {
		t.Errorf("Expected title 'Config File', got %s", cfg.Directories[0].Title)
	}

	if cfg.Directories[1].Source != "/app/auth" {
		t.Errorf("Expected source /app/auth, got %s", cfg.Directories[1].Source)
	}
}

func TestConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	srcPath := filepath.Join(tmpDir, "src")
	destPath := filepath.Join(tmpDir, "dest")

	configContent := `
directories:
  - title: Test
    source: ` + srcPath + `
    destination: ` + destPath + `
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedConfigDir := filepath.Join(cwd, ".gofilesync")
	if cfg.ConfigDir != expectedConfigDir {
		t.Errorf("Expected default configDir %s, got %s", expectedConfigDir, cfg.ConfigDir)
	}

	if cfg.DebounceInMs != 300 {
		t.Errorf("Expected default debounceInMs 300, got %d", cfg.DebounceInMs)
	}
}

func TestNoDirectoriesError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
configDir: /test
debounceInMs: 100
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing directories")
	}

	if err.Error() != "no directories defined in configuration" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestEmptyDirectoriesError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
directories: []
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for empty directories")
	}
}

func TestInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
directories:
  - title: Test
    source: /src
  destination: /dest
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestGetPaths(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	cfg := &Config{
		ConfigDir: stateDir,
	}

	if cfg.GetStateDir() != stateDir {
		t.Errorf("Wrong state dir: %s", cfg.GetStateDir())
	}

	expectedBackupDir := filepath.Join(stateDir, ".backups")
	if cfg.GetBackupDir() != expectedBackupDir {
		t.Errorf("Wrong backup dir: %s", cfg.GetBackupDir())
	}

	expectedLogFile := filepath.Join(stateDir, "sync.log")
	if cfg.GetLogFile() != expectedLogFile {
		t.Errorf("Wrong log file: %s", cfg.GetLogFile())
	}
}

func TestConfigWithAbsolutePaths(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create absolute paths for testing
	srcDir := filepath.Join(tmpDir, "source", "app")
	destDir := filepath.Join(tmpDir, "destination", "www")

	configContent := `
directories:
  - title: App Files
    source: ` + srcDir + `
    destination: ` + destDir + `
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Directories[0].Source != srcDir {
		t.Errorf("Expected source %s, got %s", srcDir, cfg.Directories[0].Source)
	}

	if cfg.Directories[0].Destination != destDir {
		t.Errorf("Expected destination %s, got %s", destDir, cfg.Directories[0].Destination)
	}
}
