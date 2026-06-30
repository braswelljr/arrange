# arrange

> Automatically organize files into folders by file type - run once, watch forever, or let a background service handle it.

`arrange` scans a directory and moves files into categorized subfolders (Pictures, Videos, Documents, Audio, etc.) based on their extensions. For video and audio it goes further: it parses filenames to detect TV series, movies, and multi-part films and builds a clean folder hierarchy automatically.

It can run as a one-shot command, react to new files in real-time via a filesystem watcher, or run as a persistent background service (macOS LaunchAgent / Linux systemd).

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [run](#run)
  - [media](#media)
  - [watch](#watch)
  - [service](#service)
  - [version](#version)
- [Running as a Service](#running-as-a-service)
  - [macOS (LaunchAgent)](#macos-launchagent)
  - [Linux (systemd)](#linux-systemd)
- [Configuration](#configuration)
  - [File Categories](#file-categories)
  - [Exempt Extensions](#exempt-extensions)
  - [Media Creators](#media-creators)
  - [Excluded Directories](#excluded-directories)
  - [Custom Categories](#custom-categories)
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

Or install directly into `$GOPATH/bin`:

```bash
make install
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

Files are moved into subfolders: `~/Downloads/Pictures`, `~/Downloads/Videos`, `~/Downloads/Documents`, etc.

To keep it organized automatically, [install it as a service](#running-as-a-service).

---

## Commands

### `run`

Scans a source directory and moves every file into its matching category subfolder.

```bash
arrange run <src> [dest]
```

| Argument | Description                                                     |
|----------|-----------------------------------------------------------------|
| `src`    | Directory to scan                                               |
| `dest`   | Destination root for organized folders (default: same as `src`) |

**Flags**

| Flag        | Short | Description                               |
|-------------|-------|-------------------------------------------|
| `--exclude` | `-c`  | Comma-separated list of filenames to skip |

**Behaviour notes**

- Browser-generated duplicate suffixes (`file (1).pdf`, `file (2).pdf`) are stripped before moving - the destination is always `file.pdf`.
- If the destination filename already exists, a versioned suffix is appended: `file-v1.pdf`, `file-v2.pdf`, and so on.
- In-progress downloads (`.crdownload`, `.part`, `.aria2`, `.!qb`, etc.) are never touched.
- Torrent meta-files (`.torrent`) are never moved.

**Examples**

```bash
# Organize Downloads in-place
arrange run ~/Downloads

# Move files from Downloads into a separate folder
arrange run ~/Downloads ~/Organized

# Skip specific files
arrange run ~/Downloads --exclude "notes.txt,resume.pdf"
```

---

### `media`

Parses video and audio filenames to detect TV series, movies, and multi-part films and arranges them into a clean hierarchy.

```bash
arrange media <src> [dest]
```

| Argument | Description                               |
|----------|-------------------------------------------|
| `src`    | Directory to scan                         |
| `dest`   | Destination root (default: same as `src`) |

**Output structure**

```
TV series   →  <Title>/Season XX/<Title> SxxExx [quality].ext
Movie       →  <Title> (YYYY)/<Title> (YYYY) [quality].ext
Multi-part  →  <Title> (YYYY)/Part N/<Title> (YYYY) [quality].ext
```

**Creator grouping**

Add director or franchise names to `media_creators` in your config. All their titles will be grouped under a shared top-level folder:

```
config:  "media_creators": ["Tyler Perry"]
result:  Tyler Perry/
           Why Did I Get Married (2007)/...
           Madea Goes to Jail (2009)/...
```

Both `"Tyler Perry's …"` and `"Tyler Perrys …"` (common in dot-separated filenames) are matched automatically.

**Examples**

```bash
arrange media ~/Downloads
arrange media ~/Downloads ~/Media
```

---

### `watch`

Watches a directory for new files and organizes them automatically as they arrive.

```bash
arrange watch <dir>
```

**Flags**

| Flag          | Short | Default | Description                   |
|---------------|-------|---------|-------------------------------|
| `--recursive` | `-r`  | `false` | Also watch all subdirectories |

**Behaviour notes**

- Changes are debounced (800 ms) so a burst of files (archive extraction, batch copy) triggers a single organize pass.
- In recursive mode, newly created subdirectories are watched automatically.
- Files that arrive in app sub-folders (Telegram Desktop, WhatsApp, etc.) are sorted into the **root** watched directory - e.g. a video dropped into `~/Downloads/Telegram Desktop/` is moved to `~/Downloads/Videos/`, not `~/Downloads/Telegram Desktop/Videos/`.
- Directories listed in `excluded_dirs` in your config are never watched.

```bash
# Watch Downloads and all its subdirectories
arrange watch --recursive ~/Downloads
```

The process runs in the foreground. Use [`service`](#service) to run it persistently in the background.

---

### `service`

Manages `arrange` as a background service (macOS LaunchAgent or Linux systemd user unit).

```bash
arrange service <command>
```

| Subcommand      | Description                                     |
|-----------------|-------------------------------------------------|
| `install <dir>` | Install the service and set it to watch `<dir>` |
| `start`         | Start the installed service                     |
| `stop`          | Stop the running service                        |
| `status`        | Check whether the service is running            |
| `uninstall`     | Remove the service entirely                     |

---

### `version`

Print the current version.

```bash
arrange version
```

---

## Running as a Service

### macOS (LaunchAgent)

`arrange` runs as a LaunchAgent so it starts automatically on login and watches your Downloads folder silently in the background.

**1. Build and install the binary**

```bash
make build
sudo mv ./bin/arrange /usr/local/bin/arrange
```

> The binary path is recorded in the LaunchAgent plist - don't move it after installation.

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

**5. Verify**

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

This creates and enables a systemd user unit. The service restarts automatically on boot.

**3. Check status**

```bash
arrange service status
# or directly:
systemctl --user status arrange
```

---

## Configuration

The config file is created automatically on first run at:

| Platform      | Path |
| ------------- | ---- |
| macOS / Linux | `~/.config/arrange/config.json` |
| Custom        | Pass `--config-path <path>` to any command |

Full schema:

```json
{
  "unknown_files_folder": "Other",
  "excluded_dirs": [],
  "media_creators": [],
  "known_files": []
}
```

### File Categories

The default configuration organizes files into:

| Folder         | Example extensions                                              |
|----------------|-----------------------------------------------------------------|
| `Pictures`     | `jpg` `png` `gif` `webp` `heic` `svg` `psd` `raw` `arw` `nef` … |
| `Videos`       | `mp4` `mkv` `mov` `avi` `webm` `ts` `m2ts` `vob` …              |
| `Audio`        | `mp3` `flac` `wav` `aac` `opus` `wma` `ogg` `m4a` …             |
| `Documents`    | `pdf` `docx` `xlsx` `pptx` `txt` `md` `csv` `tex` …             |
| `eBooks`       | `epub` `mobi` `azw3` `fb2` `djvu` `cbr` `cbz` …                 |
| `Applications` | `exe` `dmg` `deb` `apk` `msi` `appimage` `ipa` …                |
| `Archive`      | `zip` `tar` `gz` `rar` `7z` `xz` `zst` …                        |
| `DiskImages`   | `iso` `img` `vmdk` `vhd` `qcow2` …                              |
| `Fonts`        | `ttf` `otf` `woff` `woff2` `fon` …                              |
| `Database`     | `sqlite` `db` `sql` `mdb` `accdb` …                             |
| `Subtitles`    | `srt` `ass` `vtt` `sub` `idx` `sup` …                           |
| `Design`       | `fig` `sketch` `xd` `ai` `indd` `afdesign` …                    |
| `3D Models`    | `obj` `fbx` `stl` `blend` `gltf` `glb` `c4d` …                  |
| `Other`        | anything not matched above                                      |

### Exempt Extensions

Exempt entries (`exempt_files: true`) are **never moved**.

| Group                 | Extensions                                                                    |
|-----------------------|-------------------------------------------------------------------------------|
| Source code           | `go` `py` `js` `ts` `rs` `java` `html` `css` `json` `yaml` …                  |
| Torrent meta          | `torrent` `magnet`                                                            |
| In-progress downloads | `crdownload` `part` `download` `!qb` `!ut` `!bt` `aria2` `fdm` `idm` `ytdl` … |

### Media Creators

List director or franchise names in `media_creators` and `arrange media` will group all their titles under a shared folder:

```json
{
  "media_creators": ["Tyler Perry", "Christopher Nolan", "Pixar"]
}
```

### Excluded Directories

List subdirectory names in `excluded_dirs` to prevent `arrange watch` from watching them:

```json
{
  "excluded_dirs": ["node_modules", ".git", "tmp"]
}
```

Files that land in unlisted app subdirectories (Telegram Desktop, WhatsApp, etc.) are still organized normally - they are moved to the correct category folder under the root watched directory.

### Custom Categories

Edit `~/.config/arrange/config.json` to add or override categories:

```json
{
  "unknown_files_folder": "Other",
  "media_creators": ["Tyler Perry"],
  "known_files": [
    {
      "folder": "Design",
      "extensions": ["fig", "sketch", "xd"],
      "exempt_files": false
    },
    {
      "folder": "",
      "extensions": ["log"],
      "exempt_files": true
    }
  ]
}
```

Set `exempt_files: true` on any entry to make those extensions always be skipped.

---

## Global Flags

Available on every command:

| Flag            | Short | Default                         | Description                  |
|-----------------|-------|---------------------------------|------------------------------|
| `--config-path` | `-C`  | `~/.config/arrange/config.json` | Path to a custom config file |
| `--help`        | `-h`  | -                               | Help for the current command |
