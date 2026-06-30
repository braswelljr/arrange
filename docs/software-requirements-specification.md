# Software Requirements Specification (SRS)

**Project:** arrange  
**Version:** 1.0  
**Date:** 2026-06-30  
**Status:** Baseline

---

## 1. Introduction

### 1.1 Purpose

This document specifies the software requirements for `arrange`, a cross-platform CLI tool that organises files into categorised subfolders by file extension, with intelligent media filename parsing for TV series, movies, and multi-part films.

### 1.2 Scope

The specification covers all five commands (`run`, `watch`, `media`, `service`, `version`), the configuration system, the media filename parser, the file operations layer, and the background service integration. It does not cover the user interface beyond the CLI itself.

### 1.3 Definitions

| Term                      | Definition                                                                                                               |
|---------------------------|--------------------------------------------------------------------------------------------------------------------------|
| **Source directory**      | The directory scanned for files to organise                                                                              |
| **Destination directory** | The root under which organised subfolders are created; defaults to the source directory                                  |
| **Category folder**       | A named subfolder created under the destination directory (e.g. `Videos`, `Documents`)                                   |
| **Exempt extension**      | A file extension that `arrange` will never move under any circumstance                                                   |
| **Media hierarchy**       | The structured subdirectory tree produced for video/audio files: `Category/Title/Season XX/` or `Category/Title (YYYY)/` |
| **Season pack**           | A video file containing a full season with no individual episode marker                                                  |
| **Collision-safe path**   | A destination path that avoids overwriting an existing file by appending `-v1`, `-v2`, etc.                              |
| **Browser suffix**        | A duplicate-download suffix appended by browsers: `(1)`, `(2)`, etc.                                                     |

### 1.4 Overview

Section 2 gives system context. Section 3 lists functional requirements. Section 4 lists non-functional requirements. Section 5 covers external interfaces. Section 6 covers constraints.

---

## 2. System Context

