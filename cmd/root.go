package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/braswelljr/arrange/internal/common"
	"github.com/braswelljr/arrange/internal/config"
	"github.com/braswelljr/arrange/internal/logger"
)

/**
 * CmdOptions holds shared state passed to every subcommand.
 *
 * Fields:
 *   - StdErr:      destination for error output (defaults to os.Stderr)
 *   - StdOut:      destination for normal output (defaults to os.Stdout)
 *   - ConfigPath:  path to the JSON config file; empty means use the platform default
 *   - Log:         structured logger shared across all subcommands
 */
type CmdOptions struct {
	StdErr *os.File
	StdOut *os.File

	ConfigPath string
	Log        *logger.Logger
}

/**
 * NewRootCmd builds the top-level cobra command with all subcommands attached.
 *
 * @param opts  shared options (logger, config path, I/O streams) passed to every subcommand
 * @returns     configured *cobra.Command ready to be executed via cmd.Execute()
 *
 * Usage:
 *   opts := &CmdOptions{StdOut: os.Stdout, StdErr: os.Stderr, Log: logger.New(...)}
 *   root := NewRootCmd(opts)
 *   root.Execute()
 */
func NewRootCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   common.AppName,
		Short: "Organize files into folders by file types",
	}

	cmd.PersistentFlags().StringVarP(&opts.ConfigPath, "config-path", "C", config.Path(), "Path to config file")

	cmd.AddCommand(newRunCmd(opts))
	cmd.AddCommand(newMediaCmd(opts))
	cmd.AddCommand(newSetupCmd(opts))
	cmd.AddCommand(newServiceCmd(opts))
	cmd.AddCommand(newWatchCmd(opts))
	cmd.AddCommand(newVersionCmd(opts))

	return cmd
}
