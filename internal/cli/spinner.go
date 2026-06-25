package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
var spinnerLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

type spinnerHandle struct {
	done    chan struct{}
	wg      sync.WaitGroup
	started bool
}

// shouldShowSpinner returns true when the spinner should be displayed:
// interactive mode, stderr is a TTY, not stdout/json/dry-run/non-interactive.
func shouldShowSpinner(cmd *cobra.Command) bool {
	if !tui.IsInteractive() {
		return false
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return false
	}
	if flagBool(cmd, "stdout") {
		return false
	}
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonMode {
		return false
	}
	if flagBool(cmd, "dry-run") {
		return false
	}
	return true
}

// startSpinner launches a spinner on stderr with the given label.
// Call stop() on the returned handle when the operation completes.
// If the spinner should not be shown, returns a no-op handle.
func startSpinner(cmd *cobra.Command, label string) *spinnerHandle {
	if !shouldShowSpinner(cmd) {
		return &spinnerHandle{}
	}

	h := &spinnerHandle{done: make(chan struct{})}
	h.wg.Add(1)
	h.started = true

	go func() {
		defer h.wg.Done()
		w := io.Writer(os.Stderr)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		frameIdx := 0
		for {
			select {
			case <-h.done:
				// Clear the spinner line
				fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", len(label)+4))
				return
			case <-ticker.C:
				frame := spinnerStyle.Render(spinnerFrames[frameIdx])
				msg := fmt.Sprintf("\r%s %s", frame, spinnerLabelStyle.Render(label))
				fmt.Fprint(w, msg)
				frameIdx = (frameIdx + 1) % len(spinnerFrames)
			}
		}
	}()

	return h
}

func (h *spinnerHandle) stop() {
	if !h.started {
		return
	}
	close(h.done)
	h.wg.Wait()
}
