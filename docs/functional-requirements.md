# Functional Requirements

**Project:** arrange  
**Version:** 1.0  
**Date:** 2026-06-30

This document lists every functional requirement for the `arrange` CLI, grouped by feature area. Each requirement has a unique ID, a user story, the expected behaviour, and any edge cases or acceptance criteria.

---

## 1. File Scanning

### FR-SCAN-01 — Flat directory scan

**Story:** As a user, I want `arrange run <dir>` to scan the top level of a directory so I can organise all the files without touching subdirectories.

**Behaviour:**
- Read the directory entries of `<dir>`.
- Include regular files only; skip directories, symlinks, and device files.
- Normalise each file's extension to lowercase without a leading dot.

**Acceptance criteria:**
- `ScanDir` returns `SmartFile` entries for all regular files in the directory.
- Subdirectories do not appear in the result.
- Extensions are lowercase and dot-free (`"mp4"`, not `".MP4"`).

---

### FR-SCAN-02 — Recursive directory scan

**Story:** As a user, I want `arrange run --recursive <dir>` to scan all subdirectories so I can organise nested content in one pass.

**Behaviour:**
- Walk all subdirectories of `<dir>` via `filepath.WalkDir`.
- Invoke the `skipDir` callback before descending into any directory; skip it if the callback returns true.
- Collect regular files from all non-skipped directories.

**Acceptance criteria:**
- Files in nested subdirectories are included in the result.
- Directories in the skip set are not descended into and none of their files appear in the result.
- All returned `SmartFile.Path` values are absolute.

---

### FR-SCAN-03 — In-place recursive skip

**Story:** As a user, when I run `arrange run --recursive <dir>` on a directory I already organised (src == dest), I want already-organised category folders to be skipped so files are not moved again.

**Behaviour:**
- When `srcDir == destDir`, build the skip set from `cfg.DestFolders()`.
- Pass this skip set to `WalkDir` so any subdirectory whose name (case-insensitive) matches a category folder is not descended into.

**Acceptance criteria:**
- Running `arrange run --recursive <dir>` twice on the same directory produces zero moves on the second run.
- Files inside `Videos/`, `Documents/`, etc. remain untouched.

---

## 2. File Categorisation

### FR-CAT-01 — Extension-to-folder mapping

**Story:** As a user, I want each file to land in the folder that matches its extension so I can find it by type.

**Behaviour:**
- Look up the file's extension in the config map.
- Return the configured `folder` and whether the extension is `exempt`.
- If no entry matches, return `unknown_files_folder` (default `"Other"`), non-exempt.

**Acceptance criteria:**
- `cfg.Get("mp4")` returns `("Videos", false)`.
- `cfg.Get("go")` returns `("Code", true)`.
- `cfg.Get("xyz123")` returns `("Other", false)`.

---

### FR-CAT-02 — Exempt extension enforcement

**Story:** As a user, I never want `arrange` to touch files that are being downloaded or that are source code, because moving them would corrupt the download or break my project.

**Behaviour:**
- When `cfg.Get(ext)` returns `exempt == true`, skip the file entirely.
- Do not log, do not move, do not create a destination directory.

**Accepted exempt groups:**
- In-progress downloads: `crdownload`, `part`, `download`, `opdownload`, `!qb`, `!ut`, `!bt`, `aria2`, `fdm`, `jdownload`, `idm`, `idt`, `ytdl`, `axel`, `dlpart`, `downloading`, `partial`, `incomplete`, `unfinished`, `DS_Store`.
- Torrent meta: `torrent`, `magnet`.
- Source code: `go`, `py`, `js`, `ts`, `rs`, `cpp`, `c`, `java`, `html`, `css`, `json`, `yaml`, `sh`, and all other source/config extensions listed in the default config.

**Acceptance criteria:**
- A file named `video.crdownload` is never moved regardless of `--recursive` or any other flag.
- A file named `main.go` is never moved.
- Running `arrange run` with only exempt files in the directory exits with no moves and no error.

