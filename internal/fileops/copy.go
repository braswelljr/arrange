package fileops

import (
	"fmt"
	"io"
	"os"
)

// copyThenRemove copies src to dst and removes src on success.
// Used as a cross-device fallback when os.Rename fails with EXDEV.
func copyThenRemove(src, dst string) error {
	srcInfo, _ := os.Stat(src)

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		// Close and remove the partial destination before returning the copy error.
		_ = out.Close()
		_ = os.Remove(dst)
		return fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("close destination %s: %w", dst, err)
	}

	// Preserve source permissions where the OS supports it.
	if srcInfo != nil {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("chmod %s: %w", dst, err)
		}
	}

	// Close src explicitly before removal — Windows cannot remove an open file.
	// The deferred close is a harmless no-op after this.
	if err := in.Close(); err != nil {
		return fmt.Errorf("close source %s: %w", src, err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source %s: %w", src, err)
	}
	return nil
}
