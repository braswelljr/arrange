# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Overview

**arrange** is a cross-platform CLI tool (`github.com/braswelljr/arrange`) that scans a directory and moves files into sub-folders organised by type — Documents, Images, Videos, Audio, etc. It supports one-shot runs, recursive scans, filesystem watch mode, and OS-level daemon/service management.

## Project Structure

```
arrange/
  cmd/                    # Cobra command definitions (one file per sub-command)
    root.go               # Root command + CmdOptions; wires all sub-commands
    run.go                # `arrange run` — one-shot organise pass
    watch.go              # `arrange watch` — fsnotify-based continuous watcher
    services.go           # `arrange service` — daemon management (non-Windows)
    services_windows.go   # Windows stub for the service command
    media.go              # `arrange media` — media-only organise sub-command
    setup.go              # `arrange setup` — interactive config setup
    version.go            # `arrange version`
  internal/
    common/               # App-wide constants: AppName, Version
    config/               # Config loading, file-type definitions, path resolution
    fileops/              # Filesystem helpers: scan, walk, move, copy, safe paths
    logger/               # Colored, terminal-aware, goroutine-safe logger
    media/                # Media filename parser and organiser
  main.go                 # Entry point — builds CmdOptions and calls NewRootCmd
  Makefile
  .golangci.yml
  .github/workflows/
    test.yml
    lint.yml
    build.yml
```

## Development Commands

```bash
# Build
make build/darwin         # macOS binary → bin/arrange
make build/linux          # Linux amd64  → bin/arrange-linux
make build/windows        # Windows amd64 → bin/arrange.exe
make build/all            # All three

make install              # go install with version ldflags

# Run directly
make run                  # go run main.go

# Dependencies
make tidy                 # go mod tidy
make download             # go mod download

# Quality
make test                 # go test -race ./...
make vet                  # go vet ./...
make lint                 # golangci-lint run
make fix                  # gofmt -s -w . && goimports -w ./...

# Clean
make clean                # remove bin/
```

The binary version string is injected at link time:

```
-ldflags "-X github.com/braswelljr/arrange/internal/common.Version=<tag>"
```

## Tech Stack

- **Go 1.25** — module path `github.com/braswelljr/arrange`
- **Cobra** (`github.com/spf13/cobra`) — CLI framework
- **fsnotify** (`github.com/fsnotify/fsnotify`) — cross-platform filesystem events for `watch`
- **takama/daemon** (`github.com/takama/daemon`) — OS daemon/service management (non-Windows)
- **fatih/color** (`github.com/fatih/color`) — colored terminal output
- **golang.org/x/term** — terminal width detection

## CLI Commands

| Command | Description |
|---|---|
| `arrange run <src> [dest]` | One-shot scan; moves files into typed sub-folders |
| `arrange run -r <src>` | Recursive scan; skips already-organised folders |
| `arrange watch <dir>` | Watch mode; organises on `CREATE` events (debounced 800ms) |
| `arrange media <src> [dest]` | Media-only organise (video + audio smart-parsing) |
| `arrange setup` | Interactive config setup wizard |
| `arrange service install <dir>` | Install as OS daemon watching `<dir>` |
| `arrange service start/stop/status/uninstall` | Manage the daemon |
| `arrange version` | Print build version |

Global flag: `--config-path / -C <path>` overrides the default config file location.

## Package Architecture

### `internal/config`

- `Config` struct — JSON-serialised; fields: `UnknownFilesFolder`, `KnownFiles []FileExt`, `ExcludedDirs []string`, `MediaCreators []string`
- `NewConfig(path)` — singleton per path (RW-mutex cached); creates a default config on first run
- `Config.Get(ext)` → `(folder, exempt bool)` — single-extension lookup; exempt extensions are skipped entirely
- `Config.IsExcludedPath(path)` — checks every path component against `ExcludedDirs` (case-insensitive)
- `Config.DestFolders()` — set of all destination folder names; used by `run -r` to skip already-organised subdirs
- `Config.ExtSet(folder)` — all extensions belonging to a named folder (used to build the media set)
- Platform-specific config path: `path_unix.go` / `path_windows.go`
- Folder name constants live in `file_types.go` (`FolderVideos`, `FolderAudio`, `FolderOther`, etc.)

### `internal/fileops`

- `SmartFile` / `SmartFiles` — enriched file descriptor: `Name`, `Path`, `Ext`, `Size`
- `ScanDir(dir)` — flat scan, returns only files
- `WalkDir(dir, skipFn)` — recursive walk; `skipFn(name) bool` prunes directories
- `Move(src, dst)` — atomic rename with cross-device copy-then-delete fallback (`move.go` / `move_windows.go`)
- `SafeDestPath(dir, stem, ext)` — collision-safe destination path (`file (1).ext`, `file (2).ext`, …)
- `EnsureDir(path)` — `os.MkdirAll` with 0750
- `StripBrowserSuffix(stem)` — removes browser duplicate suffixes like ` (1)`
- `copy.go` — used by Move for cross-device fallback

### `internal/media`

- `MediaInfo` — parsed representation: `Type`, `Title`, `Year`, `Season`, `Episode`, `EpisodeEnd`, `Part`, `Quality`, `OrigPath`, `Ext`
- `MediaType` constants: `TypeUnknown`, `TypeTVSeries`, `TypeMovie`, `TypeMoviePart`
- `ParseName(filename) *MediaInfo` — extracts all metadata from a bare filename using compiled regexes; handles scene, P2P, streaming, anime naming conventions
- `Parse(path) *MediaInfo` — wraps `ParseName` and sets `OrigPath`
- `MediaInfo.DestDir(creator)` — relative destination path (`Title (YYYY)/` or `Title/Season 01/`)
- `MediaInfo.DestName()` — cleaned filesystem-safe filename with quality tag
- `MediaInfo.CreatorMatch(creators)` — matches possessive creator prefixes (e.g. `"Tyler Perry's …"`)
- `SanitizeFilename(s)` — removes/replaces characters illegal on Windows, macOS, and Linux
- `Organise(srcDir, destDir, cfg, moveFn)` — injectable-move organiser used by tests and the `media` sub-command

