package fileops

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// reBrowserSuffix matches the duplicate counter browsers append when a
// filename already exists: "document (1).pdf", "video (2).mkv", etc.
var reBrowserSuffix = regexp.MustCompile(`\s*\(\d+\)\s*$`)

// StripBrowserSuffix removes a browser-generated duplicate suffix from a
// filename stem (no extension).
//
//	"document (1)" → "document"
//	"photo"        → "photo"  (unchanged)
func StripBrowserSuffix(stem string) string {
	return strings.TrimSpace(reBrowserSuffix.ReplaceAllString(stem, ""))
}

// SafeDestPath returns a unique destination path for a file with the given
// stem and extension inside dir.  If dir/stem.ext already exists it appends
// -v1, -v2, … until a free slot is found.
func SafeDestPath(dir, stem, ext string) string {
	candidate := filepath.Join(dir, stem+"."+ext)
	base := filepath.Join(dir, stem)
	for i := 1; FileExists(candidate); i++ {
		candidate = fmt.Sprintf("%s-v%d.%s", base, i, ext)
	}
	return candidate
}
