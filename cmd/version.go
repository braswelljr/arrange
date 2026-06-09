package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/braswelljr/arrange/internal/common"
)

func newVersionCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print %s version information", common.AppName),
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(opts.StdOut, "%s version %s\n", common.AppName, common.Version)

			if err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
