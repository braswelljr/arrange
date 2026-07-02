//go:build windows

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/braswelljr/arrange/internal/common"
)

// svcName is the Windows service identifier registered with the SCM.
var svcName = common.AppName

// arrangeService adapts the watch loop to the Windows Service Control Manager.
// It runs watchRun in a goroutine and cancels it when the SCM asks the service
// to stop or the machine shuts down.
type arrangeService struct {
	opts *CmdOptions
	dir  string
}

// Execute is the svc.Handler entry point invoked by the SCM.
func (s *arrangeService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// destDir is the watched root; the service always organizes in place.
		if err := watchRun(ctx, s.opts, s.dir, true, false, false); err != nil {
			s.opts.Log.Errorf("watch service: %v", err)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: accepted}

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			cancel()
			<-done
			return false, 0
		default:
			s.opts.Log.Warnf("unexpected service control request: %d", c.Cmd)
		}
	}
	return false, 0
}

// newServiceCmd builds the Windows `service` command tree. It manages a real
// Windows service through the SCM (install/start/stop/status/uninstall) and
// exposes a hidden `_run` sub-command that the SCM launches to host the watcher.
func newServiceCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service <command>",
		Short: "Manage arrange as a Windows service",
	}

	installCmd := &cobra.Command{
		Use:   "install <dir>",
		Short: "Install and register the service to watch <dir> at startup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			watchDir, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolve watch directory: %w", err)
			}
			if !dirExists(watchDir) {
				return fmt.Errorf("watch directory does not exist: %s", watchDir)
			}
			if err := installService(watchDir); err != nil {
				return err
			}
			opts.Log.Successf("installed service %q watching %s", svcName, watchDir)
			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:    "_run <dir>",
		Hidden: true, // launched by the SCM, not by users
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return svc.Run(svcName, &arrangeService{opts: opts, dir: args[0]})
		},
	}

	cmd.AddCommand(
		installCmd,
		runCmd,
		simpleServiceCmd("start", "Start the service", opts, controlStart),
		simpleServiceCmd("stop", "Stop the service", opts, controlStop),
		simpleServiceCmd("status", "Show the service status", opts, controlStatus),
		uninstallCmd(opts),
	)

	return cmd
}

// serviceAction is a management operation dispatched against the installed service.
type serviceAction func(*mgr.Service) (string, error)

func simpleServiceCmd(use, short string, opts *CmdOptions, action serviceAction) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg, err := withService(action)
			if err != nil {
				return err
			}
			opts.Log.Info(msg)
			return nil
		},
	}
}

func uninstallCmd(opts *CmdOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"remove"},
		Short:   "Uninstall the service",
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := removeService(); err != nil {
				return err
			}
			opts.Log.Successf("removed service %q", svcName)
			return nil
		},
	}
}

// ── SCM helpers ────────────────────────────────────────────────────────────────

func installService(watchDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager (run as Administrator?): %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	if s, err := m.OpenService(svcName); err == nil {
		_ = s.Close()
		return fmt.Errorf("service %q is already installed", svcName)
	}

	s, err := m.CreateService(svcName, exe, mgr.Config{
		DisplayName: common.AppName + " file organizer",
		Description: "Watches a directory and organizes files into folders by type.",
		StartType:   mgr.StartAutomatic,
	}, "service", "_run", watchDir)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer func() { _ = s.Close() }()

	return nil
}

func removeService() error {
	return withServiceErr(func(s *mgr.Service) error {
		if err := s.Delete(); err != nil {
			return fmt.Errorf("delete service: %w", err)
		}
		return nil
	})
}

func controlStart(s *mgr.Service) (string, error) {
	if err := s.Start(); err != nil {
		return "", fmt.Errorf("start service: %w", err)
	}
	return fmt.Sprintf("service %q started", svcName), nil
}

func controlStop(s *mgr.Service) (string, error) {
	status, err := s.Control(svc.Stop)
	if err != nil {
		return "", fmt.Errorf("stop service: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		if status, err = s.Query(); err != nil {
			return "", fmt.Errorf("query service: %w", err)
		}
	}
	return fmt.Sprintf("service %q stopped", svcName), nil
}

func controlStatus(s *mgr.Service) (string, error) {
	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("query service: %w", err)
	}
	return fmt.Sprintf("service %q state: %s", svcName, stateString(status.State)), nil
}

// withService opens the installed service, runs action, and closes it.
func withService(action serviceAction) (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("connect to service manager (run as Administrator?): %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	s, err := m.OpenService(svcName)
	if err != nil {
		return "", fmt.Errorf("service %q is not installed", svcName)
	}
	defer func() { _ = s.Close() }()

	return action(s)
}

func withServiceErr(action func(*mgr.Service) error) error {
	_, err := withService(func(s *mgr.Service) (string, error) {
		return "", action(s)
	})
	return err
}

func stateString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "start pending"
	case svc.StopPending:
		return "stop pending"
	case svc.Running:
		return "running"
	case svc.ContinuePending:
		return "continue pending"
	case svc.PausePending:
		return "pause pending"
	case svc.Paused:
		return "paused"
	default:
		return fmt.Sprintf("unknown (%d)", state)
	}
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
