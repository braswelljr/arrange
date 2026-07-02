package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
	"github.com/braswelljr/arrange/internal/logger"
)

// testOpts returns a CmdOptions wired to a throw-away config in a temp dir.
func testOpts(t *testing.T) *CmdOptions {
	t.Helper()
	buf := &bytes.Buffer{}
	return &CmdOptions{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Log:        logger.New(buf, buf),
	}
}

// writeFile creates a file containing one byte at the given path.
func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestRunE_EmptyDir verifies that running on an empty source produces no error.
func TestRunE_EmptyDir(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	if err := runE(opts, src, src, false, false, false); err != nil {
		t.Fatalf("runE on empty dir: %v", err)
	}
}

// TestRunE_OrganizesFiles checks that common file types land in their category folders.
func TestRunE_OrganizesFiles(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "report.pdf"))
	writeFile(t, filepath.Join(src, "photo.jpg"))
	writeFile(t, filepath.Join(src, "notes.txt"))

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	type check struct{ path string }
	checks := []check{
		{filepath.Join(dest, config.FolderDocuments, "report.pdf")},
		{filepath.Join(dest, config.FolderPictures, "photo.jpg")},
		{filepath.Join(dest, config.FolderDocuments, "notes.txt")},
	}
	for _, c := range checks {
		if _, err := os.Stat(c.path); err != nil {
			t.Errorf("expected file at %s: %v", c.path, err)
		}
	}
}

// TestRunE_VideoHierarchy checks that a TV-show video file lands in a structured directory.
func TestRunE_VideoHierarchy(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	// Standard SxxExx name — parser should resolve title / season / episode / quality.
	writeFile(t, filepath.Join(src, "The.Office.S01E05.720p.mkv"))

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	// Destination must be nested under Videos/<Title>/Season 01/
	seriesDir := filepath.Join(dest, config.FolderVideos, "The Office", "Season 01")
	entries, err := os.ReadDir(seriesDir)
	if err != nil {
		t.Fatalf("expected series directory %s: %v", seriesDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("series directory is empty after runE")
	}
}

// TestRunE_SkipsExemptFiles verifies that exempt extensions (in-progress downloads,
// source code, torrent meta) are never moved.
func TestRunE_SkipsExemptFiles(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	exempt := []string{
		"downloading.crdownload",
		"meta.torrent",
		"main.go",
	}
	for _, name := range exempt {
		writeFile(t, filepath.Join(src, name))
	}

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	for _, name := range exempt {
		// File must still be in src.
		if _, err := os.Stat(filepath.Join(src, name)); err != nil {
			t.Errorf("exempt file %q was moved out of src: %v", name, err)
		}
	}
}

// TestRunE_ExcludeFlag verifies that files named in the exclude list are not moved.
func TestRunE_ExcludeFlag(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "keep.pdf"))
	writeFile(t, filepath.Join(src, "skip.pdf"))

	if err := runE(opts, src, dest, false, false, false, "skip.pdf"); err != nil {
		t.Fatalf("runE: %v", err)
	}

	// "keep.pdf" should have moved to Documents/.
	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "keep.pdf")); err != nil {
		t.Errorf("keep.pdf not found in dest: %v", err)
	}
	// "skip.pdf" must still be in src.
	if _, err := os.Stat(filepath.Join(src, "skip.pdf")); err != nil {
		t.Errorf("skip.pdf was moved despite being excluded: %v", err)
	}
}

// TestRunE_Recursive checks that a nested subdirectory is also scanned.
func TestRunE_Recursive(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "top.pdf"))
	writeFile(t, filepath.Join(src, "subdir", "nested.jpg"))

	if err := runE(opts, src, dest, true, false, false); err != nil {
		t.Fatalf("runE recursive: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "top.pdf")); err != nil {
		t.Errorf("top.pdf not organized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, config.FolderPictures, "nested.jpg")); err != nil {
		t.Errorf("nested.jpg not organized: %v", err)
	}
}

// TestRunE_ZeroByteSkipped ensures zero-byte files are never moved.
func TestRunE_ZeroByteSkipped(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	path := filepath.Join(src, "empty.pdf")
	if err := os.WriteFile(path, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Error("zero-byte file was moved")
	}
}

// TestDirLocker_Concurrent verifies that concurrent locks on the same directory
// are mutually exclusive (run with -race to catch data races).
func TestDirLocker_Concurrent(t *testing.T) {
	locker := new(dirLocker)
	dir := filepath.Join(t.TempDir(), "shared")

	const goroutines = 64
	counter := 0
	var wg sync.WaitGroup

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locker.Lock(dir)
			counter++
			locker.Unlock(dir)
		}()
	}
	wg.Wait()

	if counter != goroutines {
		t.Errorf("counter = %d, want %d", counter, goroutines)
	}
}

// TestDirLocker_IndependentDirs verifies that locks on different directories
// do not block each other.
func TestDirLocker_IndependentDirs(t *testing.T) {
	locker := new(dirLocker)
	root := t.TempDir()

	const goroutines = 16
	var wg sync.WaitGroup
	done := make(chan struct{}, goroutines)

	for i := range goroutines {
		dir := filepath.Join(root, "d"+string(rune('a'+i)))
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			locker.Lock(d)
			done <- struct{}{}
			locker.Unlock(d)
		}(dir)
	}

	wg.Wait()
	close(done)

	if n := len(done); n != goroutines {
		t.Errorf("got %d completions, want %d", n, goroutines)
	}
}