---

### FR-CAT-03 — Zero-byte file skip

**Story:** As a user, I want zero-byte files to be ignored because they are likely incomplete writes.

**Behaviour:**
- After scanning, filter out any `SmartFile` with `Size == 0`.

**Acceptance criteria:**
- A zero-byte `.pdf` file stays in the source directory after `arrange run`.

---

### FR-CAT-04 — Exclude-by-name flag

**Story:** As a user, I want to exclude specific files from a run using `--exclude` so I can protect files I'm actively using.

**Behaviour:**
- `--exclude` accepts a comma-separated list of base filenames.
- Any file whose `Name` exactly matches an entry in the list is skipped.

**Acceptance criteria:**
- `arrange run <dir> --exclude notes.txt,resume.pdf` does not move `notes.txt` or `resume.pdf`.
- Other files in the directory are moved normally.

---

## 3. Browser Suffix Stripping

### FR-STRIP-01 — Remove duplicate download suffixes

**Story:** As a user, I want files like `document (1).pdf` and `document (2).pdf` to arrive at the destination as `document.pdf` (with versioned suffixes for the duplicates) rather than keeping the browser numbering.

**Behaviour:**
- Before computing the destination filename, call `StripBrowserSuffix` on the stem.
- `StripBrowserSuffix` removes trailing ` (N)` patterns (space, parentheses, one or more digits) from the stem.
- The resulting clean stem is used as the destination filename.

**Acceptance criteria:**
- `document (1).pdf` → destination stem `document`.
- `movie (12).mkv` → destination stem `movie`.
- `my file.txt` (no suffix) → destination stem `my file`.

---

## 4. Collision-Safe Destination

### FR-COL-01 — Versioned suffix on name collision

**Story:** As a user, I want `arrange` to never silently overwrite an existing file at the destination.

**Behaviour:**
- `SafeDestPath(dir, stem, ext)` checks whether `dir/stem.ext` exists.
- If it does, try `dir/stem-v1.ext`, `dir/stem-v2.ext`, and so on until a free path is found.
- Return the first free path.

**Acceptance criteria:**
- Moving a second `document.pdf` to `Documents/` produces `Documents/document-v1.pdf`.
- A third produces `Documents/document-v2.pdf`.
- `SafeDestPath` never returns a path that already exists.

---

### FR-COL-02 — Atomic check-and-move per directory

**Story:** As a user running `arrange run` with many files targeting the same directory, I want no two workers to overwrite each other's files.

**Behaviour:**
- Each destination directory has an exclusive mutex (via `dirLocker`).
- The `SafeDestPath` call and the `Move` call execute inside the same mutex scope for a given destination directory.

**Acceptance criteria:**
- With 50 concurrent goroutines all moving files to the same directory, no two files end up at the same destination path.
- The race detector (`go test -race`) reports no data races.

---

## 5. Media Filename Parsing

### FR-PARSE-01 — TV series recognition

**Story:** As a media collector, I want `arrange` to recognise TV series files from their filenames so they land in the correct season folder.

**Behaviour:**
- Parse the stem for any of the recognised series patterns (see SRS §3.8).
- Set `Type = TypeTVSeries`, `Season`, `Episode`, `EpisodeEnd` (for multi-episode files).

**Acceptance criteria:**

| Input filename                 | Title          | Season | Episode | EpisodeEnd |
|--------------------------------|----------------|--------|---------|------------|
| `Breaking.Bad.S01E05.720p.mkv` | `Breaking Bad` | 1      | 5       | 0          |
| `Show.S01E01-E03.mkv`          | `Show`         | 1      | 1       | 3          |
| `Show.S013E17.mkv`             | `Show`         | 13     | 17      | 0          |
| `Show.1x05.mkv`                | `Show`         | 1      | 5       | 0          |
| `Show.S02.Complete.mkv`        | `Show`         | 2      | 0       | 0          |
| `Show.E05.mkv`                 | `Show`         | 0      | 5       | 0          |

