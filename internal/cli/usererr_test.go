package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserErrorExitCode(t *testing.T) {
	tests := []struct {
		cat  ErrorCategory
		want int
	}{
		{CatConfig, 2},
		{CatAPI, 3},
		{CatImage, 4},
		{CatGeneric, 1},
	}
	for _, tc := range tests {
		ue := &UserError{Category: tc.cat, Message: "test"}
		if ue.ExitCode() != tc.want {
			t.Errorf("ExitCode() = %d, want %d", ue.ExitCode(), tc.want)
		}
	}
}

func TestUserErrorError(t *testing.T) {
	// When raw is set, Error() returns raw's message
	raw := errors.New("raw go error")
	ue := &UserError{Category: CatImage, Message: "friendly", Raw: raw}
	if ue.Error() != "raw go error" {
		t.Errorf("Error() = %q, want 'raw go error'", ue.Error())
	}
	// When raw is nil, Error() returns Message
	ue2 := &UserError{Category: CatImage, Message: "friendly"}
	if ue2.Error() != "friendly" {
		t.Errorf("Error() = %q, want 'friendly'", ue2.Error())
	}
}

func TestUserErrorUnwrap(t *testing.T) {
	raw := errors.New("inner")
	ue := &UserError{Category: CatConfig, Message: "outer", Raw: raw}
	if !errors.Is(ue, raw) {
		t.Error("errors.Is should find raw via Unwrap")
	}
}

func TestRenderUserErrorNoHint(t *testing.T) {
	ue := &UserError{Category: CatImage, Message: "Something went wrong."}
	var buf bytes.Buffer
	renderUserError(&buf, ue)
	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Errorf("should contain 'Error:', got: %q", output)
	}
	if !strings.Contains(output, "Something went wrong.") {
		t.Errorf("should contain message, got: %q", output)
	}
	if strings.Contains(output, "Hint:") {
		t.Errorf("should not contain Hint when empty, got: %q", output)
	}
}

func TestRenderUserErrorWithHint(t *testing.T) {
	ue := &UserError{Category: CatAPI, Message: "API failed.", Hint: "Check your key."}
	var buf bytes.Buffer
	renderUserError(&buf, ue)
	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Errorf("should contain 'Error:', got: %q", output)
	}
	if !strings.Contains(output, "API failed.") {
		t.Errorf("should contain message, got: %q", output)
	}
	if !strings.Contains(output, "Hint:") {
		t.Errorf("should contain 'Hint:', got: %q", output)
	}
	if !strings.Contains(output, "Check your key.") {
		t.Errorf("should contain hint text, got: %q", output)
	}
}

func TestValidateOutputPathEmpty(t *testing.T) {
	ue := validateOutputPath("")
	if ue != nil {
		t.Errorf("empty path should return nil, got: %v", ue)
	}
}

func TestValidateOutputPathDirectory(t *testing.T) {
	dir := t.TempDir()
	ue := validateOutputPath(dir)
	if ue == nil {
		t.Fatal("directory path should return error")
	}
	if !strings.Contains(ue.Message, "is a directory") {
		t.Errorf("error should mention directory, got: %s", ue.Message)
	}
	if !strings.Contains(ue.Hint, ".png") {
		t.Errorf("hint should mention filename, got: %s", ue.Hint)
	}
}

func TestValidateOutputPathValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.png")
	ue := validateOutputPath(path)
	if ue != nil {
		t.Errorf("valid file path should return nil, got: %v", ue)
	}
}

func TestValidateOutputPathNonexistentParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent_subdir", "output.png")
	ue := validateOutputPath(path)
	if ue == nil {
		t.Fatal("path with nonexistent parent dir should return error")
	}
	if !strings.Contains(ue.Message, "does not exist") {
		t.Errorf("error should mention directory does not exist, got: %s", ue.Message)
	}
}

func TestDebugLog(t *testing.T) {
	// Use a temp HOME so we don't pollute the real one
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	ue := &UserError{
		Category: CatAPI,
		Message:  "Generation failed.",
		Raw:      fmt.Errorf("http request: connection refused"),
	}
	debugLog(ue)

	// Verify log file was created
	logPath := filepath.Join(tmpHome, ".potaco", "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		home, _ := os.UserHomeDir()
		t.Fatalf("debug log should be created at %s (HOME=%s, UserHomeDir=%s): %v", logPath, tmpHome, home, err)
	}
	logStr := string(data)
	if !strings.Contains(logStr, "category=api") {
		t.Errorf("log should contain category=api, got: %s", logStr)
	}
	if !strings.Contains(logStr, "connection refused") {
		t.Errorf("log should contain raw error, got: %s", logStr)
	}
}

func TestConfigUserErr(t *testing.T) {
	ue := configUserErr("msg", "hint", fmt.Errorf("raw"))
	if ue.Category != CatConfig {
		t.Errorf("category = %v, want CatConfig", ue.Category)
	}
	if ue.Message != "msg" {
		t.Errorf("message = %q, want 'msg'", ue.Message)
	}
	if ue.Hint != "hint" {
		t.Errorf("hint = %q, want 'hint'", ue.Hint)
	}
}

func TestAPIUserErr(t *testing.T) {
	ue := apiUserErr("msg", "hint", fmt.Errorf("raw"))
	if ue.Category != CatAPI {
		t.Errorf("category = %v, want CatAPI", ue.Category)
	}
}

func TestImageUserErr(t *testing.T) {
	ue := imageUserErr("msg", "hint", fmt.Errorf("raw"))
	if ue.Category != CatImage {
		t.Errorf("category = %v, want CatImage", ue.Category)
	}
}
