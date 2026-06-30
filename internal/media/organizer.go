package media

import (
	"path/filepath"
	"strings"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
)

// Result describes the outcome for a single file.
type Result struct {
	Info     *MediaInfo
	DestPath string
	Skipped  bool
	Reason   string
}

// Organise scans srcDir and moves every recognised file into a structured
// hierarchy under destDir.  Extension lookup, creator grouping, and
// in-progress exemptions all come from cfg.
//
// move is injected so tests can replace fileops.Move without touching the FS.
func Organise(
	srcDir, destDir string,
	cfg *config.Config,
	move func(src, dst string) error,
) ([]*Result, error) {
	media := buildMediaSet(cfg)
	creators := cfg.MediaCreators

	files, err := fileops.ScanDir(srcDir)
	if err != nil {
		return nil, err
	}

	var results []*Result

	for _, f := range *files {
		if f.Size == 0 {
			continue
		}

		folder, exempt := cfg.Get(f.Ext)
		if exempt {
			continue
		}

		cleanStem := fileops.StripBrowserSuffix(strings.TrimSuffix(f.Name, filepath.Ext(f.Name)))

		var destRelDir, destName string

		if media[f.Ext] {
			parsed := ParseName(cleanStem + "." + f.Ext)
			parsed.OrigPath = f.Path
			creator := parsed.CreatorMatch(creators)
			destRelDir = parsed.DestDir(creator)
			destName = parsed.DestName()
		} else {
			if folder == "" {
				folder = config.FolderOther
			}
			destRelDir = folder
			destName = SanitizeFilename(cleanStem) + "." + f.Ext
		}

		fullDestDir := filepath.Join(destDir, destRelDir)
		destPath := fileops.SafeDestPath(fullDestDir, strings.TrimSuffix(destName, "."+f.Ext), f.Ext)

		r := &Result{Info: &MediaInfo{OrigPath: f.Path, Ext: f.Ext}, DestPath: destPath}

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