---

### FR-PARSE-02 — Movie recognition

**Story:** As a media collector, I want movies to be identified with their year so they are placed in a `Title (YYYY)/` folder.

**Behaviour:**
- Extract the year from parentheses or a bare 4-digit year in the range 1900–2099.
- Set `Type = TypeMovie`, `Year`.

**Acceptance criteria:**

| Input filename                      | Title             | Year |
|-------------------------------------|-------------------|------|
| `Inception.2010.1080p.mkv`          | `Inception`       | 2010 |
| `The.Dark.Knight.(2008).BluRay.mkv` | `The Dark Knight` | 2008 |
| `Avengers.mkv` (no year)            | `Avengers`        | 0    |

---

### FR-PARSE-03 — Quality extraction

**Story:** As a media collector, I want quality tags in brackets in the destination filename so I can distinguish HD from SD copies at a glance.

**Behaviour:**
- Extract the quality tag; resolution takes priority over source tag.
- Append HDR modifier when present.

**Acceptance criteria:**

| Input                              | Quality        |
|------------------------------------|----------------|
| `Show.S01E05.1080p.BluRay.mkv`     | `1080p`        |
| `Movie.2160p.HDR10+.mkv`           | `2160p HDR10+` |
| `Movie.BluRay.mkv` (no resolution) | `BluRay`       |
| `Movie.WEB-DL.DV.mkv`              | `WEB-DL DV`    |

---

### FR-PARSE-04 — Title case normalisation

**Story:** As a media collector, I want titles formatted in proper English title case so my library looks clean.

**Behaviour:**
- Capitalise the first letter of most words.
- Known abbreviations are always all-caps: `UK`, `US`, `FBI`, `CIA`, `BBC`, `HBO`, `TV`, `DJ`, etc.
- Articles and short prepositions are lowercase in the middle of a title: `a`, `an`, `the`, `of`, `in`, `on`, `at`, `to`, `and`, `but`, `or`, `nor`, `for`, `with`, `by`, `as`, `vs`.
- The first word of a title is always capitalised even if it is a small word.

**Acceptance criteria:**

| Input stem                      | Title                      |
|---------------------------------|----------------------------|
| `Game.Of.Thrones.S01E01`        | `Game of Thrones`          |
| `fbi.S01E01`                    | `FBI`                      |
| `love.island.uk.S13E17`         | `Love Island UK`           |
| `the.shawshank.redemption.1994` | `The Shawshank Redemption` |

---

### FR-PARSE-05 — Creator grouping

**Story:** As a media collector who has many films by the same director, I want them grouped under a shared top-level folder.

**Behaviour:**
- Compare the title against each creator in `media_creators`.
- A match occurs when the title begins with `<creator>'s`, `<creator>s`, or `<creator>` followed by a space.
- Matching is case-insensitive.

**Acceptance criteria:**
- `media_creators: ["Tyler Perry"]`; file `Tyler.Perrys.Madea.Goes.to.Jail.2009.mkv` → dest `Videos/Tyler Perry/Madea Goes to Jail (2009)/`.
- File `Inception.2010.mkv` with the same config → dest `Videos/Inception (2010)/` (no creator match).

---

## 6. Background Service

### FR-SVC-01 — Install as OS service

**Story:** As a server administrator, I want `arrange` to start automatically on boot and watch a directory without manual intervention.

**Behaviour:**
- `arrange service install <dir>`:
  1. Runs `arrange run <dir>` for an initial organisation pass.
  2. Installs `arrange watch <dir>` as the service command.
  3. On macOS: creates a LaunchAgent plist in `~/Library/LaunchAgents/`.
  4. On Linux: creates a systemd user unit.

