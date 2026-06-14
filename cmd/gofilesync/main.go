package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
	"github.com/ajxudir/gofilesync/internal/progress"
	"github.com/ajxudir/gofilesync/internal/sync"
	"github.com/ajxudir/gofilesync/internal/watcher"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const usage = `gofilesync — watch and synchronize files from a YAML config

Usage:
  gofilesync <config.yaml> <command> [options]

Commands:
  sync           Run a one-time synchronization (shows initial progress)
  watch          Sync once, then watch for changes and keep syncing
  clear-cache    Remove stored backups/cache

Options:
  --dry-run      Show what would happen without changing any files
  --debug        Enable debug logging
  -q, --quiet    Suppress progress output; print only the final summary
                 (useful for CI/CD pipelines and cron jobs)
  --poll <ms>    Poll every <ms> instead of using filesystem notifications
                 (recommended for mounted volumes, e.g. Docker on macOS)
  -h, --help     Show this help and exit
  -v, --version  Show version and exit

Examples:
  gofilesync ./gofilesync.yml sync
  gofilesync ./gofilesync.yml watch --poll 200
`

func usageError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n\n", args...)
	fmt.Fprint(os.Stderr, usage)
	os.Exit(2)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usageError("a config file and command are required")
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	case "-v", "--version", "version":
		fmt.Printf("gofilesync %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if len(args) < 2 {
		usageError("a command is required (sync, watch, clear-cache)")
	}

	configPath := args[0]
	command := args[1]

	switch command {
	case "sync", "watch", "clear-cache":
	case "-h", "--help":
		fmt.Print(usage)
		return
	default:
		usageError("unknown command %q", command)
	}

	dryRun := false
	debug := false
	quiet := false
	pollMs := 0

	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--debug":
			debug = true
		case "-q", "--quiet":
			quiet = true
		case "-h", "--help":
			fmt.Print(usage)
			return
		case "--poll":
			if i+1 >= len(args) {
				usageError("--poll requires a value in milliseconds")
			}
			var err error
			pollMs, err = strconv.Atoi(args[i+1])
			if err != nil {
				usageError("invalid --poll value %q (must be a number of milliseconds)", args[i+1])
			}
			if pollMs < 0 {
				usageError("--poll must be 0 or a positive number of milliseconds")
			}
			if pollMs > 0 && pollMs < 100 {
				fmt.Fprintf(os.Stderr, "Warning: a very short poll interval (%dms) may cause high CPU usage\n", pollMs)
			}
			i++
		default:
			usageError("unknown option %q", arg)
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fatal("%v", err)
	}

	logPath := cfg.GetLogFile()
	logger, err := logger.New(logPath, dryRun)
	if err != nil {
		fatal("cannot create log file: %v", err)
	}
	logger.SetDebugMode(debug)
	defer logger.Close()

	manager, err := sync.NewManager(cfg, logger, dryRun)
	if err != nil {
		fatal("cannot initialize: %v", err)
	}

	switch command {
	case "sync":
		if err := manager.Init(os.Stderr, quiet); err != nil {
			fatal("sync failed: %v", err)
		}

	case "watch":
		if err := manager.Init(os.Stderr, quiet); err != nil {
			logger.Warn("Initial sync had problems: " + err.Error())
		}

		mode := "filesystem notifications"
		if pollMs > 0 {
			mode = fmt.Sprintf("polling every %dms", pollMs)
		}
		progress.Banner(os.Stderr, "Watching for changes", mode)

		w := watcher.New(cfg, manager, logger)
		w.SetPollInterval(pollMs)
		if err := w.Start(); err != nil {
			fatal("watch failed: %v", err)
		}

	case "clear-cache":
		if err := manager.ClearCache(); err != nil {
			fatal("clear-cache failed: %v", err)
		}
		logger.Info("Cache cleared")
	}
}
