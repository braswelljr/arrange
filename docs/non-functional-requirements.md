# Non-Functional Requirements

**Project:** arrange  
**Version:** 1.0  
**Date:** 2026-06-30

This document specifies the quality attributes, constraints, and operational characteristics that govern `arrange` across all features.

---

## 1. Performance

### NFR-PERF-01 — Concurrent file processing

**Requirement:** `arrange run` shall process files concurrently using a worker pool of `runtime.NumCPU()` goroutines (capped to the number of files), so CPU-intensive operations (media parsing, directory creation) and I/O-bound operations (move syscalls) overlap.

**Rationale:** A sequential loop on a directory of 500 files with complex media parsing would be 4–8× slower than a parallel worker pool on a quad-core machine.

**Target:** A directory of 500 mixed files shall be fully organised in under 5 seconds on a machine with ≥ 4 logical cores and an SSD.

---

### NFR-PERF-02 — Regex compilation cost

**Requirement:** All regular expressions in `internal/media/parser.go` shall be compiled once at package initialisation via `regexp.MustCompile` at package level. No regex shall be compiled inside a function that is called per-file.

**Rationale:** `regexp.Compile` allocates; doing so 500 times per run for a 500-file directory wastes CPU and memory.

---

### NFR-PERF-03 — Watch debounce latency

**Requirement:** After the last filesystem event in a burst, the organise pass shall begin within 800 ms ± 50 ms.

**Target:** Single file drop → organise begins within 850 ms.

---

### NFR-PERF-04 — Memory footprint

**Requirement:** Peak memory usage for a 1 000-file directory shall not exceed 64 MiB resident set size.

**Rationale:** `arrange` is intended for always-on service use; excessive memory usage on a NAS or small server is unacceptable.

---

## 2. Reliability

### NFR-REL-01 — No data loss on move failure

**Requirement:** If a file move fails (permission denied, out of disk space, network interruption on a network drive), the source file shall remain intact and in its original location. `arrange` shall never delete a source file unless the copy to the destination has been verified.

**Implementation:** `fileops.Move` uses `os.Rename` (atomic on POSIX) for same-device moves. For cross-device moves, it copies first, verifies success, and only then removes the source.

---

### NFR-REL-02 — In-progress download safety

**Requirement:** Files with an exempt extension shall never be moved, renamed, read for content, or stat-ed beyond the filesystem entry check required to skip them. This guarantee shall hold regardless of command, flag combination, or config.

**Acceptance test:** Create a `.crdownload` file. Run `arrange run --recursive <dir>`. Verify the file is untouched.

---

### NFR-REL-03 — Idempotency

**Requirement:** Running `arrange run <dir>` twice on the same directory shall produce the same result. The second run shall move zero files when the first run completed successfully (assuming no new files were added between runs and `src == dest`).

**Acceptance test:** Run `arrange run --recursive <dir>`. Immediately run it again. Verify the second run logs "No files found" and exits cleanly.

---

### NFR-REL-04 — Error isolation

**Requirement:** A failure moving one file shall not prevent `arrange` from processing the remaining files in the same run. All errors shall be collected and returned as a joined error at the end.

**Acceptance test:** Make one file in a directory read-only (chmod 000). Run `arrange run <dir>`. Verify other files are moved and the read-only file's error appears in the output.

---

### NFR-REL-05 — No file overwrites

**Requirement:** `arrange` shall never silently overwrite an existing file at the destination. When a name collision occurs, a versioned suffix (`-v1`, `-v2`, …) shall be appended to produce a unique path before moving.

---

## 3. Concurrency Safety

### NFR-CONC-01 — Race-free operation

**Requirement:** The entire codebase shall pass `go test -race ./...` with zero detected data races.

**Scope:** Worker pool goroutines, `dirLocker`, `Logger`, `config.NewConfig` singleton, filesystem watcher goroutine.

---

### NFR-CONC-02 — Per-directory move atomicity

**Requirement:** For any given destination directory, the `SafeDestPath` lookup and the subsequent `Move` call shall execute under the same mutex so no two workers can claim the same free destination path simultaneously.

---

### NFR-CONC-03 — Logger goroutine safety

**Requirement:** All `Logger` methods shall be safe to call from multiple goroutines simultaneously. The internal `sync.Mutex` shall protect every write to the underlying `io.Writer`.

---

## 4. Correctness

### NFR-CORR-01 — Test coverage minimums

**Requirement:** The following minimum coverage levels shall be maintained in CI:

| Package | Minimum Coverage |
|---------|-----------------|
| `internal/logger` | 95% |
| `internal/media` | 85% |
| `internal/config` | 80% |
| `internal/fileops` | 75% |
| `cmd` (`runE`, `organiseFile`) | 85% |

---

### NFR-CORR-02 — Media parser accuracy

**Requirement:** The parser shall correctly extract title, season, episode, year, and quality for ≥ 95% of filenames conforming to common scene and P2P conventions, as validated by the test suite in `internal/media/parser_test.go`.

---

### NFR-CORR-03 — Extension normalisation consistency

**Requirement:** All file extensions used as map keys or returned from `cfg.Get` shall be lowercase and without a leading dot. Any code path that introduces an extension string shall normalise it using `strings.ToLower(strings.TrimLeft(filepath.Ext(name), "."))`.

---

## 5. Usability

### NFR-USE-01 — Zero-configuration first run

**Requirement:** A user who installs `arrange` and runs `arrange run ~/Downloads` without reading the documentation shall see their files organised into sensible folders. The default configuration shall cover the 200+ most common file extensions.

---

### NFR-USE-02 — Informative terminal output

