package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/braswelljr/arrange/internal/fileops"
)

var (
	cfg       *Config
	pathCache string
	cfgMu     sync.RWMutex
)

/**
 * Config is the top-level configuration loaded from the user's JSON file.
 *
 * Fields:
 *   - UnknownFilesFolder:  destination folder for unrecognized extensions (default: "Other")
 *   - KnownFiles:          ordered list of extension → folder mappings
 *   - ExcludedDirs:        directory names to never watch or organize
 *   - MediaCreators:       creator names used for top-level grouping of media files
 *   - Path:                absolute path of the loaded config file (set by NewConfig)
 */
type Config struct {
	UnknownFilesFolder string    `json:"unknown_files_folder"`
	KnownFiles         []FileExt `json:"known_files"`
	ExcludedDirs       []string  `json:"excluded_dirs"`
	MediaCreators      []string  `json:"media_creators"`

	Path string

	initOnce     sync.Once
	exemptedExts map[string]string
	includedExts map[string]string
	excludedDirs map[string]struct{}
}

/**
 * FileExt maps a set of file extensions to a destination folder.
 *
 * Fields:
 *   - Extensions:  list of lowercase extensions without a leading dot (e.g. "mp4", "mkv")
 *   - Folder:      destination folder name relative to the watched root (e.g. "Videos")
 *   - ExemptFiles: when true, matching files are never moved (used for in-progress downloads)
 */
type FileExt struct {
	Extensions  []string `json:"extensions"`
	Folder      string   `json:"folder"`
	ExemptFiles bool     `json:"exempt_files"`
}

/**
 * NewConfig loads (or creates) the config file at path and returns a parsed Config.
 *
 * If path is empty the platform-default config path is used (see config.Path()).
 * If the file does not exist, a default config is written to disk before returning.
 * Results are cached per path; the same *Config pointer is returned on subsequent calls.
 *
 * @param path  absolute path to the JSON config file, or "" for the platform default
 * @returns     (*Config, nil) on success, or (nil, error) on I/O or parse failure
 *
 * Usage:
 *   cfg, err := config.NewConfig("")          // use default path
 *   cfg, err := config.NewConfig("/my/path")  // explicit path
 */
func NewConfig(path string) (*Config, error) {
	if path == "" {
		path = Path()
	}

	cfgMu.RLock()
	if cfg != nil && path == pathCache {
		c := cfg
		cfgMu.RUnlock()
		return c, nil
	}
	cfgMu.RUnlock()

	if !fileops.FileExists(path) {
		err := os.MkdirAll(filepath.Dir(path), 0750)
		if err != nil {
			return nil, err
		}

		defaultCfg := &Config{
			UnknownFilesFolder: defaultUnknownExtFolderName,
			KnownFiles:         defaultKnownFiles,
			ExcludedDirs:       defaultExcludedDirs,
			MediaCreators:      []string{},
		}
		data, err := json.MarshalIndent(defaultCfg, "", "\t")
		if err != nil {
			return nil, fmt.Errorf("could not marshal default config: %w", err)
		}
		err = os.WriteFile(path, data, 0600)
		if err != nil {
			return nil, fmt.Errorf("could not create config file: %w", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	newCfg := &Config{
		UnknownFilesFolder: defaultUnknownExtFolderName,
	}

	err = json.Unmarshal(data, newCfg)
	if err != nil {
		return nil, err
	}

	newCfg.Path = path

	cfgMu.Lock()
	cfg = newCfg
	pathCache = path
	cfgMu.Unlock()

	return newCfg, nil
}

func (c *Config) initCfgMap() {
	c.initOnce.Do(func() {
		c.includedExts = make(map[string]string)
		c.exemptedExts = make(map[string]string)
		c.excludedDirs = make(map[string]struct{}, len(c.ExcludedDirs))

		for _, dir := range c.ExcludedDirs {
			c.excludedDirs[strings.ToLower(dir)] = struct{}{}
		}

		for _, file := range c.KnownFiles {
			for _, extension := range file.Extensions {
				extension = strings.TrimLeft(extension, ".")
				if file.ExemptFiles {
					c.exemptedExts[extension] = file.Folder
					continue
				}
				c.includedExts[extension] = file.Folder
			}
		}
	})
}

// IsExcludedPath reports whether any path component matches an entry in
// ExcludedDirs (case-insensitive).
func (c *Config) IsExcludedPath(path string) bool {
	c.initCfgMap()
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		if _, ok := c.excludedDirs[strings.ToLower(part)]; ok {
			return true
		}
	}
	return false
}

// ExtSet returns a lowercase lookup map of every extension that belongs to
// the named folder across non-exempt file groups.
func (c *Config) ExtSet(folder string) map[string]bool {
	out := make(map[string]bool)
	for _, kf := range c.KnownFiles {
		if !kf.ExemptFiles && strings.EqualFold(kf.Folder, folder) {
			for _, ext := range kf.Extensions {
				out[strings.ToLower(strings.TrimLeft(ext, "."))] = true
			}
		}
	}
	return out
}

// DestFolders returns the lowercased set of all unique destination folder names
// used by non-exempt file groups.  Useful for skipping already-organized
// subdirectories during a recursive scan.
func (c *Config) DestFolders() map[string]struct{} {
	c.initCfgMap()
	out := make(map[string]struct{})
	for _, folder := range c.includedExts {
		if folder != "" {
			out[strings.ToLower(folder)] = struct{}{}
		}
	}
	if c.UnknownFilesFolder != "" {
		out[strings.ToLower(c.UnknownFilesFolder)] = struct{}{}
	}
	return out
}

// Get returns the destination folder and whether the extension is exempt.
func (c *Config) Get(extension string) (string, bool) {
	c.initCfgMap()

	if folder, ok := c.exemptedExts[extension]; ok {
		return folder, true
	}
	if folder, ok := c.includedExts[extension]; ok {
		return folder, false
	}
	return c.UnknownFilesFolder, false
}
