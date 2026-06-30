package fileops

import (
	"os"
	"path/filepath"
	"strings"
)

// SmartFile is a lightweight descriptor for a single file entry.
type SmartFile struct {
	Name string
	Path string
	Ext  string // lowercase, no leading dot
	Size int64
}

// SmartFiles is a slice of SmartFile pointers with a convenience Len method.
type SmartFiles []*SmartFile

func (s *SmartFiles) Len() int { return len(*s) }

// DirExists reports whether path exists and is a directory.
func DirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// FileExists reports whether path exists (file or directory).
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates dir and all parents if they do not already exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, os.ModePerm)
}

// ScanDir reads the top level of dir and returns every non-directory entry.
// Extensions are lowercased and stripped of their leading dot.
func ScanDir(dir string) (*SmartFiles, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files SmartFiles
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		name := e.Name()
		files = append(files, &SmartFile{
			Name: name,
			Path: filepath.Join(dir, name),
			Ext:  strings.ToLower(strings.TrimLeft(filepath.Ext(name), ".")),
			Size: info.Size(),
		})
	}
	return &files, nil
}
