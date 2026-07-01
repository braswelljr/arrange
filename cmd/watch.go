package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/braswelljr/arrange/internal/config"
)

// debounceDelay batches rapid bursts (archive extractions, multi-file copies)
// into a single organize pass per source directory.
const debounceDelay = 800 * time.Millisecond

func newWatchCmd(opts *CmdOptions) *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "watch <directory>",
		Short: "Watches a directory for new files and organizes them automatically",
		Long: `Watches a directory for newly created files and organizes them according to
your config.  Files that arrive via Telegram, WhatsApp, or any other app
sub-folder are moved into the correct category folder under the root watched
directory — e.g. a video dropped into ~/Downloads/Telegram Desktop/ is
moved to ~/Downloads/Videos/.

In-progress downloads (crdownload, part, aria2, …) are never touched.

Use --recursive / -r to also watch all subdirectories.  Directories listed
in excluded_dirs in your config are skipped from automatic watching.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return watchRun(opts, args[0], recursive)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively watch subdirectories")

	return cmd
}

func watchRun(opts *CmdOptions, dir string, recursive bool) error {
	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			opts.Log.Errorf("closing watcher: %v", err)
		}
	}()

	var watcherMu sync.Mutex

	// addDir registers path with the watcher unless the user has explicitly
	// listed it in excluded_dirs.
	addDir := func(path string) {
		if cfg.IsExcludedPath(path) {
			opts.Log.Warnf("skipping excluded directory: %s", path)
			return
		}
		watcherMu.Lock()
		defer watcherMu.Unlock()
		if err := watcher.Add(path); err != nil {
			opts.Log.Errorf("watch %s: %v", path, err)
			return
		}
		opts.Log.Infof("watching %s", path)
	}

	addDir(dir)

	if recursive {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() || path == dir {
				return nil
			}
			addDir(path)
			return nil
		})
	}

	// scheduleRun debounces an organize pass.
	// srcDir is the directory where the event was observed; destDir is always
	// the root watched directory so files from app sub-folders (Telegram,
	// WhatsApp, …) are sorted into ~/Downloads/Videos/, ~/Downloads/Audio/,
	// etc. rather than creating a parallel hierarchy inside the sub-folder.
	var (
		timersMu sync.Mutex
		timers   = make(map[string]*time.Timer)
	)

	scheduleRun := func(srcDir string) {
		// Don't re-organize files that have already landed in a destination
		// folder — doing so would rename them with -v1, -v2, … suffixes.
		rel, relErr := filepath.Rel(dir, srcDir)
		if relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			topDir := strings.ToLower(strings.SplitN(filepath.ToSlash(rel), "/", 2)[0])
			if _, ok := cfg.DestFolders()[topDir]; ok {
				return
			}
		}

		timersMu.Lock()
		defer timersMu.Unlock()
		if t, ok := timers[srcDir]; ok {
			t.Stop()
		}
		timers[srcDir] = time.AfterFunc(debounceDelay, func() {
			if err := runE(opts, srcDir, dir, false); err != nil {
				opts.Log.Errorf("organize %s → %s: %v", srcDir, dir, err)
			}
			timersMu.Lock()
			delete(timers, srcDir)
			timersMu.Unlock()
		})
	}

	errc := make(chan error, 1)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					errc <- nil
					return
				}

				opts.Log.Eventf("%s %s", event.Op, event.Name)

				if event.Op.Has(fsnotify.Create) {
					// In recursive mode, start watching any new sub-directory
					// that appears (unless the user excluded it).
					if recursive {
						if fi, statErr := os.Stat(event.Name); statErr == nil && fi.IsDir() {
							addDir(event.Name)
							continue
						}
					}

					// Schedule an organize pass sourced from the directory
					// that contains the new file.
					scheduleRun(filepath.Dir(event.Name))
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					errc <- nil
					return
				}
				opts.Log.Errorf("watcher: %v", err)
			}
		}
	}()

	return <-errc
}
