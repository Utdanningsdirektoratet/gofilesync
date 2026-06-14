package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ajxudir/gofilesync/internal/progress"
)

type Logger struct {
	file      *os.File
	mu        sync.Mutex
	dryRun    bool
	debugMode bool
	color     bool
}

const (
	LevelDEBUG = "DEBUG"
	LevelINFO  = "INFO"
	LevelWARN  = "WARN"
	LevelERROR = "ERROR"
)

func New(logPath string, dryRun bool) (*Logger, error) {
	color := progress.ColorEnabled(os.Stderr)
	if dryRun {
		return &Logger{dryRun: true, color: color}, nil
	}

	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &Logger{file: file, dryRun: dryRun, color: color}, nil
}

func levelColor(level string) string {
	switch level {
	case LevelERROR:
		return "31"
	case LevelWARN:
		return "33"
	case LevelDEBUG:
		return "2"
	default:
		return "36"
	}
}

func (l *Logger) SetDebugMode(debug bool) {
	l.debugMode = debug
}

func (l *Logger) log(level, message string) {
	if level == LevelDEBUG && !l.debugMode {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	plain := fmt.Sprintf("[%s]  %-5s  %s", timestamp, level, message)

	console := plain
	if l.color {
		console = fmt.Sprintf("\x1b[2m[%s]\x1b[0m  \x1b[%sm%-5s\x1b[0m  %s",
			timestamp, levelColor(level), level, message)
	}
	fmt.Fprintln(os.Stderr, console)

	if !l.dryRun && l.file != nil {
		l.file.WriteString(plain + "\n")
	}
}

func (l *Logger) Debug(message string) {
	l.log(LevelDEBUG, message)
}

func (l *Logger) Info(message string) {
	l.log(LevelINFO, message)
}

func (l *Logger) Warn(message string) {
	l.log(LevelWARN, message)
}

func (l *Logger) Error(message string) {
	l.log(LevelERROR, message)
}

func (l *Logger) Line(console, file string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(os.Stderr, console)
	if !l.dryRun && l.file != nil {
		l.file.WriteString(file + "\n")
	}
}

func (l *Logger) Blank() {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(os.Stderr)
	if !l.dryRun && l.file != nil {
		l.file.WriteString("\n")
	}
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
