package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")

	if FileExists(f) {
		t.Error("want false for missing file")
	}
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if !FileExists(f) {
		t.Error("want true for existing file")
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()

	if !DirExists(dir) {
		t.Error("want true for existing dir")
	}
	if DirExists(filepath.Join(dir, "nope")) {
		t.Error("want false for missing dir")
	}
	// A file must not satisfy DirExists.
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if DirExists(f) {
		t.Error("want false when path is a file")
	}
}

func TestEnsureDir(t *testing.T) {
	nested := filepath.Join(t.TempDir(), "a", "b", "c")

	if err := EnsureDir(nested); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if !DirExists(nested) {
		t.Error("directory not created")
	}
	// Calling again must not error (idempotent).
	if err := EnsureDir(nested); err != nil {
		t.Errorf("second EnsureDir: %v", err)
	}
}

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	names := []string{"a.txt", "b.mp4", "c.PDF"} // PDF in upper-case
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// A subdirectory must not appear in the result.
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0750); err != nil {
		t.Fatal(err)
	}

	files, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	if files.Len() != 3 {
		t.Fatalf("got %d files, want 3", files.Len())
	}

	extSet := make(map[string]bool, 3)
	for _, f := range *files {
		// Extensions must be lowercase and without leading dot.
		if f.Ext != strings.ToLower(f.Ext) || strings.HasPrefix(f.Ext, ".") {
			t.Errorf("bad extension %q for %s", f.Ext, f.Name)
		}
		extSet[f.Ext] = true
	}
	for _, want := range []string{"txt", "mp4", "pdf"} {
		if !extSet[want] {
			t.Errorf("missing extension %q", want)
		}
	}
}
