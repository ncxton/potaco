package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
)

// DetectTerminalProtocol checks the terminal environment and returns
// the name of the supported inline image protocol: "iterm", "kitty",
// "sixel", or "" if none is supported.
func DetectTerminalProtocol() string {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	switch {
	case termProgram == "iTerm.app":
		return "iterm"
	case termProgram == "WezTerm":
		return "iterm" // WezTerm supports iTerm2 inline images
	case termProgram == "kitty":
		return "kitty"
	case term == "xterm-kitty" || strings.HasPrefix(term, "kitty"):
		return "kitty"
	case strings.Contains(term, "sixel"):
		return "sixel"
	default:
		return ""
	}
}

// DisplayInTerminal encodes the image for the detected terminal protocol
// and returns a string to print to stdout. If no protocol is supported,
// returns a fallback message with the file path.
func DisplayInTerminal(img image.Image, path string) string {
	proto := DetectTerminalProtocol()

	switch proto {
	case "iterm":
		return itermDisplay(img, path)
	case "kitty":
		return kittyDisplay(img, path)
	case "sixel":
		return sixelDisplay(img, path)
	default:
		return fmt.Sprintf("Saved to: %s (terminal does not support inline image preview)", path)
	}
}

// itermDisplay encodes an inline image using the iTerm2 escape sequence.
func itermDisplay(img image.Image, path string) string {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	name := base64.StdEncoding.EncodeToString([]byte(filepathBase(path)))
	return fmt.Sprintf("\x1B]1337;File=inline=1;name=%s:%s\x07", name, b64)
}

// kittyDisplay encodes an inline image using the Kitty graphics protocol.
func kittyDisplay(img image.Image, path string) string {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	// Kitty sends chunks; for simplicity we send it all at once
	// Escape sequence: ESC ] 9 9 8 ; ... ST
	return fmt.Sprintf("\x1B_Ga=T,f=100,s=0,v=0,c=0,q=0;%s\x1B\\", b64)
}

// sixelDisplay is a stub for sixel support. For v0, we fall back to the
// message if sixel encoding is not yet implemented.
func sixelDisplay(img image.Image, path string) string {
	return fmt.Sprintf("Saved to: %s (sixel preview not yet implemented)", path)
}

func filepathBase(path string) string {
	// Simple basename without importing path/filepath
	idx := strings.LastIndex(path, "/")
	if idx >= 0 && idx < len(path)-1 {
		return path[idx+1:]
	}
	return path
}
