//go:build !windows

package fileops

import (
	"errors"
	"os"
	"syscall"
)

// Move moves src to dst within the same filesystem (fast, atomic).
// Falls back to a copy-then-delete only when src and dst live on different
// filesystems (EXDEV) — e.g. across mount points on Linux/macOS. Any other
// rename failure is returned verbatim so the real cause is never masked by a
// second, unrelated copy error.
func Move(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.EXDEV) {
		return copyThenRemove(src, dst)
	}
	return err
}