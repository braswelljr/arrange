//go:build windows

package fileops

import "os"

// Move moves src to dst within the same drive (fast, uses MoveFileEx).
// Falls back to a copy-then-delete when src and dst are on different drives.
func Move(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	return copyThenRemove(src, dst)
}
