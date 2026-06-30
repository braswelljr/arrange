//go:build windows

package config

import (
	"os"
	"path/filepath"
)

// Path returns the default config file path on Windows.
// Uses %APPDATA% (C:\Users\<user>\AppData\Roaming) with %USERPROFILE% as
// fallback if APPDATA is not set.
func Path() string {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir = os.Getenv("USERPROFILE")
	}
	return filepath.Join(dir, "arrange", "config.json")
}
