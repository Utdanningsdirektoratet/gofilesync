# GoFileSync

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=for-the-badge)](LICENSE)

A lightweight, cross-platform file synchronization tool with backup management,
watch mode, ignore patterns, and dry-run. Built for Docker containers and
development workflows.

## Features

- 🔄 Sync multiple file/directory pairs (one-way: source → destination)
- 👁️ Watch mode: fsnotify or polling (reliable on Docker/macOS mounts)
- 🧹 Ignore patterns (gitignore-style globs), global and per-entry
- 🥇 Priority overrides: later entries win for a shared destination, with no churn
- ⏱️ `once` entries: synced at startup only, ignored while watching
- ⚡ Fast: concurrent copies + size/mtime change detection (no content reads when
  unchanged); `checksum` opt-in for content-based comparison
- ♻️ Auto-excludes its own state dir, so it never syncs/watches its backups & log
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
workers: 8                        # concurrent copies (default: 8)
checksum: false                   # true = SHA-256 compare; false = size+mtime

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
  - title: User data
    source: /app/user
    destination: /var/www/html/user
    once: true                    # sync at startup only; ignored while watching
```

Ignore glob syntax: `*` (within a segment), `?` (one char), `**` (any number of
directories), a pattern without `/` matches by basename at any depth (e.g.
`node_modules`), and a leading `!` re-includes a previously excluded path.

An entry with `once: true` is synced during the initial sync (and the `sync`
command) but is **not** watched — changes to its source while watching are
ignored, and its destination is left in place.

## Priority & overrides

When two or more entries write to the **same destination**, the entry listed
**lower in the config wins** (lower = higher priority). For each shared file
gofilesync picks a single owner — the lowest entry that has it — so the file is
written once (no churn), not rewritten by every entry.

This makes a common layout easy: sync a whole vendor/plugin tree, then layer a
project "overrides" directory **below** it that contains only your changed files.

```yaml
sync:
  - title: Vendor plugin            # base
    source: /vendor/restful
    destination: /var/www/html/webservice/restful
  - title: Project overrides        # lower = wins for any shared file
    source: /src/restful
    destination: /var/www/html/webservice/restful
```

Drop a changed file into the overrides dir and it takes over with no other
config change; delete it and the destination reverts to the vendor version.

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

  [2026-06-14 06:43:51]  INFO   Sync complete — 2,641 synced
```

(The start and completion timestamps make the elapsed sync time visible.)

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
