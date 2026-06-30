package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

var stripCases = []struct {
	input string
	want  string
}{
	{"document (1)", "document"},
	{"video (2)", "video"},
	{"Breaking Bad (3)", "Breaking Bad"},
	{"photo", "photo"},
	{"file (10)", "file"},
	{"already clean", "already clean"},
	{"(1)", ""},
	// mid-string parenthesis must NOT be stripped
	{"a (1) b", "a (1) b"},
	// spaces around the suffix are trimmed
	{"file  (1)  ", "file"},
}

func TestStripBrowserSuffix(t *testing.T) {
	for _, c := range stripCases {
		t.Run(c.input, func(t *testing.T) {
			got := StripBrowserSuffix(c.input)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestSafeDestPath(t *testing.T) {
	dir := t.TempDir()

	t.Run("no collision returns base path", func(t *testing.T) {
		got := SafeDestPath(dir, "movie", "mkv")
		want := filepath.Join(dir, "movie.mkv")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("one collision returns v1", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(dir, "doc.pdf"), []byte{1}, 0600); err != nil {
			t.Fatal(err)
		}
		got := SafeDestPath(dir, "doc", "pdf")
		want := filepath.Join(dir, "doc-v1.pdf")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("two collisions returns v2", func(t *testing.T) {
		for _, name := range []string{"photo.jpg", "photo-v1.jpg"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte{1}, 0600); err != nil {
				t.Fatal(err)
			}
		}
		got := SafeDestPath(dir, "photo", "jpg")
		want := filepath.Join(dir, "photo-v2.jpg")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
