package cmd

import (
	"fmt"
	"os"
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
	filesToExclude := make(map[string]int, len(exclude))

	for i, f := range exclude {
		filesToExclude[f] = i
	}

	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	files, err := getFiles(srcDir)
	if err != nil {
		return err
	}

	if files.Len() <= 0 {
		opts.Log.Info("No files found")
		return nil
	}

	for _, file := range *files {
		if _, exclude := filesToExclude[file.Name]; exclude || file.Size <= 0 {
			continue
		}

		if folder, exclude := cfg.Get(file.Ext); !exclude {
			folder := filepath.Join(destDir, folder)
			if !fileops.DirExists(folder) {
				err := os.MkdirAll(folder, os.ModePerm)
				if err != nil {
					return err
				}
			}

			newPath := filepath.Join(folder, file.Name)
			base := strings.TrimSuffix(newPath, "."+file.Ext)
			for count := 1; fileops.FileExists(newPath); count++ {
				newPath = fmt.Sprintf("%s-%d.%s", base, count, file.Ext)
			}

			opts.Log.Move(file.Path, newPath)
			if err := fileops.Move(file.Path, newPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func getFiles(dir string) (*fileops.SmartFiles, error) {
	var files fileops.SmartFiles

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			name := entry.Name()
			files = append(files, &fileops.SmartFile{
				Name: name,
				Path: filepath.Join(dir, name),
				Ext:  strings.TrimLeft(filepath.Ext(name), "."),
				Size: info.Size(),
			})
		}
	}

	return &files, nil
}