**Acceptance criteria:**
- After `arrange service install ~/Downloads`, running `arrange service status` reports the service as installed.
- After `arrange service start`, new files dropped in `~/Downloads` are automatically organised.

---

### FR-SVC-02 — Service lifecycle commands

**Story:** As a server administrator, I want to start, stop, check the status of, and remove the service without editing OS configuration files.

**Behaviour:**
- `service start` — start the installed service.
- `service stop` — stop the running service.
- `service status` — print the current service state.
- `service uninstall` (alias `remove`) — uninstall the service.

**Acceptance criteria:**
- All four commands succeed when the service has been installed.
- `service stop` stops an active watch process within 5 seconds.

---

## 7. Configuration

### FR-CFG-01 — Default config creation

**Story:** As a first-time user, I want `arrange` to create a sensible default config automatically so I don't have to write one.

**Behaviour:**
- On first invocation, if no config file exists at the platform path, create one with the built-in defaults.

**Acceptance criteria:**
- Running `arrange run <dir>` on a machine with no prior config creates `~/.config/arrange/config.json`.
- The created file is valid JSON.

---

### FR-CFG-02 — Custom extension mapping

**Story:** As a power user, I want to add my own extension-to-folder mapping in the config so I can handle file types `arrange` doesn't know about.

**Behaviour:**
- A `known_files` entry in `config.json` with my custom folder and extensions overrides or extends the built-in set.
- Setting `exempt_files: true` in an entry makes those extensions permanently exempt.

**Acceptance criteria:**
- Adding `{"folder": "CAD", "extensions": ["dwg", "dxf"], "exempt_files": false}` to `known_files` causes `.dwg` files to move to `CAD/`.

---

### FR-CFG-03 — Path override flag

**Story:** As a developer, I want to point `arrange` at a test config without modifying my system config.

**Behaviour:**
- `--config-path` / `-C` is a global flag available on all commands.

**Acceptance criteria:**
- `arrange run --config-path /tmp/test.json <dir>` loads the config from `/tmp/test.json`.

---

## 8. Filesystem Watch

### FR-WATCH-01 — Debounced batch processing

**Story:** As a user whose torrent client extracts archives in rapid succession, I want all the extracted files to be organised in one pass, not one per file.

**Behaviour:**
- On each filesystem create event, reset a per-directory timer to 800 ms.
- Only trigger `runE` when the timer fires (no new events in the last 800 ms).

**Acceptance criteria:**
- 20 files extracted in 200 ms triggers exactly one `runE` call, not 20.
- A single file dropped triggers `runE` approximately 800 ms after the event.

---

### FR-WATCH-02 — Auto-watch new subdirectories

**Story:** As a user with `--recursive` enabled, I want newly created subdirectories to be watched automatically without restarting `arrange`.

**Behaviour:**
- On a directory create event (detected because `os.Stat` reports `IsDir() == true`), add the new path to the watcher.

**Acceptance criteria:**
- A subdirectory created after `watch --recursive` starts is watched within 1 second.
- Files dropped into that subdirectory are organised normally.

---

## 9. Logging and Output

### FR-LOG-01 — Move log entries

**Story:** As a user, I want to see what files are being moved so I can verify the organisation is correct.

**Behaviour:**
- For each move, print a `MOVE` log line: `<src> → <dst>`.
- Truncate long paths to fit the terminal width.

**Acceptance criteria:**
- Every file move produces exactly one `MOVE` log line.
- No log line exceeds the terminal width.

---

### FR-LOG-02 — Error reporting

**Story:** As a user, I want to know when a file move fails so I can investigate.

**Behaviour:**
- Log errors to stderr using the `ERR` level.
- Continue processing other files after a move failure.
- Return all errors joined at the end of `run`.

**Acceptance criteria:**
- If one file is unreadable, the error is logged but remaining files are moved.
- `arrange run` exits with a non-zero status when at least one error occurred.
