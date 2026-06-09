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

	Path string

	initOnce     sync.Once
	exemptedExts map[string]string
	includedExts map[string]string
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
