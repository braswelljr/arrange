package main

import (
	"os"

	"github.com/braswelljr/arrange/cmd"
	"github.com/braswelljr/arrange/internal/logger"
)

func main() {
	cmdOpts := &cmd.CmdOptions{
		StdErr: os.Stderr,
		StdOut: os.Stdout,
		Log:    logger.New(os.Stdout, os.Stderr),
	}
	runCommand(cmdOpts)
}

func runCommand(opts *cmd.CmdOptions) {
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetOut(opts.StdOut)
	rootCmd.SetErr(opts.StdErr)

	err := rootCmd.Execute()
	if err != nil {
		opts.Log.Errorf("could not run command: %s", err)
		os.Exit(2)
	}
}
