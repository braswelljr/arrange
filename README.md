# arrange

> Automatically organize files into folders by file type — run once or watch a directory forever.

`arrange` scans a directory and moves files into categorized subfolders (Pictures, Videos, Documents, Audio, etc.) based on their extensions. It can run as a one-shot command or as a persistent background service (macOS LaunchAgent / Linux systemd) that reacts to new files the moment they land.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [run](#run)
  - [watch](#watch)
  - [service](#service)
  - [version](#version)
- [Running as a Service](#running-as-a-service)
  - [macOS (LaunchAgent)](#macos-launchagent)
  - [Linux (systemd)](#linux-systemd)
- [Configuration](#configuration)
  - [File Categories](#file-categories)
  - [Adding Custom Categories](#adding-custom-categories)
- [Global Flags](#global-flags)

---

## Installation

### From source

Requires [Go 1.25+](https://go.dev/dl/).

```bash
git clone https://github.com/braswelljr/arrange.git
cd arrange
make build
sudo mv ./bin/arrange /usr/local/bin/arrange
```

### Verify

```bash
arrange version
```

---

## Quick Start

Organize your Downloads folder right now:

```bash
arrange run ~/Downloads
```

That's it. Files are moved into subfolders like `~/Downloads/Pictures`, `~/Downloads/Videos`, `~/Downloads/Documents`, etc.

To keep it organized automatically going forward, [install it as a service](#running-as-a-service).

---

## Commands

### `run`

Scans a source directory and moves files into categorized subfolders.

```bash
arrange run <src> [dest]
```

| Argument | Description                                                            |
| -------- | ---------------------------------------------------------------------- |
| `src`  | Directory to scan for files                                            |
| `dest` | (Optional) Destination root for organized folders. Defaults to `src` |

**Flags**

| Flag          | Short  | Description                               |
| ------------- | ------ | ----------------------------------------- |
| `--exclude` | `-c` | Comma-separated list of filenames to skip |

**Examples**

```bash
# Organize Downloads in-place
arrange run ~/Downloads

# Move files from Downloads into a separate Organized folder
arrange run ~/Downloads ~/Organized

# Skip specific files
arrange run ~/Downloads --exclude "notes.txt,resume.pdf"
```

---

### `watch`

Watches a directory for new files and organizes them automatically as they arrive.

```bash
arrange watch <dir>
```

```bash
arrange watch ~/Downloads
```

The process runs in the foreground. Use [`service`](#service) to run it persistently in the background.

---

### `service`

Manages `arrange` as a background service (macOS LaunchAgent or Linux system daemon).

```bash
arrange service <command>
```

| Subcommand        | Description                            |
| ----------------- | -------------------------------------- |
| `install <dir>` | Install the service to watch `<dir>` |
| `start`         | Start the installed service            |
| `stop`          | Stop the running service               |
| `status`        | Check whether the service is running   |
| `uninstall`     | Remove the service entirely            |

---

### `version`

Print the current version.

```bash
arrange version
```

---

## Running as a Service

### macOS (LaunchAgent)

Running `arrange` as a LaunchAgent means it starts automatically on login and watches your Downloads folder silently in the background.

**1. Build and install the binary**

```bash
make build
sudo mv ./bin/arrange /usr/local/bin/arrange
```

> The binary path is recorded in the LaunchAgent plist — don't move it after installation.

**2. Do an initial cleanup**

```bash
arrange run ~/Downloads
```

**3. Install the LaunchAgent**

```bash
arrange service install ~/Downloads
```

This creates `~/Library/LaunchAgents/arrange.plist` configured to run `arrange watch ~/Downloads`.

**4. Start the service**

```bash
arrange service start
```

**5. Verify it's running**

```bash
arrange service status
```

**To stop or remove:**

```bash
arrange service stop        # pause temporarily
arrange service uninstall   # remove the LaunchAgent entirely
```

---

### Linux (systemd)

**1. Build and install the binary**

```bash
make build
sudo mv ./bin/arrange /usr/local/bin/arrange
```

**2. Install and start the service**

```bash
arrange service install ~/Downloads
arrange service start
```

This creates and enables a systemd user unit. The service will restart automatically on boot.

**3. Check status**

```bash
arrange service status
# or directly via systemctl:
systemctl --user status arrange
```

---

## Configuration

The config file is created automatically on first run at:

| Platform      | Path                                         |
| ------------- | -------------------------------------------- |
| macOS / Linux | `~/.config/arrange/config.json`            |
| Custom        | Pass `--config-path <path>` to any command |

### File Categories

The default configuration organizes files into:

| Folder                | Extensions (examples)                                                      |
| --------------------- | -------------------------------------------------------------------------- |
| `Pictures`          | `jpg`, `png`, `gif`, `svg`, `heic`, `raw`, `arw`, `nef` … |
| `Videos`            | `mp4`, `mkv`, `mov`, `avi`, `webm`, `m2ts` …                  |
| `Audio`             | `mp3`, `flac`, `wav`, `aac`, `opus`, `wma` …                  |
| `Documents`         | `pdf`, `docx`, `xlsx`, `txt`, `md`, `csv`, `tex` …          |
| `eBooks`            | `epub`, `mobi`, `azw3`, `fb2`, `djvu`                            |
| `Applications`      | `exe`, `dmg`, `deb`, `apk`, `msi`, `appimage` …               |
| `Archive`           | `zip`, `tar`, `gz`, `rar`, `7z`, `jar` …                      |
| `DiskImages`        | `iso`, `img`, `vmdk`, `vhd`, `qcow2`                             |
| `Fonts`             | `ttf`, `otf`, `woff`, `woff2` …                                   |
| `Database`          | `sqlite`, `db`, `sql`, `mdb` …                                    |
| `Torrents`          | `torrent`                                                                |
| `Code` *(exempt)* | `py`, `js`, `go`, `ts`, `html`, `json`, `yaml` …            |
| `Other`             | anything not matched above                                                 |

> **Exempt categories** (`exempt_files: true`) are **skipped** — files with those extensions are left in place. `Code` is exempt by default to avoid moving source files.

### Adding Custom Categories

Edit `~/.config/arrange/config.json`:

```json
{
  "unknown_files_folder": "Other",
  "known_files": [
    {
      "folder": "Design",
      "extensions": ["fig", "sketch", "xd"],
      "exempt_files": false
    }
  ]
}
```

To **skip** certain file types entirely, set `"exempt_files": true`.

---

## Global Flags

Available on every command:

| Flag              | Short  | Default                           | Description                  |
| ----------------- | ------ | --------------------------------- | ---------------------------- |
| `--config-path` | `-C` | `~/.config/arrange/config.json` | Path to a custom config file |
| `--help`        | `-h` | —                                | Help for the current command |
