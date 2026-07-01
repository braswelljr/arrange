package media

import (
	"path/filepath"
	"strings"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
)

// Result describes the outcome for a single file processed by Organize.
type Result struct {
	Info     *MediaInfo // full parsed metadata; OrigPath is always set
	DestPath string     // absolute destination path chosen for the file
	Skipped  bool       // true when the file was not moved (see Reason)
	Reason   string     // human-readable explanation when Skipped is true
}

// Organize scans srcDir for video and audio files and moves each one into a
// structured media hierarchy under destDir using the filename parser.
//
// Only extensions that map to the Videos or Audio category are processed;
// all other files (documents, pictures, exempt types, zero-byte files) are
// silently skipped so the command stays focused on media organization.
// Creator grouping and quality-tag formatting all come from cfg.
//
// move is injected so callers (and tests) can replace fileops.Move without
// touching the filesystem.
//
// @param srcDir   directory to scan (top level only, non-recursive)
// @param destDir  root under which the media hierarchy is written
// @param cfg      loaded config supplying extension maps and creator names
// @param move     function called to relocate each file (src, dst string) error
// @returns        slice of per-file results and any directory-scan error
func Organize(
	srcDir, destDir string,
	cfg *config.Config,
	move func(src, dst string) error,
) ([]*Result, error) {
	mediaSet := buildMediaSet(cfg)
	creators := cfg.MediaCreators

	files, err := fileops.ScanDir(srcDir)
	if err != nil {
		return nil, err
	}

	var results []*Result

	for _, f := range *files {
		// Skip zero-byte files — they are incomplete or placeholder.
		if f.Size == 0 {
			continue
		}

		// Skip exempt extensions (in-progress downloads, source code, etc.).
		_, exempt := cfg.Get(f.Ext)
		if exempt {
			continue
		}

		// FR-MEDIA-02: the media command handles video and audio only.
		if !mediaSet[f.Ext] {
			continue
		}

		cleanStem := fileops.StripBrowserSuffix(strings.TrimSuffix(f.Name, filepath.Ext(f.Name)))

		parsed := ParseName(cleanStem + "." + f.Ext)
		parsed.OrigPath = f.Path
		creator := parsed.CreatorMatch(creators)

		destRelDir := parsed.DestDir(creator)
		destName := parsed.DestName()

		fullDestDir := filepath.Join(destDir, destRelDir)
		stem := strings.TrimSuffix(destName, "."+f.Ext)
		destPath := fileops.SafeDestPath(fullDestDir, stem, f.Ext)

		// Store the full parsed MediaInfo so callers get title, type, quality, etc.
		r := &Result{Info: parsed, DestPath: destPath}

		if err := fileops.EnsureDir(fullDestDir); err != nil {
			r.Skipped = true
			r.Reason = err.Error()
			results = append(results, r)
			continue
		}

		if err := move(f.Path, destPath); err != nil {
			r.Skipped = true
			r.Reason = err.Error()
		}

		results = append(results, r)
	}

	return results, nil
}

// buildMediaSet returns the union of all video and audio extensions from cfg.
func buildMediaSet(cfg *config.Config) map[string]bool {
	out := make(map[string]bool)
	for ext := range cfg.ExtSet(config.FolderVideos) {
		out[ext] = true
	}
	for ext := range cfg.ExtSet(config.FolderAudio) {
		out[ext] = true
	}
	return out
}
