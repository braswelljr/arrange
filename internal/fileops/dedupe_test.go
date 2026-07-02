package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSameContent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	a := write("a.txt", "hello world")
	b := write("b.txt", "hello world") // identical
	c := write("c.txt", "hello worlx") // same length, one byte differs
	d := write("d.txt", "hello")       // different length
	e1 := write("e1.bin", "")
	e2 := write("e2.bin", "") // both empty

	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", a, b, true},
		{"same length one byte differs", a, c, false},
		{"different length", a, d, false},
		{"both empty", e1, e2, true},
		{"self", a, a, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SameContent(tc.a, tc.b)
			if err != nil {
				t.Fatalf("SameContent: %v", err)
			}
			if got != tc.want {
				t.Errorf("SameContent = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveDest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}

	// No existing destination → canonical name, not a duplicate.
	dest, dup := ResolveDest(dir, "movie", "mkv", src, false)
	if dup || dest != filepath.Join(dir, "movie.mkv") {
		t.Fatalf("empty dir: got (%q, %v), want canonical non-dup", dest, dup)
	}

	// Existing byte-identical file at the canonical name → duplicate.
	canonical := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(canonical, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	dest, dup = ResolveDest(dir, "movie", "mkv", src, false)
	if !dup || dest != canonical {
		t.Errorf("identical dest: got (%q, %v), want (%q, true)", dest, dup, canonical)
	}

	// keepVersions=true → never dedupe, always version.
	dest, dup = ResolveDest(dir, "movie", "mkv", src, true)
	if dup || dest != filepath.Join(dir, "movie-v1.mkv") {
		t.Errorf("keepVersions: got (%q, %v), want movie-v1.mkv non-dup", dest, dup)
	}

	// Different content at the canonical name → fresh -v1 name.
	if err := os.WriteFile(canonical, []byte("DIFFERENT"), 0600); err != nil {
		t.Fatal(err)
	}
	dest, dup = ResolveDest(dir, "movie", "mkv", src, false)
	if dup || dest != filepath.Join(dir, "movie-v1.mkv") {
		t.Errorf("different content: got (%q, %v), want movie-v1.mkv non-dup", dest, dup)
	}
}