**Requirement:** Every move shall produce a coloured log line showing `src → dst`. The total terminal width shall be respected; paths shall be truncated with `…` rather than wrapping.

---

### NFR-USE-03 — Helpful CLI help text

**Requirement:** `arrange --help`, `arrange run --help`, and all subcommand `--help` outputs shall describe the command's purpose, all arguments, and all flags.

---

### NFR-USE-04 — Config discoverability

**Requirement:** On first run, `arrange` shall print the path of the created config file so the user knows where to find and edit it.

---

## 6. Portability

### NFR-PORT-01 — Target platforms

**Requirement:** `arrange` shall build and run correctly on:

| Platform | Architecture | Notes |
|----------|-------------|-------|
| macOS 13+ | arm64, amd64 | LaunchAgent service support |
| Linux (glibc 2.17+) | amd64, arm64 | systemd user unit support |
| Windows 10+ | amd64 | `service` subcommand not available |

---

### NFR-PORT-02 — Build reproducibility

**Requirement:** The build shall be reproducible from source using `make build`. No platform-specific toolchain beyond the Go compiler shall be required.

---

### NFR-PORT-03 — Platform-specific code isolation

**Requirement:** All platform-specific code shall be isolated in files with build tags (`//go:build !windows` / `//go:build windows`). No `runtime.GOOS` branch shall appear outside those files except in the service `NewService` function.

---

### NFR-PORT-04 — Config path convention

**Requirement:** The default config path shall follow platform conventions:

| Platform | Path |
|----------|------|
| macOS / Linux | `$XDG_CONFIG_HOME/arrange/config.json` (default: `~/.config/arrange/config.json`) |
| Windows | `%APPDATA%\arrange\config.json` |

---

### NFR-PORT-05 — Filename safety

**Requirement:** All output filenames produced by `SanitizeFilename` shall be valid on Windows, macOS, and Linux simultaneously. Characters illegal on Windows (`:`, `*`, `?`, `"`, `<`, `>`, `|`, `\`) shall be replaced or removed.

---

## 7. Maintainability

### NFR-MAINT-01 — Single responsibility per package

**Requirement:** Each internal package shall have a single, clearly-stated responsibility:
- `config` — configuration loading and extension lookup only.
- `fileops` — filesystem operations only (no business logic).
- `media` — filename parsing and media organisation only.
- `logger` — terminal output only.
- `cmd` — command wiring and orchestration only.

No package shall import another in a way that creates a cycle.

---

### NFR-MAINT-02 — Extension table centralisation

**Requirement:** The extension-to-folder mapping shall live exclusively in `internal/config/file_types.go`. No extension string shall be hard-coded in `cmd/`, `internal/fileops/`, or `internal/media/`.

**Rationale:** Centralisation ensures that adding a new extension requires changing one file.

---

### NFR-MAINT-03 — Folder constant usage

**Requirement:** All code that compares or constructs folder names shall use the constants defined in `internal/config/file_types.go` (`config.FolderVideos`, `config.FolderDocuments`, etc.) instead of raw strings.

---

### NFR-MAINT-04 — Skill-governed development

**Requirement:** Changes to the following areas of the codebase shall invoke the corresponding skill before implementation (see `AGENTS.md` for the full skill table):

| Area | Skill |
|------|-------|
| Goroutines, channels, sync primitives | `golang-concurrency` |
| Error creation or accumulation | `golang-error-handling` |
| New exported symbols or names | `golang-naming` |
| Test files | `golang-testing` |

---

## 8. Security

### NFR-SEC-01 — Scope limitation

**Requirement:** `arrange` shall only read from and write to directories explicitly provided by the user as arguments. It shall not read environment variables beyond `XDG_CONFIG_HOME` and `APPDATA`, and shall not make network requests.

---

### NFR-SEC-02 — Config file permissions

**Requirement:** The config file shall be created with permissions `0600` (owner read/write only). Config directories shall be created with permissions `0750`.

---

### NFR-SEC-03 — No execution of file content

**Requirement:** `arrange` shall never execute, interpret, or evaluate the content of any file it organises. It reads only the filename (and file size to skip zero-byte files). No file content is read.

---

## 9. Observability

### NFR-OBS-01 — Structured log levels

**Requirement:** Log output shall use distinct, visually differentiated levels: `INFO`, `OK`, `WARN`, `ERR`, `MOVE`, `EVENT`. Each level shall have a distinct colour in terminal output.

---

### NFR-OBS-02 — Non-TTY compatibility

**Requirement:** When stdout is not a TTY (piped or redirected), colour codes shall be suppressed and terminal-width-dependent truncation shall fall back to 80 columns.

---

### NFR-OBS-03 — Exit codes

**Requirement:**

| Condition | Exit code |
|-----------|-----------|
| All files organised successfully | 0 |
| No files found (empty or all exempt) | 0 |
| One or more file move failures | 1 |
| Fatal error (config load, scan failure) | 1 |

---

## 10. Testability

### NFR-TEST-01 — Filesystem isolation

**Requirement:** All tests that touch the filesystem shall use `t.TempDir()` for their working directories so cleanup is automatic and tests do not interfere with each other.

---

### NFR-TEST-02 — Config isolation in tests

**Requirement:** Every test that calls `config.NewConfig` shall pass a path inside `t.TempDir()` to bypass the package-level singleton cache and get a fresh default config.

---

### NFR-TEST-03 — Race detector in CI

**Requirement:** The CI pipeline shall run `go test -race ./...`. Any race condition detected shall be treated as a build failure.

**CI command:** `make test` (defined as `go test -race ./...`).

---

### NFR-TEST-04 — No test-only globals

**Requirement:** Tests shall not modify package-level variables. All test-specific state shall be local to the test function or passed through parameters.
