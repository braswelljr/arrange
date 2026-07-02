package media

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
)

// Result describes the outcome for a single file processed by Organize.
type Result struct {
	Info      *MediaInfo // full parsed metadata; OrigPath is always set
	DestPath  string     // absolute destination path chosen for the file
	Skipped   bool       // true when the file was not moved (see Reason)
	Reason    string     // human-readable explanation when Skipped is true
	Duplicate bool       // true when a byte-identical copy already existed at DestPath
}

// primary is a media file together with everything needed to place it.
type primary struct {
	file     *fileops.SmartFile
	info     *MediaInfo
	destDir  string        // absolute destination directory
	destStem string        // destination filename without extension
	sidecars []subtitleRef // subtitles that must travel with this file
}

// subtitleRef pairs a subtitle file with its language / flag tag.
type subtitleRef struct {
	file *fileops.SmartFile
	tag  string
}

// Organize scans srcDir for video and audio files and moves each one into a
// structured media hierarchy under destDir using the filename parser.
//
// Only extensions that map to the Videos or Audio category are treated as
// primaries; subtitle files (the Subtitles category) that sit beside a video
// and share its name are moved along with it and renamed to match, so a video
// and its .srt never drift apart. All other files (documents, pictures, exempt
// types, zero-byte and symlinked files, orphan subtitles) are skipped so the
// command stays focused on media organization.
//
// move is injected so callers (and tests) can replace fileops.Move without
// touching the filesystem. When dryRun is true no directories are created and
// move is never called; results describe what would happen.
//
// @param srcDir   directory to scan (top level only, non-recursive)
// @param destDir  root under which the media hierarchy is written
// @param cfg      loaded config supplying extension maps and creator names
// @param move     function called to relocate each file (src, dst string) error
// @param dryRun   when true, plan the moves without changing the filesystem
// @returns        slice of per-file results and any directory-scan error
func Organize(
	srcDir, destDir string,
	cfg *config.Config,
	move func(src, dst string) error,
	dryRun bool,
	keepDuplicates bool,
) ([]*Result, error) {
	mediaSet := buildMediaSet(cfg)
	subSet := cfg.ExtSet(config.FolderSubtitles)
	creators := cfg.MediaCreators

	files, err := fileops.ScanDir(srcDir)
	if err != nil {
		return nil, err
	}

	var primaries []*primary
	var subs []*fileops.SmartFile

	for _, f := range *files {
		// Skip zero-byte (incomplete), symlinked, and hidden/system dotfiles.
		if f.Size == 0 || f.Symlink || strings.HasPrefix(f.Name, ".") {
			continue
		}
		// Skip exempt extensions (in-progress downloads, source code, etc.).
		if _, exempt := cfg.Get(f.Ext); exempt {
			continue
		}

		switch {
		case mediaSet[f.Ext]:
			stem := fileops.StripBrowserSuffix(strings.TrimSuffix(f.Name, filepath.Ext(f.Name)))
			parsed := ParseName(stem + "." + f.Ext)
			parsed.OrigPath = f.Path
			parsed.Title = cfg.CanonicalTitle(parsed.Title)
			creator := parsed.CreatorMatch(creators)
			destName := parsed.DestName()
			primaries = append(primaries, &primary{
				file:     f,
				info:     parsed,
				destDir:  filepath.Join(destDir, parsed.DestDir(creator)),
				destStem: strings.TrimSuffix(destName, "."+f.Ext),
			})
		case subSet[f.Ext]:
			subs = append(subs, f) // matched to a primary below
			// default: non-media file — ignored by the media command.
		}
	}

	attachSubtitles(primaries, subs)

	var results []*Result
	for _, p := range primaries {
		results = append(results, placePrimary(p, move, dryRun, keepDuplicates)...)
	}

	return results, nil
}

// attachSubtitles links each subtitle to the best-matching primary in the same
// directory (longest media stem wins), preserving its language / flag tag.
// Orphan subtitles with no matching primary are dropped.
func attachSubtitles(primaries []*primary, subs []*fileops.SmartFile) {
	for _, s := range subs {
		subStem := fileops.StripBrowserSuffix(strings.TrimSuffix(s.Name, filepath.Ext(s.Name)))
		subDir := filepath.Dir(s.Path)

		var best *primary
		var bestTag string
		for _, p := range primaries {
			if filepath.Dir(p.file.Path) != subDir {
				continue
			}
			pStem := fileops.StripBrowserSuffix(strings.TrimSuffix(p.file.Name, filepath.Ext(p.file.Name)))
			tag, ok := MatchSidecar(pStem, subStem)
			if !ok {
				continue
			}
			if best == nil || len(p.file.Name) > len(best.file.Name) {
				best, bestTag = p, tag
			}
		}
		if best != nil {
			best.sidecars = append(best.sidecars, subtitleRef{file: s, tag: bestTag})
		}
	}
}

// placePrimary moves a primary and its subtitles, returning one Result per
// file. Byte-identical files whose copy already exists at the destination are
// removed as duplicates (unless keepDuplicates versions them as -vN). In
// dry-run mode it computes destinations without touching the disk.
func placePrimary(p *primary, move func(src, dst string) error, dryRun, keepDuplicates bool) []*Result {
	if !dryRun {
		if err := fileops.EnsureDir(p.destDir); err != nil {
			return []*Result{{Info: p.info, DestPath: filepath.Join(p.destDir, p.destStem+"."+p.file.Ext), Skipped: true, Reason: err.Error()}}
		}
	}

	dest, r := placeMediaFile(p.info, p.file.Path, p.destDir, p.destStem, p.file.Ext, move, dryRun, keepDuplicates)
	results := []*Result{r}

	finalStem := strings.TrimSuffix(filepath.Base(dest), "."+p.file.Ext)
	for _, sc := range p.sidecars {
		scStem := strings.TrimSuffix(SidecarName(finalStem, sc.tag, sc.file.Ext), "."+sc.file.Ext)
		info := &MediaInfo{OrigPath: sc.file.Path, Ext: sc.file.Ext}
		_, sr := placeMediaFile(info, sc.file.Path, p.destDir, scStem, sc.file.Ext, move, dryRun, keepDuplicates)
		results = append(results, sr)
	}

	return results
}

// placeMediaFile resolves one file's destination and moves it, removes it as a
// duplicate, or (dry-run) just records the plan. It returns the resolved
// destination path and the Result to report.
func placeMediaFile(info *MediaInfo, src, dir, stem, ext string, move func(src, dst string) error, dryRun, keepDuplicates bool) (string, *Result) {
	dest, dup := fileops.ResolveDest(dir, stem, ext, src, keepDuplicates)
	r := &Result{Info: info, DestPath: dest}

	switch {
	case dup:
		r.Duplicate = true
		r.Skipped = true
		r.Reason = "duplicate of existing file"
		if !dryRun {
			if err := os.Remove(src); err != nil {
				r.Reason = err.Error()
			}
		}
	case !dryRun:
		if err := move(src, dest); err != nil {
			r.Skipped = true
			r.Reason = err.Error()
		}
	}
	return dest, r
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