package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
	"github.com/braswelljr/arrange/internal/media"
)

func newRunCmd(opts *CmdOptions) *cobra.Command {
	var exclude []string
	var recursive bool
	var dryRun bool
	var keepDuplicates bool

	cmd := &cobra.Command{
		Use:   "run <src> [destination]",
		Short: "Scans and arranges files into folders according to their file types",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDir := args[0]
			destDir := srcDir
			if len(args) == 2 {
				destDir = args[1]
			}
			return runE(opts, srcDir, destDir, recursive, dryRun, keepDuplicates, exclude...)
		},
	}

	cmd.Flags().StringSliceVarP(&exclude, "exclude", "c", []string{}, "Exclude specified files or directories")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively scan subdirectories")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview the moves without touching any files")
	cmd.Flags().BoolVarP(&keepDuplicates, "keep-duplicates", "k", false, "Keep byte-identical duplicates as -vN copies instead of removing them")

	return cmd
}

// dirLocker provides a per-directory mutex so that SafeDestPath + Move are
// atomic — two workers writing into the same destination folder don't race on
// the collision-suffix check.
type dirLocker struct{ m sync.Map }

/**
 * Lock acquires the per-directory mutex, blocking until it is available.
 *
 * @param dir  absolute path of the destination directory to lock
 */
func (l *dirLocker) Lock(dir string) {
	v, _ := l.m.LoadOrStore(dir, new(sync.Mutex))
	v.(*sync.Mutex).Lock()
}

/**
 * Unlock releases the per-directory mutex previously acquired by Lock.
 *
 * @param dir  absolute path of the destination directory to unlock
 */
func (l *dirLocker) Unlock(dir string) {
	if v, ok := l.m.Load(dir); ok {
		v.(*sync.Mutex).Unlock()
	}
}

// sidecar pairs a subtitle-like file with the language / flag tag that ties it
// to its media file (empty tag = plain "<name>.srt" beside "<name>.mkv").
type sidecar struct {
	file *fileops.SmartFile
	tag  string
}

// fileGroup is one unit of work: a primary file plus any sidecars that must
// travel with it and inherit its final name (e.g. subtitles for a video).
type fileGroup struct {
	primary  *fileops.SmartFile
	sidecars []sidecar
}

func runE(opts *CmdOptions, srcDir, destDir string, recursive, dryRun, keepDuplicates bool, exclude ...string) error {
	skipNames := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		skipNames[name] = struct{}{}
	}

	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	var files *fileops.SmartFiles

	if recursive {
		destFolders := cfg.DestFolders()
		files, err = fileops.WalkDir(srcDir, func(name string) bool {
			lower := strings.ToLower(name)
			// When organizing in-place, skip already-organized category folders so
			// files that are already where they belong are not re-processed.
			if srcDir == destDir {
				if _, ok := destFolders[lower]; ok {
					return true
				}
			}
			return cfg.IsExcludedPath(name)
		})
	} else {
		files, err = fileops.ScanDir(srcDir)
	}

	if err != nil {
		return err
	}

	// Build the work list — filter zero-byte, symlinked, and explicitly
	// excluded files. Symlinks are skipped rather than moved so we never break
	// a link or silently relocate its target.
	var symlinkSkipped int
	work := make([]*fileops.SmartFile, 0, files.Len())
	for _, f := range *files {
		if _, skip := skipNames[f.Name]; skip || f.Size == 0 {
			continue
		}
		// Never touch hidden/system dotfiles (.DS_Store, .localized, .gitignore).
		// Their leading dot is not an extension, so moving them yields garbage
		// names and they are not meant to be reorganized anyway.
		if strings.HasPrefix(f.Name, ".") {
			continue
		}
		if f.Symlink {
			symlinkSkipped++
			continue
		}
		work = append(work, f)
	}

	if symlinkSkipped > 0 {
		opts.Log.Warnf("skipped %d symlink(s) — links are never moved", symlinkSkipped)
	}

	if len(work) == 0 {
		opts.Log.Info("No files found")
		return nil
	}

	groups := buildGroups(cfg, work)

	// ── Worker pool ───────────────────────────────────────────────────────────
	// One goroutine per logical CPU, capped to the number of groups so we never
	// spawn idle workers.
	locker := new(dirLocker)
	numWorkers := min(runtime.NumCPU(), len(groups))

	jobs := make(chan *fileGroup)
	errCh := make(chan error, len(groups)) // buffered — workers never block on send

	var moved, deduped int64
	var countMu sync.Mutex

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for g := range jobs {
				m, d, err := organiseGroup(opts, cfg, locker, destDir, g, dryRun, keepDuplicates)
				if err != nil {
					errCh <- err
				}
				if m > 0 || d > 0 {
					countMu.Lock()
					moved += int64(m)
					deduped += int64(d)
					countMu.Unlock()
				}
			}
		}()
	}

	for _, g := range groups {
		jobs <- g // blocks until a worker is ready — natural back-pressure
	}
	close(jobs)
	wg.Wait()
	close(errCh)

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}

	moveVerb, dupVerb := "moved", "removed"
	if dryRun {
		moveVerb, dupVerb = "would move", "would remove"
	}
	summary := fmt.Sprintf("%s %d file(s)", moveVerb, moved)
	if deduped > 0 {
		summary += fmt.Sprintf("; %s %d duplicate(s)", dupVerb, deduped)
	}
	if len(errs) > 0 {
		opts.Log.Warnf("%s; %d operation(s) failed", summary, len(errs))
	} else {
		opts.Log.Success(summary)
	}

	return errors.Join(errs...)
}

