package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	file      *os.File
	mu        sync.Mutex
	dryRun    bool
	debugMode bool
}

const (
	LevelDEBUG = "DEBUG"
	LevelINFO  = "INFO"
	LevelERROR = "ERROR"
)

func New(logPath string, dryRun bool) (*Logger, error) {
	if dryRun {
		return &Logger{dryRun: true}, nil
	}

	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &Logger{file: file, dryRun: dryRun}, nil
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
	levelPadded := fmt.Sprintf("%-5s", level)
	logLine := fmt.Sprintf("[%s]  level=%s  %s", timestamp, levelPadded, message)

	fmt.Println(logLine)

	if !l.dryRun && l.file != nil {
		l.file.WriteString(logLine + "\n")
	}
}

func (l *Logger) Debug(message string) {
	l.log(LevelDEBUG, message)
}

func (l *Logger) Info(message string) {
	l.log(LevelINFO, message)
}

func (l *Logger) Error(message string) {
	l.log(LevelERROR, message)
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
