package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	"github.com/ajxudir/gofilesync/internal/sync"
	"github.com/ajxudir/gofilesync/internal/watcher"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: gofilesync <config.yaml> <command> [options]")
		fmt.Println("Commands: sync, watch, clear-cache")
		fmt.Println("Options: --dry-run, --debug, --poll <ms>")
		os.Exit(1)
	}

	configPath := os.Args[1]
	command := os.Args[2]
	dryRun := false
	debug := false
	pollMs := 0

	for i := 3; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--debug":
			debug = true
		case "--poll":
			if i+1 < len(os.Args) {
				var err error
				pollMs, err = strconv.Atoi(os.Args[i+1])
				if err != nil {
					log.Fatalf("Invalid poll value: %s", os.Args[i+1])
				}
				if pollMs < 0 {
					log.Fatalf("Poll interval must be 0 or positive")
				}
				if pollMs > 0 && pollMs < 100 {
					log.Printf("Warning: Poll interval is very short (%dms), this may cause high CPU usage", pollMs)
				}
				i++
			} else {
				log.Fatalf("--poll requires a value in milliseconds")
			}
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
		if pollMs > 0 {
			logger.Info(fmt.Sprintf("Starting watch mode with polling (interval: %dms)", pollMs))
		} else {
			logger.Info("Starting watch mode")
		}

		if err := manager.Run(); err != nil {
			log.Fatalf("Initial sync failed: %v", err)
		}

		w := watcher.New(cfg, manager, logger)
		w.SetPollInterval(pollMs)
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
