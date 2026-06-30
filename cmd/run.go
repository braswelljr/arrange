package cmd

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
)

func newRunCmd(opts *CmdOptions) *cobra.Command {
	var exclude []string
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
			return runE(opts, srcDir, destDir, exclude...)
		},
	}

	cmd.Flags().StringSliceVarP(&exclude, "exclude", "c", []string{}, "Exclude specified files or directories")

	return cmd
}

func runE(opts *CmdOptions, srcDir, destDir string, exclude ...string) error {
	skipNames := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		skipNames[name] = struct{}{}
	}

	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	files, err := fileops.ScanDir(srcDir)
	if err != nil {
		return err
	}

	if files.Len() == 0 {
		opts.Log.Info("No files found")
		return nil
	}

	for _, f := range *files {
		if _, skip := skipNames[f.Name]; skip || f.Size == 0 {
			continue
		}

		folder, exempt := cfg.Get(f.Ext)
		if exempt {
			continue
		}

		destDir := filepath.Join(destDir, folder)
		if err := fileops.EnsureDir(destDir); err != nil {
			return err
		}

		cleanStem := fileops.StripBrowserSuffix(strings.TrimSuffix(f.Name, "."+f.Ext))
		newPath := fileops.SafeDestPath(destDir, cleanStem, f.Ext)

		opts.Log.Move(f.Path, newPath)
		if err := fileops.Move(f.Path, newPath); err != nil {
			return err
		}
	}
	return nil
}
