# Business Requirements Document (BRD)

**Project:** arrange  
**Version:** 1.0  
**Date:** 2026-06-30  
**Status:** Approved

---

## 1. Executive Summary

`arrange` is a cross-platform command-line tool that automatically organises files in a directory by moving them into categorised subfolders based on file extension. For video and audio files it goes further, parsing filenames to detect TV series, movies, and multi-part films and placing each file in a structured, human-readable hierarchy.

The tool solves a universal problem: Downloads folders accumulate hundreds of files of mixed types, making it difficult to find any particular file. Manual organisation is tedious and never happens consistently. `arrange` automates the process entirely — one command, a background watcher, or a persistent system service.

---

## 2. Business Problem

### 2.1 Problem Statement

Users who download content (video, audio, documents, software installers, fonts, etc.) accumulate flat, unsorted directories over time. Common consequences:

- Files cannot be found without a full-text search.
- Duplicate files are not detected because names differ (browser adds `(1)`, `(2)` suffixes).
- Streaming and media players cannot build libraries from a flat directory.
- Storage becomes fragmented and difficult to audit.
- In-progress downloads are accidentally moved mid-write, corrupting the file.

Existing tools either require a GUI, operate only on one file type, are not idempotent, or touch in-progress downloads.

### 2.2 Opportunity

A lightweight CLI tool that runs in the foreground, in the background, or as a system service would let users delegate directory maintenance entirely to the machine. When combined with intelligent media parsing, the tool also doubles as a media library organiser compatible with Plex, Jellyfin, Kodi, and similar systems.

### 2.3 Target Users

| User | Primary Use |
|------|-------------|
| Power users / developers | Keep Downloads tidy with a one-shot command or `--recursive` sweep |
| Media collectors | Organise TV series and movie libraries into a standard hierarchy |
| Server / NAS administrators | Run as a persistent background service on Linux (systemd) |
| macOS users | Run as a LaunchAgent that starts automatically on login |

---

## 3. Business Objectives

| ID | Objective | Measure of Success |
|----|-----------|-------------------|
| BO-1 | Automate file organisation with zero manual categorisation | User can run one command and find every file in the correct folder |
| BO-2 | Support real-time organisation without user intervention | New files are moved within 1 second of appearing in the watched directory |
| BO-3 | Never damage in-progress downloads | Zero reports of download corruption after a `watch` or `run` session |
| BO-4 | Produce a media hierarchy compatible with standard media servers | Plex / Jellyfin picks up organised files without additional metadata |
| BO-5 | Work across macOS, Linux, and Windows without reinstallation | A single binary per platform covers all supported environments |
| BO-6 | Allow user customisation without code changes | All categorisation rules are editable in a plain JSON config file |

---

## 4. Scope

### 4.1 In Scope

- File categorisation by extension into named folders.
- Intelligent media filename parsing for TV series, movies, and multi-part films.
- One-shot execution (`run`), filesystem watching (`watch`), and background service (`service`) modes.
- Recursive directory scanning.
- Collision-safe destination naming (versioned suffix).
- Browser-generated duplicate suffix stripping.
- User-configurable extension-to-folder mapping.
- User-configurable creator grouping for media files.
- Exempt extensions (never moved): in-progress downloads, source code, torrent meta.
- Cross-device move support (copy-then-delete fallback).
- macOS LaunchAgent and Linux systemd service management.
- JSON configuration file with platform-appropriate default path.

### 4.2 Out of Scope

- Duplicate detection by content hash.
- File renaming without moving.
- Graphical user interface.
- Network/remote directory support.
- Cloud storage integration.
- Windows service management (Windows Task Scheduler recommended instead).
- Metadata tagging (ID3, EXIF, etc.).

---

## 5. Stakeholders

| Role | Responsibility |
|------|---------------|
| End users | Define which file types matter and how to categorise them |
| Maintainer (`braswelljr`) | Design, implement, and release the binary |
| Media server users (Plex, Jellyfin) | Define the expected folder hierarchy for TV and movies |
| NAS / server administrators | Define service installation and lifecycle requirements |

---

## 6. Assumptions and Constraints

### 6.1 Assumptions

- Users have read and write permission on the source and destination directories.
- The Go runtime is available on the build machine; end users run a pre-built binary.
- Source and destination may be on different filesystems (cross-device moves handled automatically).
- Config file location follows XDG Base Directory Specification on Unix and `%APPDATA%` on Windows.
- Media filenames follow common scene, P2P, or streaming release conventions.

### 6.2 Constraints

- The tool must not move or touch any file whose extension is on the exempt list, even temporarily.
- Destination naming must be deterministic and idempotent: running `arrange run` twice on the same directory must not create duplicate files or move already-organised files again.
- The binary must build and run on Go 1.25+.
- The service subcommand is not available on Windows.

---

## 7. Success Criteria

| Criterion | Threshold |
|-----------|-----------|
| A fresh Downloads folder of 500 mixed files is fully organised | Under 5 seconds on a modern laptop |
| Zero in-progress download files are ever moved | 100% — any move of an exempt extension is a critical bug |
| TV series are placed in the correct `Title/Season XX/` hierarchy | ≥ 95% of files with standard scene naming |
| Config changes take effect on the next `run` or `watch` restart | Always |
| `--recursive` on a directory that was already organised is a no-op | No files move and no errors appear |

---

## 8. Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/spf13/cobra` | v1.10+ | CLI framework |
| `github.com/fsnotify/fsnotify` | v1.10+ | Cross-platform filesystem events |
| `github.com/takama/daemon` | v1.0+ | macOS LaunchAgent / Linux systemd management |
| `github.com/fatih/color` | v1.19+ | Terminal colour output |
| `golang.org/x/term` | v0.44+ | Terminal width detection |

---

## 9. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Parser misidentifies a movie title as a TV episode number | Medium | Files placed in wrong hierarchy | Comprehensive regex test suite; priority ordering places explicit series markers above bare year detection |
| User moves the binary after service installation | Low | Service fails to start | README warns; service `install` records the binary path at install time |
| Cross-device rename fails silently | Low | Data loss | `fileops.Move` explicitly handles `EXDEV` and falls back to copy-then-delete |
| Regex matches production source code extensions | n/a | Source code moved out of project | Source code extensions are always exempt; this cannot happen |
