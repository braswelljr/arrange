package cmd

import (
	"context"
	"testing"
	"time"
)

// TestWatchRun_ContextCancel verifies that watchRun returns promptly when its
// context is canceled — the mechanism the Windows service relies on to stop
// the watcher when the SCM sends a Stop/Shutdown control.
func TestWatchRun_ContextCancel(t *testing.T) {
	opts := testOpts(t)
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- watchRun(ctx, opts, dir, false, false, false) }()

	// Give the watcher a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("watchRun returned error on cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchRun did not return within 2s of context cancellation")
	}
}
