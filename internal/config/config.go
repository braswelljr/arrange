package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"

	"github.com/braswelljr/arrange/internal/fileops"
)

var (
	cfg       *Config
	pathCache string
	cfgMu     sync.RWMutex
)

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

type FileExt struct {
	Extensions  []string `json:"extensions"`
	Folder      string   `json:"folder"`
	ExemptFiles bool     `json:"exempt_files"`
}

// Path returns config file path
func Path() string {
	usrConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if usrConfigHome == "" {
		usrConfigHome = os.Getenv("HOME")
		if usrConfigHome == "" {
			usrConfigHome, _ = homedir.Expand("~/.config")
		} else {
			usrConfigHome = filepath.Join(usrConfigHome, ".config")
		}
	}
	return filepath.Join(usrConfigHome, "arrange", "config.json")
}

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

	// ExcludedDirs: no backfill needed — the new default is intentionally
	// empty, so old configs that still list app dirs keep their behaviour
	// and new configs start with nothing excluded.

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

// IsExcludedPath returns true if any path component matches an entry in
// ExcludedDirs (case-insensitive). The default list is empty, so this only
// triggers when the user adds entries to their config.
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
// the named folder in the non-exempt file groups. Callers import the folder
// name constants (e.g. config.FolderVideos) so no raw strings are needed.
//
// Example:
//
//	videos := cfg.ExtSet(config.FolderVideos)
//	if videos["mkv"] { … }
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

// Get returns the folder as string and a boolean val indicating whether the
// ext should be excluded or not.
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
