package cmd

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

func newWatchCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <dir>",
		Short: "Watches the directory for changes",
		Long:  "Watches a directory for [file] changes and automatically organizes the files in the directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]

			return watchRun(opts, dir)
		},
	}

	return cmd
}

func watchRun(opts *CmdOptions, dir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			opts.Log.Println("ERROR closing watcher:", err)
		}
	}()

	errc := make(chan error, 1)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					errc <- nil
					return
				}
				opts.Log.Println("EVENT:", event)
				if event.Op == fsnotify.Create {
					if err := runE(opts, dir, dir); err != nil {
						errc <- err
						return
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					errc <- nil
					return
				}
				opts.Log.Println("ERROR:", err)
			}
		}
	}()

	if err := watcher.Add(dir); err != nil {
		return err
	}
	return <-errc
}
