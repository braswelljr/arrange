package fileops

import (
	"errors"
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
		// Close and remove the partial destination before returning. Join any
		// cleanup failure so the original copy error is never silently lost.
		cerr := fmt.Errorf("copy %s to %s: %w", src, dst, err)
		if closeErr := out.Close(); closeErr != nil {
			cerr = errors.Join(cerr, fmt.Errorf("close partial destination %s: %w", dst, closeErr))
		}
		if rmErr := os.Remove(dst); rmErr != nil {
			cerr = errors.Join(cerr, fmt.Errorf("remove partial destination %s: %w", dst, rmErr))
		}
		return cerr
	}

	if err := out.Close(); err != nil {
		rerr := fmt.Errorf("close destination %s: %w", dst, err)
		if rmErr := os.Remove(dst); rmErr != nil {
			rerr = errors.Join(rerr, fmt.Errorf("remove destination %s: %w", dst, rmErr))
		}
		return rerr
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

	if err := removeSource(src); err != nil {
		return fmt.Errorf("remove source %s: %w", src, err)
	}
	return nil
}

// removeSource deletes src, retrying once after restoring write permission when
// the first attempt is refused (a read-only source file on Unix, or the
// read-only attribute on Windows).
func removeSource(src string) error {
	err := os.Remove(src)
	if err == nil || !errors.Is(err, os.ErrPermission) {
		return err
	}
	// Best-effort: make the file writable, then try once more.
	if chmodErr := os.Chmod(src, 0o600); chmodErr != nil {
		return err // return the original removal error, not the chmod error
	}
	return os.Remove(src)
}