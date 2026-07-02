//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/takama/daemon"

	"github.com/braswelljr/arrange/internal/common"
	"github.com/braswelljr/arrange/internal/fileops"
)

// Service is the daemon service struct
type Service struct {
	daemon.Daemon
}

/**
 * ServiceActionType identifies the daemon management operation to perform.
 *
 * Values: install | start | stop | status | remove
 */
type ServiceActionType int

const (
	install ServiceActionType = iota
	start
	stop
	status
	remove
)

var dependencies = []string{ /*"dummy.service"*/ }

/**
 * NewService creates a platform-appropriate daemon.Daemon instance.
 *
 * On Darwin it uses daemon.UserAgent; on all other platforms it uses
 * daemon.SystemDaemon so the service runs under the system init manager.
 *
 * @returns  (*Service, nil) on success, or (nil, error) if daemon.New fails
 */
func NewService() (*Service, error) {
	daemonKind := daemon.SystemDaemon
	if runtime.GOOS == "darwin" {
		daemonKind = daemon.UserAgent
	}
	srv, err := daemon.New(common.AppName, fmt.Sprint(common.AppName, " service"), daemonKind, dependencies...)
	if err != nil {
		return nil, err
	}

	return &Service{srv}, nil
}

/**
 * Manage dispatches a daemon management action to the underlying daemon.Daemon.
 *
 * @param srvType  the action to perform (install, start, stop, status, remove)
 * @param args     additional arguments forwarded to daemon.Install (only used for install)
 * @returns        human-readable status message and any error encountered
 */
func (service *Service) Manage(srvType ServiceActionType, args ...string) (string, error) {
	switch srvType {
	case install:
		return service.Install(args...)
	case remove:
		return service.Remove()
	case start:
		return service.Start()
	case stop:
		// No need to explicitly stop cron since job will be killed
		return service.Stop()
	case status:
		return service.Status()
	}

	return "", fmt.Errorf("invalid action")
}

func newServiceCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service <command>",
		Short: "Manage arrange services",
	}

	installCmd := &cobra.Command{
		Use:   "install <dir>",
		Short: "Install the service to watch <dir> and organize files at login/boot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve to an absolute path so the installed unit does not depend
			// on the working directory it happens to be launched from.
			watchDir, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolve watch directory: %w", err)
			}
			if !fileops.DirExists(watchDir) {
				return fmt.Errorf("watch directory does not exist: %s", watchDir)
			}

			// On Linux the service is a system daemon and installing it needs
			// root; warn early rather than failing deep inside the daemon lib.
			if runtime.GOOS == "linux" && os.Geteuid() != 0 {
				opts.Log.Warn("installing a systemd service usually requires root — re-run with sudo if this fails")
			}

			service, err := NewService()
			if err != nil {
				return err
			}

			if err := runE(opts, watchDir, watchDir, false, false, false); err != nil {
				return err
			}
			status, err := service.Manage(install, "watch", watchDir)

			if err != nil {
				return err
			}

			opts.Log.Success(status)
			return nil
		},
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the service",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := NewService()
			if err != nil {
				return err
			}

			status, err := service.Manage(start)
			if err != nil {
				return err
			}

			opts.Log.Success(status)
			return nil
		},
	}

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running services",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			service, err := NewService()
			if err != nil {
				return err
			}

			status, err := service.Manage(stop)

			if err != nil {
				return err
			}

			opts.Log.Info(status)
			return nil
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check the status of the service",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			service, err := NewService()
			if err != nil {
				return err
			}

			status, err := service.Manage(status)

			if err != nil {
				return err
			}

			opts.Log.Info(status)
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"remove"},
		Short:   "Uninstall the service",
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			service, err := NewService()
			if err != nil {
				return err
			}

			status, err := service.Manage(remove)

			if err != nil {
				return err
			}

			opts.Log.Success(status)
			return nil
		},
	}

	cmd.AddCommand(installCmd, startCmd, stopCmd, statusCmd, removeCmd)

	return cmd
}
