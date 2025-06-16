# GoFileSync

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=for-the-badge)](LICENSE)

A lightweight, cross-platform file synchronization tool with backup management, watch mode, and dry-run capabilities. Perfect for Docker containers and development workflows.

## Features

- 🔄 Sync multiple file/directory pairs
- 💾 Automatic backup before overwriting  
- 👁️ Watch mode with configurable debouncing
- 🧪 Dry run mode for testing
- 🔗 Follows symlinks (doesn't create them)
- 📁 Per-directory sync configuration
- 📝 Detailed logging with debug mode

## How It Works

1. **Sync Mode**: Compares source and destination files, creates backups of existing files, then copies changed files
2. **Watch Mode**: Monitors source directories for changes, debounces events, then runs sync
3. **Backup System**: Creates unique backup paths using source file path hashing
4. **Symlinks**: Automatically resolves and follows symlinks, copying actual file content

## Requirements

- Go 1.21+ (for building from source)
- Works on Linux, macOS, and Windows
- No external dependencies for running compiled binary

## Installation

### Binary Download
Download pre-compiled binaries from [Releases](https://github.com/ajxudir/gofilesync/releases)

### From Source
```bash
go install github.com/ajxudir/gofilesync/cmd/gofilesync@latest
```

### Install Script
```bash
curl -sSL https://raw.githubusercontent.com/ajxudir/gofilesync/main/install.sh | sh
```

### Development

```bash
go build -o gofilesync ./cmd/gofilesync
```

## Usage

```bash
# Sync once
gofilesync config.yaml sync

# Watch mode
gofilesync config.yaml watch

# Dry run
gofilesync config.yaml sync --dry-run

# Debug mode (shows all log messages)
gofilesync config.yaml sync --debug

# Clear backups
gofilesync config.yaml clear-cache
```

## Configuration

Create a YAML file:

```yaml
configDir: /var/www/.gofilesync
debounceInMs: 200
directories:
  - title: Config
    source: /app/source/config.php
    destination: /var/www/html/config.php
  - title: Authentication Plugins
    source: /app/source/auth
    destination: /var/www/html/auth
  - title: Themes
    source: /app/source/theme
    destination: /var/www/html/theme
```

### Configuration Details

- `configDir`: Where to store state files (defaults to `{current_directory}/.gofilesync`)
- `debounceInMs`: Milliseconds to wait before syncing after file changes
- `directories`: Array of sync pairs (required)
  - `title`: Descriptive name for logging
  - `source`: Absolute path to source file/directory
  - `destination`: Absolute path to destination

## State Management

GoFileSync stores state in `configDir`:
- `.backups/` - File backups before overwriting
- `sync.log` - Operation logs

Backup paths are unique per source file to prevent conflicts.

## Log Format

```
[2025-06-16 11:02:48]  level=INFO   Processing Config
[2025-06-16 11:02:48]  level=INFO   Backup original /var/www/config.php -> /var/www/.gofilesync/.backups/...
[2025-06-16 11:02:48]  level=INFO   Sync /app/config.php -> /var/www/config.php
[2025-06-16 11:02:48]  level=DEBUG  Unchanged /var/www/config.php -> /var/www/config.php
```

## Project Structure

```
gofilesync/
├── cmd/
│   └── gofilesync/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go       
│   ├── sync/
│   │   ├── manager.go
│   │   ├── manager_test.go      
│   │   ├── file.go
│   │   ├── file_test.go         
│   │   ├── backup.go
│   │   └── backup_test.go       
│   ├── watcher/
│   │   ├── watcher.go
│   │   └── watcher_test.go      
│   └── logger/
│       ├── logger.go
│       └── logger_test.go       
├── go.mod                       
├── Dockerfile                   
├── config.yaml
└── README.md                    
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test -v ./internal/sync/...
```

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- Built with [fsnotify](https://github.com/fsnotify/fsnotify) for file watching
- YAML parsing by [yaml.v3](https://gopkg.in/yaml.v3)