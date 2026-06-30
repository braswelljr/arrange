//go:build windows

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newServiceCmd(_ *CmdOptions) *cobra.Command {
	const windowsHelp = `Automatic service management is not available on Windows.

To run arrange automatically at login, create a Task Scheduler entry:
  1. Open Task Scheduler  (Win+R → taskschd.msc)
  2. Create Basic Task → trigger "When I log on"
  3. Action: Start a Program
     Program: C:\path\to\arrange.exe
     Arguments: watch C:\Users\<you>\Downloads`

	cmd := &cobra.Command{
		Use:   "service <command>",
		Short: "Manage arrange as a background service (Windows: use Task Scheduler)",
		Long:  windowsHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("service management is not supported on Windows — see 'arrange service --help'")
		},
	}

	// Register the same sub-command names so scripts don't break with
	// "unknown command" errors — they all print the Task Scheduler guidance.
	notSupported := func(sub string) *cobra.Command {
		return &cobra.Command{
			Use:   sub,
			Short: "Not supported on Windows",
			RunE: func(cmd *cobra.Command, args []string) error {
				cmd.Println(windowsHelp)
				return nil
			},
		}
	}

	cmd.AddCommand(
		notSupported("install"),
		notSupported("start"),
		notSupported("stop"),
		notSupported("status"),
		notSupported("uninstall"),
	)

	return cmd
}
