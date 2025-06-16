package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	"github.com/ajxudir/gofilesync/internal/sync"
	"github.com/ajxudir/gofilesync/internal/watcher"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: gofilesync <config.yaml> <command> [options]")
		fmt.Println("Commands: sync, watch, clear-cache")
		fmt.Println("Options: --dry-run, --debug")
		os.Exit(1)
	}

	configPath := os.Args[1]
	command := os.Args[2]
	dryRun := false
	debug := false

	for _, arg := range os.Args[3:] {
		if arg == "--dry-run" {
			dryRun = true
		}
		if arg == "--debug" {
			debug = true
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logPath := cfg.GetLogFile()
	logger, err := logger.New(logPath, dryRun)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	logger.SetDebugMode(debug)
	defer logger.Close()

	manager, err := sync.NewManager(cfg, logger, dryRun)
	if err != nil {
		log.Fatalf("Failed to create sync manager: %v", err)
	}

	switch command {
	case "sync":
		if err := manager.Run(); err != nil {
			log.Fatalf("Sync failed: %v", err)
		}

	case "watch":
		logger.Info("Starting watch mode")
		if err := manager.Run(); err != nil {
			log.Fatalf("Initial sync failed: %v", err)
		}

		w := watcher.New(cfg, manager, logger)
		if err := w.Start(); err != nil {
			log.Fatalf("Watch failed: %v", err)
		}

	case "clear-cache":
		if err := manager.ClearCache(); err != nil {
			log.Fatalf("Clear cache failed: %v", err)
		}
		logger.Info("Cache cleared")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
