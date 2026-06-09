package main

import (
	"fmt"
	"log"
	"os"

	"github.com/braswelljr/arrange/cmd"
)

func main() {
	cmdOpts := &cmd.CmdOptions{
		StdErr: os.Stderr,
		StdOut: os.Stdout,
		Log:    &log.Logger{},
	}
	runCommand(cmdOpts)
}

func runCommand(opts *cmd.CmdOptions) {
	opts.Log = log.New(opts.StdOut, "", log.Ldate|log.Ltime)

	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetOut(opts.StdOut)
	rootCmd.SetErr(opts.StdErr)

	err := rootCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintf(opts.StdErr, "could not run command: %s\n", err)
		os.Exit(2)
	}
}