```
┌─────────────────────────────────────────────────────┐
│                    User's Machine                   │
│                                                     │
│  ┌──────────────┐     ┌───────────────────────────┐ │
│  │  Source Dir  │────▶│       arrange CLI         │ │
│  │  (Downloads) │     │                           │ │
│  └──────────────┘     │  ┌─────────┐ ┌─────────┐  │ │
│                       │  │ config  │ │  media  │  │ │
│  ┌──────────────┐     │  │ system  │ │ parser  │  │ │
│  │   Dest Dir   │◀────│  └─────────┘ └─────────┘  │ │
│  │  (Organised) │     │  ┌─────────┐ ┌─────────┐  │ │
│  └──────────────┘     │  │fileops  │ │ logger  │  │ │
│                       │  └─────────┘ └─────────┘  │ │
│  ┌──────────────┐     └───────────────────────────┘ │
│  │ config.json  │◀────── ~/.config/arrange/         │
│  └──────────────┘                                   │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │ OS Service Layer (LaunchAgent / systemd)     │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

`arrange` is a single statically-linked binary. It reads a JSON config file, scans a directory, and moves files. All side effects are filesystem operations on the user's local machine.

---

## 3. Functional Requirements

### 3.1 Command: `run`

**FR-RUN-01** The `run` command shall accept one required argument (source directory) and one optional argument (destination directory). When the destination is omitted it shall default to the source directory.

**FR-RUN-02** `run` shall scan the source directory, categorise each file by extension, and move it to `<dest>/<CategoryFolder>/<filename>`.

**FR-RUN-03** When `--recursive` / `-r` is set, `run` shall walk all subdirectories of the source directory, not just the top level.

**FR-RUN-04** When `--recursive` is set and `src == dest`, `run` shall skip any subdirectory whose name matches a known destination category folder (case-insensitive) to avoid re-processing already-organised files.

**FR-RUN-05** `run` shall accept `--exclude` / `-c` with a comma-separated list of filenames. Files whose base name exactly matches an entry in this list shall not be moved.

**FR-RUN-06** `run` shall skip files whose size is zero bytes.

**FR-RUN-07** `run` shall skip files whose extension is on the exempt list.

**FR-RUN-08** Before moving, `run` shall strip browser-generated duplicate suffixes (`(1)`, `(2)`, etc.) from the destination filename stem.

**FR-RUN-09** If the computed destination path already exists, `run` shall append a versioned suffix (`-v1`, `-v2`, …) to the stem to produce a unique path.

**FR-RUN-10** `run` shall process files concurrently using a worker pool of `runtime.NumCPU()` goroutines (capped to the number of files).

**FR-RUN-11** If any individual file move fails, `run` shall collect the error and continue processing remaining files. All errors shall be returned as a joined error at the end.

### 3.2 Command: `watch`

**FR-WATCH-01** The `watch` command shall accept one argument (the directory to watch) and begin monitoring it for filesystem create events using OS-native notifications.

**FR-WATCH-02** When a new file appears in the watched directory, `watch` shall trigger an organise pass on the directory containing the new file after a debounce delay of 800 ms.

**FR-WATCH-03** The debounce window shall reset on each new event within the same directory, so a burst of files (archive extraction, multi-file copy) triggers a single pass.

**FR-WATCH-04** When `--recursive` / `-r` is set, `watch` shall also watch all existing subdirectories. Newly created subdirectories shall be added to the watch set automatically.

**FR-WATCH-05** Files that arrive in a watched subdirectory (e.g. `Telegram Desktop/`) shall be moved to the correct category folder under the **root** watched directory, not inside the subdirectory.

**FR-WATCH-06** Directories listed in `excluded_dirs` in the config shall never be added to the watch set.

**FR-WATCH-07** `watch` shall run in the foreground and block until interrupted. It shall not daemonise itself; use `service` for background operation.

### 3.3 Command: `media`

**FR-MEDIA-01** The `media` command shall accept one required argument (source directory) and one optional argument (destination directory).

**FR-MEDIA-02** `media` shall scan the source directory for video and audio files only and organise them into the standard media hierarchy using the filename parser.

**FR-MEDIA-03** `media` shall apply creator grouping: if a title begins with a creator name from `media_creators` (possessive or plain prefix), that title's files shall be placed under `<creator>/` as the top-level folder.

### 3.4 Command: `service`

**FR-SVC-01** The `service` command shall manage `arrange` as a background OS service. It shall be available on macOS and Linux only.

**FR-SVC-02** `service install <dir>` shall perform an initial `run` on `<dir>`, then install `arrange watch <dir>` as a system service (macOS: LaunchAgent; Linux: systemd user unit).

**FR-SVC-03** `service start`, `service stop`, `service status`, and `service uninstall` (alias: `remove`) shall delegate to the OS service manager.

### 3.5 Command: `version`

**FR-VER-01** The `version` command shall print the build version string embedded at compile time via `-ldflags`.

### 3.6 Configuration System

**FR-CFG-01** On first run, `arrange` shall create a default config file at the platform-appropriate path:

| Platform | Path |
|----------|------|
| macOS / Linux | `$XDG_CONFIG_HOME/arrange/config.json` (defaults to `~/.config/arrange/config.json`) |
| Windows | `%APPDATA%\arrange\config.json` |

**FR-CFG-02** All commands shall accept `--config-path` / `-C` to override the default config path.

**FR-CFG-03** The config file shall support the following fields:

| Field                  | Type     | Description                                                  |
|------------------------|----------|--------------------------------------------------------------|
| `unknown_files_folder` | string   | Folder name for unrecognised extensions (default: `"Other"`) |
| `excluded_dirs`        | []string | Directory names excluded from `watch`                        |
| `media_creators`       | []string | Creator/franchise names for media grouping                   |
| `known_files`          | []object | Extension-to-folder mapping entries (see FR-CFG-04)          |

**FR-CFG-04** Each `known_files` entry shall have:
- `folder` (string): destination folder name; empty string means exempt without a folder.
- `extensions` ([]string): file extensions this entry covers (without leading dot).
- `exempt_files` (bool): when true, matching files are never moved.

**FR-CFG-05** The first matching entry for a given extension shall win. Subsequent entries for the same extension are ignored.

**FR-CFG-06** Extensions not matched by any entry shall be moved to `unknown_files_folder`.

### 3.7 File Extension Categorisation

**FR-EXT-01** The built-in default config shall include the following categories with at least the listed extensions:

| Category     | Representative Extensions                               |
|--------------|---------------------------------------------------------|
| Pictures     | jpg, png, gif, webp, heic, psd, svg, raw, cr2, arw, nef |
| Videos       | mp4, mkv, mov, avi, webm, ts, m2ts, vob                 |
| Audio        | mp3, flac, wav, aac, opus, wma, ogg, m4a                |
| Documents    | pdf, docx, xlsx, pptx, txt, md, csv, rtf                |
| eBooks       | epub, mobi, azw3, fb2, djvu, cbr, cbz                   |
| Applications | exe, dmg, deb, apk, msi, appimage, ipa                  |
| Archive      | zip, tar, gz, rar, 7z, xz, zst                          |
| DiskImages   | iso, img, vmdk, vhd, qcow2                              |
| Fonts        | ttf, otf, woff, woff2                                   |
| Database     | sqlite, db, sql, mdb                                    |
| Subtitles    | srt, ass, vtt, sub, idx                                 |
| Design       | fig, sketch, xd, ai, indd, afdesign                     |
| 3D Models    | obj, fbx, stl, blend, gltf, c4d                         |

**FR-EXT-02** The following extension groups shall always be exempt (never moved):

| Group | Extensions (representative) |
|-------|-----------------------------|
| Source code | go, py, js, ts, rs, java, html, css, json, yaml, sh |
| Torrent meta | torrent, magnet |
| In-progress downloads | crdownload, part, download, !qb, !ut, aria2, fdm, idm |

### 3.8 Media Filename Parser

**FR-PARSE-01** The parser shall identify TV series, movies, and multi-part movies from a bare filename.

**FR-PARSE-02** The parser shall recognise the following episode/season naming patterns (in priority order):

| Pattern                       | Example                       |
|-------------------------------|-------------------------------|
| `S01E05`, `S01E05-E07`        | `Show.S01E05.mkv`             |
| `S01 E05` (space variant)     | `Show S01 E05.mkv`            |
| `1x05` (NxNN format)          | `Show.1x05.mkv`               |
| `Season 1 Episode 5`          | `Show.Season.1.Episode.5.mkv` |
| `Season 1` (season-only pack) | `Show.Season.1.Complete.mkv`  |
| `S01` (season pack shorthand) | `Show.S02.mkv`                |
| `Episode 05` / `Ep 05`        | `Show.Ep05.mkv`               |
| `E05` (bare episode marker)   | `Show.E05.mkv`                |
| Anime dash `- 05`             | `Show Name - 05 [720p].mkv`   |

Season numbers shall support 1–3 digits (e.g. `S013E17` for season 13, episode 17).

**FR-PARSE-03** The parser shall extract a year from `(YYYY)` or bare `YYYY` in the range 1900–2099.

**FR-PARSE-04** The parser shall extract a quality tag using the following priority:

1. Resolution (`4K`/`UHD`, `2160p`, `1080p`, `720p`, `480p`, `360p`) — always preferred.
2. Source (`BluRay`, `WEBRip`, `WEB-DL`, `HDTV`, `DVDRip`, etc.) — fallback.
3. HDR modifier (`HDR10+`, `HDR10`, `HDR`, `DV`, `Dolby Vision`) — appended to the base quality tag.

**FR-PARSE-05** The parser shall strip the following noise from extracted titles: codec names (x264, x265, HEVC, AAC, DTS, FLAC), release modifiers (REMUX, PROPER, REPACK, EXTENDED), scene group names (YIFY, RARBG, SubsPlease, HorribleSubs), and quality tokens.

**FR-PARSE-06** The parser shall normalise titles to English title case:
- Known abbreviations (UK, US, USA, FBI, CIA, BBC, HBO, TV, DJ, etc.) shall always be all-caps.
- Articles and short prepositions (a, an, the, of, in, on, at, to, and, but, or, for, with, by, as, vs) shall be lowercased except as the first word.
- All other words shall be capitalised on the first letter.

**FR-PARSE-07** The parser shall normalise dot-separated filenames by replacing dots with spaces when dots outnumber spaces. Underscores shall always be replaced with spaces.

**FR-PARSE-08** The parser shall strip leading release-group brackets (`[GroupName]`, `(GroupName)`) from filenames before parsing.

### 3.9 Media Destination Layout

**FR-DEST-01** The destination directory structure for media files shall follow:

| File type                | Structure                                                 |
|--------------------------|-----------------------------------------------------------|
| TV series (S+E known)    | `Category/Title/Season XX/Title SxxExx [quality].ext`     |
| TV series (season pack)  | `Category/Title/Season XX/Title Sxx.ext`                  |
| TV series (bare episode) | `Category/Title/Title Exx.ext`                            |
| Movie                    | `Category/Title (YYYY)/Title (YYYY) [quality].ext`        |
| Movie (no year)          | `Category/Title/Title [quality].ext`                      |
| Multi-part movie         | `Category/Title (YYYY)/Part N/Title (YYYY) [quality].ext` |

**FR-DEST-02** When a creator is matched, the creator name shall be inserted as the top-level folder inside the category:
```
Videos/Tyler Perry/Madea Goes to Jail (2009)/Madea Goes to Jail (2009) [1080p].mkv
```

**FR-DEST-03** All output filenames shall be sanitised: `:` → ` -`, `/` and `\` → `-`, `|` → `-`, `*`, `?`, `"`, `<`, `>` → removed. The result shall be safe on Windows, macOS, and Linux.

### 3.10 File Operations

**FR-OPS-01** When source and destination are on the same filesystem, `arrange` shall move files using `os.Rename` (atomic on most Unix filesystems).

**FR-OPS-02** When `os.Rename` fails with a cross-device error (`EXDEV`), `arrange` shall fall back to copying the file to the destination then removing the source.

**FR-OPS-03** Destination directories shall be created automatically with permissions `0750`.

**FR-OPS-04** Before moving, `arrange` shall check that the destination file does not exist. If it does, a versioned suffix shall be appended to the stem: `-v1`, `-v2`, and so on, until a free path is found.

**FR-OPS-05** Within a single `run` invocation, the `SafeDestPath + Move` pair for a given destination directory shall execute atomically under a per-directory mutex so concurrent workers cannot both claim the same free path.