// TestOrganiseFile_Exempt confirms that organiseFile returns nil without moving
// an exempt file.
func TestOrganiseFile_Exempt(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	name := "script.go"
	writeFile(t, filepath.Join(src, name))

	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	f := fakeSmartFile(t, src, name, "go", 1)
	if err := organiseFile(opts, cfg, new(dirLocker), dest, f); err != nil {
		t.Fatalf("organiseFile returned unexpected error: %v", err)
	}

	// File must still be in src.
	if _, err := os.Stat(filepath.Join(src, name)); err != nil {
		t.Error("exempt file was moved")
	}
}

// TestOrganiseFile_Document verifies the destination path for a PDF.
func TestOrganiseFile_Document(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "report.pdf"))

	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	f := fakeSmartFile(t, src, "report.pdf", "pdf", 1)
	if err := organiseFile(opts, cfg, new(dirLocker), dest, f); err != nil {
		t.Fatalf("organiseFile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "report.pdf")); err != nil {
		t.Errorf("report.pdf not at expected dest: %v", err)
	}
}

// TestRunE_SubtitleFollowsVideo verifies that a subtitle sitting beside a video
// is moved into the same destination folder and renamed to match the video,
// preserving its language tag.
func TestRunE_SubtitleFollowsVideo(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "The.Office.S01E05.720p.mkv"))
	writeFile(t, filepath.Join(src, "The.Office.S01E05.720p.en.srt"))
	writeFile(t, filepath.Join(src, "The.Office.S01E05.720p.srt")) // no language tag

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	seasonDir := filepath.Join(dest, config.FolderVideos, "The Office", "Season 01")
	want := []string{
		"The Office S01E05 [720p].mkv",
		"The Office S01E05 [720p].en.srt",
		"The Office S01E05 [720p].srt",
	}
	for _, name := range want {
		if _, err := os.Stat(filepath.Join(seasonDir, name)); err != nil {
			t.Errorf("expected %s beside the video: %v", name, err)
		}
	}

	// The subtitle must NOT have been routed to the standalone Subtitles folder.
	if _, err := os.Stat(filepath.Join(dest, config.FolderSubtitles)); err == nil {
		t.Error("subtitle was routed to Subtitles/ instead of following its video")
	}
}

// TestRunE_OrphanSubtitle verifies that a subtitle with no matching video is
// organized on its own into the Subtitles folder.
func TestRunE_OrphanSubtitle(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "loose.srt"))

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, config.FolderSubtitles, "loose.srt")); err != nil {
		t.Errorf("orphan subtitle not organized into Subtitles/: %v", err)
	}
}

// TestRunE_DryRunMovesNothing verifies that --dry-run leaves every file in place.
func TestRunE_DryRunMovesNothing(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	writeFile(t, filepath.Join(src, "report.pdf"))

	if err := runE(opts, src, dest, false, true, false); err != nil {
		t.Fatalf("runE dry-run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(src, "report.pdf")); err != nil {
		t.Error("dry-run moved report.pdf out of src")
	}
	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "report.pdf")); err == nil {
		t.Error("dry-run created a file in dest")
	}
}

// TestRunE_DuplicateRemoved verifies that a byte-identical file whose copy is
// already organized is removed from the source instead of creating a -vN copy.
func TestRunE_DuplicateRemoved(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	// Pre-existing organized copy.
	organized := filepath.Join(dest, config.FolderDocuments, "report.pdf")
	if err := os.MkdirAll(filepath.Dir(organized), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(organized, []byte("identical"), 0600); err != nil {
		t.Fatal(err)
	}
	// Same-content file waiting in src.
	dup := filepath.Join(src, "report.pdf")
	if err := os.WriteFile(dup, []byte("identical"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	// Source duplicate must be gone, and no -v1 copy created.
	if _, err := os.Stat(dup); !os.IsNotExist(err) {
		t.Error("duplicate source was not removed")
	}
	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "report-v1.pdf")); err == nil {
		t.Error("a -v1 copy was created for an identical duplicate")
	}
}

// TestRunE_KeepDuplicates verifies that --keep-duplicates versions an identical
// file as -v1 instead of removing it.
func TestRunE_KeepDuplicates(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	organized := filepath.Join(dest, config.FolderDocuments, "report.pdf")
	if err := os.MkdirAll(filepath.Dir(organized), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(organized, []byte("identical"), 0600); err != nil {
		t.Fatal(err)
	}
	dup := filepath.Join(src, "report.pdf")
	if err := os.WriteFile(dup, []byte("identical"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := runE(opts, src, dest, false, false, true); err != nil {
		t.Fatalf("runE: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "report-v1.pdf")); err != nil {
		t.Errorf("--keep-duplicates did not create a -v1 copy: %v", err)
	}
}

// TestRunE_DifferentContentVersions verifies that a same-named but different
// file is versioned (not removed) so no data is lost.
func TestRunE_DifferentContentVersions(t *testing.T) {
	opts := testOpts(t)
	src := t.TempDir()
	dest := t.TempDir()

	organized := filepath.Join(dest, config.FolderDocuments, "report.pdf")
	if err := os.MkdirAll(filepath.Dir(organized), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(organized, []byte("ORIGINAL"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "report.pdf"), []byte("DIFFERENT"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := runE(opts, src, dest, false, false, false); err != nil {
		t.Fatalf("runE: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, config.FolderDocuments, "report-v1.pdf")); err != nil {
		t.Errorf("different-content file was not versioned as -v1: %v", err)
	}
}

// fakeSmartFile builds a SmartFile pointing at an existing file in dir.
func fakeSmartFile(t *testing.T, dir, name, ext string, size int64) *fileops.SmartFile {
	t.Helper()
	return &fileops.SmartFile{Name: name, Path: filepath.Join(dir, name), Ext: ext, Size: size}
}