// buildGroups partitions the work list into groups. Media files (video/audio)
// collect any sidecar files (subtitles, etc.) that share their directory and
// name so the sidecars are moved alongside and renamed to match. Every other
// file — including sidecars with no matching media — becomes its own group.
func buildGroups(cfg *config.Config, work []*fileops.SmartFile) []*fileGroup {
	type primary struct {
		file  *fileops.SmartFile
		stem  string
		group *fileGroup
	}

	var primaries []*primary
	var subs []*fileops.SmartFile
	groups := make([]*fileGroup, 0, len(work))

	for _, f := range work {
		folder, exempt := cfg.Get(f.Ext)
		switch {
		case !exempt && (folder == config.FolderVideos || folder == config.FolderAudio):
			g := &fileGroup{primary: f}
			groups = append(groups, g)
			primaries = append(primaries, &primary{file: f, stem: cleanStem(f), group: g})
		case !exempt && folder == config.FolderSubtitles:
			subs = append(subs, f) // resolved below
		default:
			groups = append(groups, &fileGroup{primary: f})
		}
	}

	for _, s := range subs {
		subStem := cleanStem(s)
		subDir := filepath.Dir(s.Path)

		var best *primary
		var bestTag string
		for _, p := range primaries {
			if filepath.Dir(p.file.Path) != subDir {
				continue
			}
			tag, ok := media.MatchSidecar(p.stem, subStem)
			if !ok {
				continue
			}
			// Prefer the longest media stem so "Movie 2" wins over "Movie".
			if best == nil || len(p.stem) > len(best.stem) {
				best, bestTag = p, tag
			}
		}

		if best != nil {
			best.group.sidecars = append(best.group.sidecars, sidecar{file: s, tag: bestTag})
			continue
		}
		// Orphan sidecar — organize it on its own (lands in Subtitles/).
		groups = append(groups, &fileGroup{primary: s})
	}

	return groups
}

// cleanStem returns a file's name without its extension and without any
// browser-generated "(1)" duplicate suffix.
func cleanStem(f *fileops.SmartFile) string {
	return fileops.StripBrowserSuffix(strings.TrimSuffix(f.Name, "."+f.Ext))
}

// organiseFile determines the destination for a single file and moves it.
// Safe to call from multiple goroutines simultaneously. Retained as a thin
// wrapper over organiseGroup for callers (and tests) that work one file at a
// time.
func organiseFile(opts *CmdOptions, cfg *config.Config, locker *dirLocker, destRoot string, f *fileops.SmartFile) error {
	_, _, err := organiseGroup(opts, cfg, locker, destRoot, &fileGroup{primary: f}, false, false)
	return err
}

