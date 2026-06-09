package fileops

import (
	"os"
)

type SmartFile struct {
	Name    string
	Path    string
	Ext     string
	NewPath string
	Size    int64
}

type SmartFiles []*SmartFile

func (s *SmartFiles) Len() int {
	return len(*s)
}

func DirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
