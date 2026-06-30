package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/arrange/internal/config"
)

// testConfig writes a default config to a temp path and returns it.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.NewConfig(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("testConfig: %v", err)
	}
	return cfg
}

type moveCall struct{ src, dst string }

func TestOrganise(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// src files: name → should it be moved?
	type fileSpec struct {
		name  string
		size  int
		moved bool
	}
	specs := []fileSpec{
		{"movie.mkv", 10, true},
		{"song.mp3", 10, true},
		{"document.pdf", 10, true},
		{"photo.jpg", 10, true},
		{"downloading.crdownload", 10, false}, // exempt
		{"meta.torrent", 10, false},           // exempt
		{"empty.mkv", 0, false},               // zero-size skipped
		{"duplicate (1).mp4", 10, true},       // browser suffix
	}

	for _, s := range specs {
		content := make([]byte, s.size)
		if err := os.WriteFile(filepath.Join(srcDir, s.name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}

	var calls []moveCall
	mockMove := func(src, dst string) error {
		calls = append(calls, moveCall{src, dst})
		return nil
	}

	cfg := testConfig(t)
	results, err := Organise(srcDir, destDir, cfg, mockMove)
	if err != nil {
		t.Fatalf("Organise: %v", err)
	}

	// Count expected moves.
	var wantMoved int
	for _, s := range specs {
		if s.moved {
			wantMoved++
		}
	}

	var moved int
	for _, r := range results {
		if !r.Skipped {
			moved++
		}
	}
	if moved != wantMoved {
		t.Errorf("moved %d files, want %d", moved, wantMoved)
	}
	if len(calls) != wantMoved {
		t.Errorf("mockMove called %d times, want %d", len(calls), wantMoved)
	}

	// The browser suffix file must arrive WITHOUT "(1)" in the destination name.
	for _, c := range calls {
		if strings.Contains(filepath.Base(c.dst), "(1)") {
			t.Errorf("browser suffix not stripped: destination is %q", c.dst)
		}
	}

	// Destination paths must be under destDir, not srcDir.
	for _, c := range calls {
		if !strings.HasPrefix(c.dst, destDir) {
			t.Errorf("destination %q is not under destDir %q", c.dst, destDir)
		}
	}
}

func TestOrganiseEmptyDir(t *testing.T) {
	cfg := testConfig(t)
	results, err := Organise(t.TempDir(), t.TempDir(), cfg, func(_, _ string) error {
		t.Error("move called on empty dir")
		return nil
	})
	if err != nil {
		t.Fatalf("Organise: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestOrganiseCreatorGrouping(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	name := "Tyler.Perrys.Madea.Goes.to.Jail.2009.mkv"
	if err := os.WriteFile(filepath.Join(srcDir, name), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig(t)
	cfg.MediaCreators = []string{"Tyler Perry"}

	var dst string
	_, err := Organise(srcDir, destDir, cfg, func(_, d string) error {
		dst = d
		return nil
	})
	if err != nil {
		t.Fatalf("Organise: %v", err)
	}

	if !strings.Contains(dst, "Tyler Perry") {
		t.Errorf("destination %q does not contain creator folder", dst)
	}
}
