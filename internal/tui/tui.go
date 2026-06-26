// Package tui provides shared TUI helpers for interactive terminal flows.
package tui

import (
	"errors"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// nonInteractive is set by the CLI when --non-interactive is passed.
var nonInteractive bool

// SetNonInteractive enables or disables non-interactive mode. When true,
// IsInteractive returns false regardless of TTY status.
func SetNonInteractive(v bool) {
	nonInteractive = v
}

// IsInteractive returns true when stdin is a TTY and the user has not opted
// out of interactive mode via the --non-interactive flag or the
// POTACO_NON_INTERACTIVE environment variable.
func IsInteractive() bool {
	if nonInteractive {
		return false
	}
	if os.Getenv("POTACO_NON_INTERACTIVE") == "1" {
		return false
	}
	return IsTTY()
}

// IsTTY returns true when stdin is a terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// newForm creates a huh.Form with the Esc key bound to quit alongside
// the default ctrl+c. This allows users to cancel any interactive TUI
// flow by pressing Esc.
func newForm(groups ...*huh.Group) *huh.Form {
	form := huh.NewForm(groups...)
	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c / esc", "quit"),
	)
	form.WithKeyMap(keymap)
	return form
}

// isCancelled returns true when the error from form.Run() indicates the
// user aborted (pressed Esc or Ctrl+C).
func isCancelled(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}

// ConfirmAction shows a yes/no confirmation prompt and returns the result.
// Returns (false, error) when the user presses Esc/Ctrl+C.
func ConfirmAction(prompt string) (bool, error) {
	var result bool
	form := newForm(huh.NewGroup(
		huh.NewConfirm().
			Title(prompt).
			Value(&result),
	))
	if err := form.Run(); err != nil {
		return false, err
	}
	return result, nil
}
