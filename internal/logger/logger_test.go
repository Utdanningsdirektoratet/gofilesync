package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	log, err := New(logPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	if log.file == nil {
		t.Error("Logger file should be created")
	}

	if log.dryRun {
		t.Error("Logger should not be in dry run mode")
	}
}

func TestDryRunLogger(t *testing.T) {
	log, err := New("", true)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	if log.file != nil {
		t.Error("Dry run logger should not create file")
	}

	if !log.dryRun {
		t.Error("Logger should be in dry run mode")
	}
}

func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	log, err := New(logPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	log.Info("Info message")
	log.Debug("Debug message")
	log.Error("Error message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 log lines (debug filtered), got %d", len(lines))
	}

	tests := []struct {
		line  int
		level string
		msg   string
	}{
		{0, "INFO", "Info message"},
		{1, "ERROR", "Error message"},
	}

	for _, test := range tests {
		if !strings.Contains(lines[test.line], "level="+test.level) {
			t.Errorf("Line %d: missing level=%s", test.line, test.level)
		}
		if !strings.Contains(lines[test.line], test.msg) {
			t.Errorf("Line %d: missing message '%s'", test.line, test.msg)
		}
	}
}

func TestLogFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	log, err := New(logPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	log.Info("Test message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	line := string(content)

	// Check format: [YYYY-MM-DD HH:MM:SS]  level=INFO   Message
	if !strings.HasPrefix(line, "[") {
		t.Error("Log should start with [")
	}

	if !strings.Contains(line, "]  level=") {
		t.Error("Log should have proper spacing after timestamp")
	}

	// Extract timestamp
	end := strings.Index(line, "]")
	if end == -1 {
		t.Fatal("No closing ] found")
	}

	timestamp := line[1:end]
	_, err = time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		t.Errorf("Invalid timestamp format: %v", err)
	}
}

func TestLoggerCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nested", "dir", "test.log")

	log, err := New(logPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	if _, err := os.Stat(filepath.Dir(logPath)); os.IsNotExist(err) {
		t.Error("Logger should create necessary directories")
	}
}

func TestDebugMode(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	log, err := New(logPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Debug off by default
	log.Debug("Should not appear")
	log.Info("Should appear")

	content, _ := os.ReadFile(logPath)
	if strings.Contains(string(content), "Should not appear") {
		t.Error("Debug message should not be logged when debug mode is off")
	}
	if !strings.Contains(string(content), "Should appear") {
		t.Error("Info message should be logged")
	}

	// Enable debug mode
	log.SetDebugMode(true)
	log.Debug("Debug enabled")

	content2, _ := os.ReadFile(logPath)
	if !strings.Contains(string(content2), "Debug enabled") {
		t.Error("Debug message should be logged when debug mode is on")
	}
}
