package cmd

import (
	"errors"
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
			return runE(opts, srcDir, destDir, recursive, exclude...)
		},
	}

	cmd.Flags().StringSliceVarP(&exclude, "exclude", "c", []string{}, "Exclude specified files or directories")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively scan subdirectories")

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

func runE(opts *CmdOptions, srcDir, destDir string, recursive bool, exclude ...string) error {
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

	// Build the work list — filter zero-byte and explicitly excluded files.
	work := make([]*fileops.SmartFile, 0, files.Len())
	for _, f := range *files {
		if _, skip := skipNames[f.Name]; skip || f.Size == 0 {
			continue
		}
		work = append(work, f)
	}

	if len(work) == 0 {
		opts.Log.Info("No files found")
		return nil
	}

	// ── Worker pool ───────────────────────────────────────────────────────────
	// One goroutine per logical CPU, capped to the number of files so we never
	// spawn idle workers.
	locker := new(dirLocker)
	numWorkers := runtime.NumCPU()
	if numWorkers > len(work) {
		numWorkers = len(work)
	}

	jobs := make(chan *fileops.SmartFile)
	errCh := make(chan error, len(work)) // buffered — workers never block on send

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				if err := organiseFile(opts, cfg, locker, destDir, f); err != nil {
					errCh <- err
				}
			}
		}()
	}

	for _, f := range work {
		jobs <- f // blocks until a worker is ready — natural back-pressure
	}
	close(jobs)
	wg.Wait()
	close(errCh)

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

// organiseFile determines the destination for a single file and moves it.
// Safe to call from multiple goroutines simultaneously.
func organiseFile(opts *CmdOptions, cfg *config.Config, locker *dirLocker, destRoot string, f *fileops.SmartFile) error {
	folder, exempt := cfg.Get(f.Ext)
	if exempt {
		return nil
	}

	cleanStem := fileops.StripBrowserSuffix(strings.TrimSuffix(f.Name, "."+f.Ext))

	var relDir, destName string

	switch folder {
	case config.FolderVideos, config.FolderAudio:
		// Smart media parsing: TV series → Title/Season XX/, movies → Title (YYYY)/
		parsed := media.ParseName(cleanStem + "." + f.Ext)
		parsed.OrigPath = f.Path
		creator := parsed.CreatorMatch(cfg.MediaCreators)
		relDir = filepath.Join(folder, parsed.DestDir(creator))
		destName = parsed.DestName()
	default:
		relDir = folder
		destName = media.SanitizeFilename(cleanStem) + "." + f.Ext
	}

	fullDestDir := filepath.Join(destRoot, relDir)
	if err := fileops.EnsureDir(fullDestDir); err != nil {
		return err
	}

	stem := strings.TrimSuffix(destName, "."+f.Ext)

	// Skip files already sitting in their correct destination — prevents
	// rename loops in watch mode where the move itself triggers a new event.
	if filepath.Clean(filepath.Dir(f.Path)) == filepath.Clean(fullDestDir) {
		return nil
	}

	// Lock the destination directory so SafeDestPath + Move are atomic.
	// Without this, two workers racing on the same dest dir could both see
	// "file.mkv" as free and overwrite each other.
	locker.Lock(fullDestDir)
	newPath := fileops.SafeDestPath(fullDestDir, stem, f.Ext)
	err := fileops.Move(f.Path, newPath)
	locker.Unlock(fullDestDir)

	// Log outside the critical section — Logger is goroutine-safe and
	// there is no reason to hold the per-dir lock while writing to stdout.
	opts.Log.Move(f.Path, newPath)
	return err
}
