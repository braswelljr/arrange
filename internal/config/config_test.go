package config

import (
	"path/filepath"
	"testing"
)

func newTestConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := NewConfig(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	return cfg
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestGetKnownExtension(t *testing.T) {
	cfg := newTestConfig(t)

	cases := []struct {
		ext    string
		folder string
	}{
		{"mp4", FolderVideos},
		{"mkv", FolderVideos},
		{"mp3", FolderAudio},
		{"flac", FolderAudio},
		{"pdf", FolderDocuments},
		{"docx", FolderDocuments},
		{"jpg", FolderPictures},
		{"png", FolderPictures},
		{"epub", FolderEBooks},
		{"zip", FolderArchive},
		{"ttf", FolderFonts},
		{"exe", FolderApps},
		{"iso", FolderDiskImages},
		{"sqlite", FolderDatabase},
		{"srt", FolderSubtitles},
		{"fig", FolderDesign},
		{"stl", FolderModels3D},
	}

	for _, c := range cases {
		t.Run(c.ext, func(t *testing.T) {
			folder, exempt := cfg.Get(c.ext)
			if exempt {
				t.Errorf("Get(%q) returned exempt=true, want false", c.ext)
			}
			if folder != c.folder {
				t.Errorf("Get(%q) = %q, want %q", c.ext, folder, c.folder)
			}
		})
	}
}

func TestGetExemptExtension(t *testing.T) {
	cfg := newTestConfig(t)

	exempt := []string{
		// in-progress downloads
		"crdownload", "part", "download", "opdownload",
		"!qb", "!ut", "!bt", "aria2",
		// torrent meta
		"torrent",
		// source code
		"go", "py", "js", "ts",
	}

	for _, ext := range exempt {
		t.Run(ext, func(t *testing.T) {
			_, isExempt := cfg.Get(ext)
			if !isExempt {
				t.Errorf("Get(%q) exempt=false, want true", ext)
			}
		})
	}
}

func TestGetUnknownExtension(t *testing.T) {
	cfg := newTestConfig(t)

	folder, exempt := cfg.Get("xyz123unknown")
	if exempt {
		t.Error("unknown extension should not be exempt")
	}
	if folder != FolderOther {
		t.Errorf("got %q, want %q", folder, FolderOther)
	}
}

// ── ExtSet ────────────────────────────────────────────────────────────────────

func TestExtSet(t *testing.T) {
	cfg := newTestConfig(t)

	videos := cfg.ExtSet(FolderVideos)
	for _, ext := range []string{"mp4", "mkv", "avi", "mov"} {
		if !videos[ext] {
			t.Errorf("ExtSet(Videos) missing %q", ext)
		}
	}
	// Exempt extensions must not appear in the set.
	if videos["crdownload"] || videos["torrent"] || videos["go"] {
		t.Error("exempt extension appeared in ExtSet(Videos)")
	}

	audio := cfg.ExtSet(FolderAudio)
	for _, ext := range []string{"mp3", "flac", "wav", "aac"} {
		if !audio[ext] {
			t.Errorf("ExtSet(Audio) missing %q", ext)
		}
	}
	// Video extensions must not bleed into audio set.
	if audio["mp4"] || audio["mkv"] {
		t.Error("video extension appeared in ExtSet(Audio)")
	}
}

// ── IsExcludedPath ────────────────────────────────────────────────────────────

func TestIsExcludedPath(t *testing.T) {
	// Default config: nothing is excluded.
	cfg := newTestConfig(t)
	if cfg.IsExcludedPath("/home/user/Downloads/file.mkv") {
		t.Error("default config should not exclude any path")
	}

	// Set ExcludedDirs before the first lazy init call so initCfgMap picks them up.
	cfg2 := newTestConfig(t)
	cfg2.ExcludedDirs = []string{"node_modules", ".git"}

	if !cfg2.IsExcludedPath("/project/node_modules/lodash/index.js") {
		t.Error("node_modules path should be excluded")
	}
	if !cfg2.IsExcludedPath("/project/.git/config") {
		t.Error(".git path should be excluded")
	}
	if cfg2.IsExcludedPath("/project/src/main.go") {
		t.Error("normal path should not be excluded")
	}
}
