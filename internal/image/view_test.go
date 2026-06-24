package image

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestDetectTerminalProtocolUnset(t *testing.T) {
	// Clear terminal env vars
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")

	proto := DetectTerminalProtocol()
	if proto != "" {
		t.Errorf("protocol = %q, want empty string", proto)
	}
}

func TestDetectTerminalProtocolIterm(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	proto := DetectTerminalProtocol()
	if proto != "iterm" {
		t.Errorf("protocol = %q, want 'iterm'", proto)
	}
}

func TestDetectTerminalProtocolKitty(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "kitty")

	proto := DetectTerminalProtocol()
	if proto != "kitty" {
		t.Errorf("protocol = %q, want 'kitty'", proto)
	}
}

func TestDisplayInTerminalUnsupported(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.Black)

	output := DisplayInTerminal(img, "/tmp/test.png")
	if !strings.Contains(output, "Saved to:") {
		t.Errorf("output should contain 'Saved to:' fallback, got: %q", output)
	}
	if !strings.Contains(output, "does not support") {
		t.Errorf("output should mention unsupported terminal, got: %q", output)
	}
}

func TestDisplayInTerminalIterm(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.Black)

	output := DisplayInTerminal(img, "/tmp/test.png")
	if !strings.Contains(output, "\x1B]1337") {
		t.Errorf("output should contain iTerm2 escape sequence, got: %q", output)
	}
}
