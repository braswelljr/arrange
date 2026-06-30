//go:build !windows

package config

import (
	"os"
	"path/filepath"
)

// Path returns the default config file path, following the XDG Base Directory
// spec on Linux and the conventional ~/.config location on macOS.
func Path() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "arrange", "config.json")
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = "~"
	}
	return filepath.Join(home, ".config", "arrange", "config.json")
}
