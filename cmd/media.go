package cmd

import (
	"github.com/spf13/cobra"

	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/fileops"
	"github.com/braswelljr/arrange/internal/media"
)

func newMediaCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media <src> [dest]",
		Short: "Organizes media files into structured series / movie / audio folders",
		Long: `Parses video filenames to detect TV series, movies, and multi-part movies,
then arranges them into a tidy hierarchy:

  TV series   → <Title>/Season XX/<Title> SxxExx [quality].ext
  Movie       → <Title> (YYYY)/<Title> (YYYY) [quality].ext
  Multi-part  → <Title> (YYYY)/Part N/<Title> (YYYY) [quality].ext

Creator grouping: add names to "media_creators" in your config file and all
their films will be placed under a shared top-level folder, e.g.

  config:  "media_creators": ["Tyler Perry"]
  result:  Tyler Perry/
             Why Did I Get Married (2007)/...
             Madea Goes to Jail (2009)/...
             Why Did I Get Married Too (2010)/...`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			dest := src
			if len(args) == 2 {
				dest = args[1]
			}
			return mediaRunE(opts, src, dest)
		},
	}
	return cmd
}

func mediaRunE(opts *CmdOptions, src, dest string) error {
	cfg, err := config.NewConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	results, err := media.Organize(src, dest, cfg, fileops.Move)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		opts.Log.Info("no media files found")
		return nil
	}

	for _, r := range results {
		if r.Skipped {
			opts.Log.Warnf("skipped %s: %s", r.Info.OrigPath, r.Reason)
			continue
		}
		opts.Log.Move(r.Info.OrigPath, r.DestPath)
	}

	return nil
}