### `internal/logger`

- `Logger` — goroutine-safe (`sync.Mutex`), writes to separate `out`/`err` writers
- Methods: `Info`, `Infof`, `Success`, `Successf`, `Warn`, `Warnf`, `Error`, `Errorf`, `Event`, `Eventf`, `Move`, `Header`, `Footer`, `Separator`
- `Move(src, dst)` — truncates paths to fit terminal width with a `→` arrow
- `TermWidth()` — reads terminal columns via `x/term`, falls back to 80

### `internal/common`

- `AppName = "arrange"` (string constant)
- `Version` (var, injected at link time via ldflags)

## Key Implementation Patterns

### Worker Pool in `run`

`cmd/run.go` fans out one goroutine per logical CPU (capped to `len(work)`). A `dirLocker` (`sync.Map` of `*sync.Mutex`) serialises `SafeDestPath + Move` per destination directory so concurrent workers don't race on collision-suffix generation.

```go
locker := new(dirLocker)
numWorkers := runtime.NumCPU()
if numWorkers > len(work) {
    numWorkers = len(work)
}
jobs := make(chan *fileops.SmartFile)
errCh := make(chan error, len(work))
// fan out workers, drain errCh, return errors.Join(...)
```

### Debounce in `watch`

`cmd/watch.go` uses a `map[string]*time.Timer` (guarded by its own mutex) to batch rapid filesystem events — archive extractions, multi-file copies — into one organise pass per source directory, 800ms after the last `CREATE` event.

### Injectable Move for Tests

`media.Organise` accepts a `move func(src, dst string) error` parameter. Tests pass an in-memory recorder instead of `fileops.Move`, so the organiser logic is tested without touching the filesystem.

### Config Singleton

`config.NewConfig` returns the cached `*Config` for a given path without re-reading the file. Callers must not mutate the returned struct.

### Platform-specific Code

| File | Build tag | Purpose |
|---|---|---|
| `cmd/services.go` | `//go:build !windows` | Real daemon management via `takama/daemon` |
| `cmd/services_windows.go` | (implicit) | Stub — not supported on Windows |
| `internal/fileops/move.go` | `//go:build !windows` | `rename` → copy+remove fallback |
| `internal/fileops/move_windows.go` | (implicit) | Windows-specific move |
| `internal/config/path_unix.go` | `//go:build !windows` | XDG / macOS config path |
| `internal/config/path_windows.go` | (implicit) | `%APPDATA%` config path |

## Code Style

- **No unnecessary comments** — only add a comment when the _why_ is non-obvious.
- **Error wrapping** — always `fmt.Errorf("context: %w", err)` to preserve the chain.
- **Error joining** — `errors.Join(errs...)` for collecting multiple worker errors.
- **`sync.Map`** for per-key locks (see `dirLocker`); plain `sync.Mutex` for everything else.
- **`sync.Once`** for lazy initialisation inside long-lived structs (see `Config.initCfgMap`).

## Linting

Config: `.golangci.yml`

Enabled extras: `misspell`, `unconvert`, `revive`, `gosimple`, `gocritic`

Disabled (legitimate in this codebase): `funlen`, `cyclop`, `maintidx`, `gocognit` — the media parser and `organiseFile` use intentional `switch`/`if` chains that would false-positive on complexity checks.

Key `errcheck` exclusions: `*.Close()` returns, `os.Remove` in cleanup paths.

## Testing

- **Race detector always on** in CI: `go test -race ./...`
- Test files follow standard `_test.go` naming with `package foo_test` for black-box tests.
- `media.Organise` uses an injectable `moveFn` — pass a recording stub to test organiser logic without filesystem I/O.
- `cmd/run_test.go` exercises the full run pipeline on a temp directory.
- `golangci-lint` skips `gocritic` and `revive` for `_test.go` files.

## CI

Three independent workflow files under `.github/workflows/`:

| File | Trigger | What it does |
|---|---|---|
| `test.yml` | push/PR → main | `go vet` + `go test -race` on ubuntu, macOS, windows; uploads `coverage.out` artifact from the Linux runner |
| `lint.yml` | push/PR → main | `golangci-lint run --timeout=5m` on ubuntu |
| `build.yml` | push/PR → main | Cross-compile for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64; uploads each binary as an artifact |

Go version is read from `go.mod` via `go-version-file: go.mod` — no manual version pin in CI.

## Important Files

- `main.go` — entry point; constructs `CmdOptions` (logger, stdio) and calls `NewRootCmd`
- `internal/config/config.go` — `Config` struct, `NewConfig`, all lookup methods
- `internal/config/file_types.go` — folder name constants and default `KnownFiles` / `ExcludedDirs`
- `internal/media/parser.go` — all compiled regexes and `ParseName` logic
- `internal/media/info.go` — `MediaInfo`, `DestDir`, `DestName`, `CreatorMatch`
- `internal/fileops/move.go` — atomic move with cross-device fallback
- `internal/fileops/file.go` — `ScanDir`, `WalkDir`, `SmartFile`
- `cmd/run.go` — worker pool, `organiseFile`, `dirLocker`
- `cmd/watch.go` — fsnotify event loop and debounce logic
- `.golangci.yml` — linter configuration
