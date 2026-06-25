// Package tui provides shared TUI helpers for interactive terminal flows.
package tui

import (
	"os"

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
