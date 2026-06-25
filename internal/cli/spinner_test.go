package cli

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestSpinnerStopNoOp(t *testing.T) {
	// A no-op spinner (not started) should be safe to stop
	h := &spinnerHandle{}
	h.stop()
}

func TestShouldShowSpinnerNonInteractive(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("POTACO_NON_INTERACTIVE", "1")
	defer t.Setenv("POTACO_NON_INTERACTIVE", "")

	cmd := &cobra.Command{}
	cmd.Flags().Bool("stdout", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	if shouldShowSpinner(cmd) {
		t.Error("should not show spinner when non-interactive")
	}
}

func TestShouldShowSpinnerStdout(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("HOME", t.TempDir())

	cmd := &cobra.Command{}
	cmd.Flags().Bool("stdout", true, "")
	cmd.Flags().Bool("dry-run", false, "")
	if shouldShowSpinner(cmd) {
		t.Error("should not show spinner when --stdout is set")
	}
}

func TestShouldShowSpinnerDryRun(t *testing.T) {
	resetRootCmdFlags(t)
	t.Setenv("HOME", t.TempDir())

	cmd := &cobra.Command{}
	cmd.Flags().Bool("stdout", false, "")
	cmd.Flags().Bool("dry-run", true, "")
	if shouldShowSpinner(cmd) {
		t.Error("should not show spinner when --dry-run is set")
	}
}

func TestSpinnerStartStop(t *testing.T) {
	// We can't easily test the actual visual output, but we can verify
	// that start/stop doesn't hang or panic.
	h := &spinnerHandle{done: make(chan struct{}), started: true}
	go func() {
		time.Sleep(50 * time.Millisecond)
		h.stop()
	}()
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		<-h.done
	}()
	h.wg.Wait()
}
