# GoFileSync

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=for-the-badge)](LICENSE)

A lightweight, cross-platform file synchronization tool with backup management,
watch mode, ignore patterns, and dry-run. Built for Docker containers and
development workflows.

## Features

- 🔄 Sync multiple file/directory pairs (one-way: source → destination)
- 👁️ Watch mode: fsnotify or polling (reliable on Docker/macOS mounts)
- 🧹 Ignore patterns (gitignore-style globs), global and per-entry
- 💾 Automatic backup before overwriting; restore on source deletion
- 🗑️ Deletion propagation for managed files (never touches unmanaged files)
- 🧪 Dry-run mode; `--quiet` for CI/CD pipelines
- 🔗 Resolves symlinks (copies content, doesn't create links)
- 🔐 Opt-in permission preservation
- 🖥️ Cross-platform: Linux, macOS, Windows

## Installation

```bash
# From source
go install github.com/ajxudir/gofilesync/cmd/gofilesync@latest

# Or build locally
go build -o gofilesync ./cmd/gofilesync
```

Pre-compiled binaries are on the [Releases](https://github.com/ajxudir/gofilesync/releases) page.

## Usage

```bash
gofilesync <config.yaml> <command> [options]
```

Commands:

```bash
gofilesync config.yaml sync                  # one-time sync
gofilesync config.yaml watch                 # sync once, then watch (fsnotify)
gofilesync config.yaml watch --poll 300      # watch via polling (Docker/macOS)
gofilesync config.yaml clear-cache           # remove stored backups
```

Options: `--dry-run`, `--debug`, `-q/--quiet`, `--poll <ms>`, `-h/--help`,
`-v/--version`.

Exit codes: `0` success · `1` runtime error (e.g. a file failed to sync) ·
`2` usage error. Status/progress/logs go to **stderr**; stdout stays clean for
piping.

## Configuration

```yaml
configDir: /var/www/.gofilesync   # state dir (default: ./.gofilesync)
debounceInMs: 200                 # watch debounce (default: 300)
preservePermissions: false        # opt-in: keep source file mode

ignore:                           # gitignore-style globs, applied to every entry
  - "**/.git/**"
  - "node_modules"
  - "*.log"

sync:                             # required
  - title: Config                 # name shown in output
    source: /app/config.php       # absolute path to a file or directory
    destination: /var/www/config.php
  - title: Auth
    source: /app/auth
    destination: /var/www/auth
    ignore:                       # optional per-entry globs (merged with global)
      - "tests/**"
```

Ignore glob syntax: `*` (within a segment), `?` (one char), `**` (any number of
directories), a pattern without `/` matches by basename at any depth (e.g.
`node_modules`), and a leading `!` re-includes a previously excluded path.

## Output

The initial sync and watch mode share a consistent, bounded layout (colored on a
terminal, plain in pipes/CI/log files):

```
──────────────────────────────────────────────────
gofilesync · Initial sync
2,641 files · 9 entries · ignoring 3 patterns
──────────────────────────────────────────────────

  [2026-06-14 06:43:38]  INFO
    ▸ Config  (1 file)
    ▸ Local plugins  (264 files)

  [100%] Initialization complete — 2,641 synced
```

In watch mode each change batch is one concise line per entry (a single file
shows its path, many collapse to a count, so a folder delete never floods):

```
  [2026-06-14 06:46:26]  INFO
    ▸ Auth plugins  (changed plugin5/lib.php)
    ▸ Filter plugins  (308 removed)
```

## State management

GoFileSync stores state under `configDir`:

- `.backups/` — backups taken before overwriting an existing destination
- `managed.json` — files the tool has written (basis for safe deletion handling)
- `sync.log` — operation log

## Building & testing

```bash
make build          # build ./bin/gofilesync
make test           # end-to-end test suite
make test-features  # per-feature tests
make test-projects  # project-setup tests
```

Tests are end-to-end: they compile the real binary and drive it as a subprocess
against real configs and files (`test/features/` per-feature, `test/` project
setups).

## Releasing

```bash
export GITHUB_TOKEN=$(gh auth token)
git tag -a 1.2.0 -m "Release 1.2.0" && git push origin main 1.2.0
goreleaser release --clean
```

## License

MIT License — see [LICENSE](LICENSE).

## Acknowledgments

- File watching by [fsnotify](https://github.com/fsnotify/fsnotify)
- YAML parsing by [yaml.v3](https://gopkg.in/yaml.v3)
