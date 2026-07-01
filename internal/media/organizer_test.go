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
		{"document.pdf", 10, false},           // non-media — skipped by Organize
		{"photo.jpg", 10, false},              // non-media — skipped by Organize
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
	results, err := Organize(srcDir, destDir, cfg, mockMove)
	if err != nil {
		t.Fatalf("Organize: %v", err)
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
	results, err := Organize(t.TempDir(), t.TempDir(), cfg, func(_, _ string) error {
		t.Error("move called on empty dir")
		return nil
	})
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestOrganizeOnlyMediaFiles verifies FR-MEDIA-02: the media command must
// ignore documents, pictures, and any other non-video/non-audio extension.
func TestOrganizeOnlyMediaFiles(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	nonMedia := []string{"report.pdf", "photo.png", "notes.txt", "archive.zip"}
	media := []string{"episode.mkv", "track.mp3"}

	for _, name := range append(nonMedia, media...) {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	var moved []string
	cfg := testConfig(t)
	results, err := Organize(srcDir, destDir, cfg, func(src, _ string) error {
		moved = append(moved, filepath.Base(src))
		return nil
	})
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	// Only the two media files should appear in results.
	if len(results) != len(media) {
		t.Errorf("got %d results, want %d (media only)", len(results), len(media))
	}
	if len(moved) != len(media) {
		t.Errorf("move called %d times, want %d", len(moved), len(media))
	}
	for _, nm := range nonMedia {
		for _, m := range moved {
			if m == nm {
				t.Errorf("non-media file %q was moved — should have been skipped", nm)
			}
		}
	}
}

// TestOrganizeResultInfo verifies that Result.Info carries the full parsed MediaInfo.
func TestOrganizeResultInfo(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	name := "Breaking.Bad.S03E07.1080p.BluRay.mkv"
	if err := os.WriteFile(filepath.Join(srcDir, name), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig(t)
	results, err := Organize(srcDir, destDir, cfg, func(_, _ string) error { return nil })
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	info := results[0].Info
	if info.Type != TypeTVSeries {
		t.Errorf("Type = %v, want TypeTVSeries", info.Type)
	}
	if info.Season != 3 {
		t.Errorf("Season = %d, want 3", info.Season)
	}
	if info.Episode != 7 {
		t.Errorf("Episode = %d, want 7", info.Episode)
	}
	if info.Quality == "" {
		t.Error("Quality is empty, want a non-empty quality tag")
	}
	if info.Title == "" {
		t.Error("Title is empty, want a non-empty title")
	}
	if info.OrigPath == "" {
		t.Error("OrigPath is empty")
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
	_, err := Organize(srcDir, destDir, cfg, func(_, d string) error {
		dst = d
		return nil
	})
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	if !strings.Contains(dst, "Tyler Perry") {
		t.Errorf("destination %q does not contain creator folder", dst)
	}
}
