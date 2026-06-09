package fileops

import "os"

func Move(src, dest string) error {
	return os.Rename(src, dest)
}