// organiseGroup routes a group's primary file to its destination and moves any
// attached sidecars alongside it, renamed to share the primary's final name.
// It returns how many files were moved and how many byte-identical duplicates
// were removed (or, in dry-run mode, would be). When keepDuplicates is true,
// identical files are versioned as -vN instead of removed.
// Safe to call from multiple goroutines simultaneously.
func organiseGroup(opts *CmdOptions, cfg *config.Config, locker *dirLocker, destRoot string, g *fileGroup, dryRun, keepDuplicates bool) (int, int, error) {
	f := g.primary
	folder, exempt := cfg.Get(f.Ext)
	if exempt {
		return 0, 0, nil
	}

	stem := cleanStem(f)

	var relDir, destName string

	switch folder {
	case config.FolderVideos, config.FolderAudio:
		// Smart media parsing: TV series → Title/Season XX/, movies → Title (YYYY)/
		parsed := media.ParseName(stem + "." + f.Ext)
		parsed.OrigPath = f.Path
		parsed.Title = cfg.CanonicalTitle(parsed.Title)
		creator := parsed.CreatorMatch(cfg.MediaCreators)
		relDir = filepath.Join(folder, parsed.DestDir(creator))
		destName = parsed.DestName()
	default:
		relDir = folder
		destName = media.SanitizeFilename(stem) + "." + f.Ext
	}

	fullDestDir := filepath.Join(destRoot, relDir)
	destStem := strings.TrimSuffix(destName, "."+f.Ext)

	// Skip files already sitting in their correct destination — prevents
	// rename loops in watch mode where the move itself triggers a new event.
	if filepath.Clean(filepath.Dir(f.Path)) == filepath.Clean(fullDestDir) {
		return 0, 0, nil
	}

	if !dryRun {
		if err := fileops.EnsureDir(fullDestDir); err != nil {
			return 0, 0, err
		}
		// Lock the destination directory so the resolve + move is atomic for the
		// whole group. Without this, two workers racing on the same dest dir
		// could both see "file.mkv" as free and overwrite each other, and a
		// sidecar could be named off a stem another worker has already taken.
		locker.Lock(fullDestDir)
		defer locker.Unlock(fullDestDir)
	}

	var moved, deduped int
	var errs []error

	// Place the primary first so its final name (which may carry a "-vN"
	// collision suffix) can seed the sidecar names.
	dest, m, d, err := placeOne(opts, fullDestDir, destStem, f.Ext, f.Path, dryRun, keepDuplicates)
	if err != nil {
		return 0, 0, err
	}
	moved += m
	deduped += d

	finalStem := strings.TrimSuffix(filepath.Base(dest), "."+f.Ext)
	for _, sc := range g.sidecars {
		scStem := strings.TrimSuffix(media.SidecarName(finalStem, sc.tag, sc.file.Ext), "."+sc.file.Ext)
		_, m, d, err := placeOne(opts, fullDestDir, scStem, sc.file.Ext, sc.file.Path, dryRun, keepDuplicates)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		moved += m
		deduped += d
	}

	return moved, deduped, errors.Join(errs...)
}

// placeOne resolves the destination for src and either moves it, removes it as
// a byte-identical duplicate of a file already there, or (in dry-run) logs the
// intended action. It returns the resolved destination path plus the moved and
// deduped deltas (0 or 1 each).
func placeOne(opts *CmdOptions, dir, stem, ext, src string, dryRun, keepDuplicates bool) (string, int, int, error) {
	dest, dup := fileops.ResolveDest(dir, stem, ext, src, keepDuplicates)

	switch {
	case dup && dryRun:
		opts.Log.Warnf("duplicate of %s — would remove %s", dest, src)
		return dest, 0, 1, nil
	case dup:
		if err := os.Remove(src); err != nil {
			return dest, 0, 0, fmt.Errorf("remove duplicate source %s: %w", src, err)
		}
		opts.Log.Warnf("duplicate of %s — removed %s", dest, src)
		return dest, 0, 1, nil
	case dryRun:
		opts.Log.Plan(src, dest)
		return dest, 1, 0, nil
	default:
		if err := fileops.Move(src, dest); err != nil {
			return dest, 0, 0, err
		}
		opts.Log.Move(src, dest)
		return dest, 1, 0, nil
	}
}