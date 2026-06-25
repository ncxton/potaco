package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/tui"
	"golang.org/x/term"
)

// ErrorCategory classifies an error for exit code selection and rendering.
type ErrorCategory int

const (
	CatConfig  ErrorCategory = 2
	CatAPI     ErrorCategory = 3
	CatImage   ErrorCategory = 4
	CatGeneric ErrorCategory = 1
)

// UserError is a first-class user-facing error. It carries a friendly
// message, an optional hint, and the raw original error for debug logging.
type UserError struct {
	Category ErrorCategory
	Message  string
	Hint     string
	Raw      error
}

func (e *UserError) Error() string {
	if e.Raw != nil {
		return e.Raw.Error()
	}
	return e.Message
}

func (e *UserError) Unwrap() error {
	return e.Raw
}

func (e *UserError) ExitCode() int {
	return int(e.Category)
}

// userErr constructs a UserError with the given category, message, hint,
// and raw error.
func userErr(cat ErrorCategory, message, hint string, raw error) *UserError {
	return &UserError{
		Category: cat,
		Message:  message,
		Hint:     hint,
		Raw:      raw,
	}
}

// configUserErr, apiUserErr, imageUserErr are convenience constructors.
func configUserErr(message, hint string, raw error) *UserError {
	return userErr(CatConfig, message, hint, raw)
}

func apiUserErr(message, hint string, raw error) *UserError {
	return userErr(CatAPI, message, hint, raw)
}

func imageUserErr(message, hint string, raw error) *UserError {
	return userErr(CatImage, message, hint, raw)
}

// renderUserError prints the error to w. Uses colors when w is a TTY,
// plain text otherwise (for AI agents and piped output).
func renderUserError(w io.Writer, ue *UserError) {
	if isWriterTTY(w) {
		renderUserErrorColored(w, ue)
		return
	}
	fmt.Fprintf(w, "Error: %s\n", ue.Message)
	if ue.Hint != "" {
		fmt.Fprintf(w, "Hint: %s\n", ue.Hint)
	}
}

var (
	errLabelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	errMsgStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	errHintLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	errHintMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

func renderUserErrorColored(w io.Writer, ue *UserError) {
	fmt.Fprintf(w, "%s %s\n", errLabelStyle.Render("Error:"), errMsgStyle.Render(ue.Message))
	if ue.Hint != "" {
		fmt.Fprintf(w, "%s %s\n", errHintLabelStyle.Render("Hint:"), errHintMsgStyle.Render(ue.Hint))
	}
}

func isWriterTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// debugLog writes the raw error to ~/.potaco/debug.log with a timestamp.
// Silently skips if the log file cannot be opened.
func debugLog(ue *UserError) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(home, ".potaco")
	logPath := filepath.Join(logDir, "debug.log")

	if err := os.MkdirAll(logDir, 0700); err != nil {
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	catName := "generic"
	switch ue.Category {
	case CatConfig:
		catName = "config"
	case CatAPI:
		catName = "api"
	case CatImage:
		catName = "image"
	}

	rawMsg := ""
	if ue.Raw != nil {
		rawMsg = ue.Raw.Error()
	}
	ts := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "%s [%s] %s\n", ts, catName, rawMsg)
}

// renderAnyError is the fallback for errors that are not *UserError.
// It renders the plain error string.
func renderAnyError(w io.Writer, err error) {
	var ue *UserError
	if errors.As(err, &ue) {
		renderUserError(w, ue)
		debugLog(ue)
		return
	}
	if isWriterTTY(w) {
		fmt.Fprintf(w, "%s %s\n", errLabelStyle.Render("Error:"), errMsgStyle.Render(err.Error()))
	} else {
		fmt.Fprintf(w, "Error: %s\n", err.Error())
	}
}

// silence unused import warnings for tui/config in future expansion
var _ = tui.IsInteractive
var _ = config.DefaultConfigPath

// validateOutputPath checks that the output path is writable before
// making an API call. Returns a UserError if the path is a directory,
// not writable, or otherwise invalid. Returns nil if the path is OK
// or empty (auto-generated).
func validateOutputPath(outputPath string) *UserError {
	if outputPath == "" {
		return nil
	}
	info, err := os.Stat(outputPath)
	if err != nil && !os.IsNotExist(err) {
		// Permission error or other stat failure.
		return imageUserErr(
			fmt.Sprintf("Cannot write to '%s'.", outputPath),
			"Check that the path is valid and you have write permissions.",
			err,
		)
	}
	if err == nil && info.IsDir() {
		return imageUserErr(
			fmt.Sprintf("'%s' is a directory, not a file.", outputPath),
			"Specify a filename ending in .png or .jpeg, or omit -o to auto-generate one.",
			fmt.Errorf("output path is a directory: %s", outputPath),
		)
	}
	// Check parent directory is writable.
	dir := filepath.Dir(outputPath)
	if dir == "" {
		dir = "."
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return imageUserErr(
			fmt.Sprintf("Cannot save to '%s'. The directory does not exist.", dir),
			"Create the directory first or choose a different output path.",
			err,
		)
	}
	if !dirInfo.IsDir() {
		return imageUserErr(
			fmt.Sprintf("'%s' is not a directory.", dir),
			"The parent path must be a directory.",
			fmt.Errorf("parent path is not a directory: %s", dir),
		)
	}
	return nil
}

// friendlyPath extracts a file path from a Go error string when possible.
// Returns the path or empty string.
func friendlyPath(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	// os.PathError: "open /path: is a directory" or "open /path: no such file or directory"
	if idx := strings.Index(s, "open "); idx >= 0 {
		rest := s[idx+5:]
		if colon := strings.Index(rest, ": "); colon >= 0 {
			return rest[:colon]
		}
		return rest
	}
	if idx := strings.Index(s, "create "); idx >= 0 {
		rest := s[idx+7:]
		if colon := strings.Index(rest, ": "); colon >= 0 {
			return rest[:colon]
		}
		return rest
	}
	return ""
}
