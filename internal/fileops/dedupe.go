package fileops

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// compareChunk is the read size used when comparing two files byte-for-byte.
const compareChunk = 64 * 1024

// SameContent reports whether files a and b are byte-for-byte identical.
// It short-circuits on differing sizes, so mismatched files are rejected
// without reading their contents.
func SameContent(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if ai.Size() != bi.Size() {
		return false, nil
	}
	if ai.Size() == 0 {
		return true, nil // two empty files
	}

	fa, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer func() { _ = fa.Close() }()
	fb, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer func() { _ = fb.Close() }()

	bufA := make([]byte, compareChunk)
	bufB := make([]byte, compareChunk)
	for {
		na, ea := io.ReadFull(fa, bufA)
		nb, eb := io.ReadFull(fb, bufB)
		if na != nb || !bytes.Equal(bufA[:na], bufB[:nb]) {
			return false, nil
		}
		// Sizes are equal, so both readers reach the end together.
		if ea == io.EOF || ea == io.ErrUnexpectedEOF {
			return true, nil
		}
		if ea != nil {
			return false, ea
		}
		if eb != nil && eb != io.ErrUnexpectedEOF {
			return false, eb
		}
	}
}

// ResolveDest chooses the destination path for a file with the given stem and
// extension inside dir, comparing against any file already occupying that name.
//
// It returns (path, duplicate):
//   - no file occupies the canonical name → (dir/stem.ext, false)
//   - an existing file (the canonical name or a prior "-vN") is byte-identical
//     to srcPath → (that existing path, true); the caller should treat srcPath
//     as a redundant copy instead of writing a new version
//   - otherwise → (next free "-vN" name, false)
//
// When keepVersions is true the content check is skipped and it always versions
// on collision, exactly like SafeDestPath.
func ResolveDest(dir, stem, ext, srcPath string, keepVersions bool) (string, bool) {
	candidate := filepath.Join(dir, stem+"."+ext)
	base := filepath.Join(dir, stem)
	for i := 1; FileExists(candidate); i++ {
		if !keepVersions {
			if same, err := SameContent(srcPath, candidate); err == nil && same {
				return candidate, true
			}
		}
		candidate = fmt.Sprintf("%s-v%d.%s", base, i, ext)
	}
	return candidate, false
}
