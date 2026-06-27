package cli

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/pflag"
)

func TestEditCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "edit" || strings.HasPrefix(cmd.Use, "edit ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'edit' subcommand")
	}
}

func TestEditCommandHasImageFlag(t *testing.T) {
	imgFlag := editCmd.Flags().Lookup("image")
	if imgFlag == nil {
		t.Fatal("edit command should have --image flag")
	}
}

func TestEditCommandHasMaskFlags(t *testing.T) {
	if editCmd.Flags().Lookup("mask") == nil {
		t.Fatal("edit command should have --mask flag")
	}
	if editCmd.Flags().Lookup("mask-rect") == nil {
		t.Fatal("edit command should have --mask-rect flag")
	}
	if editCmd.Flags().Lookup("mask-circle") == nil {
		t.Fatal("edit command should have --mask-circle flag")
	}
}

func TestEditCommandHasExtendFlag(t *testing.T) {
	if editCmd.Flags().Lookup("extend") == nil {
		t.Fatal("edit command should have --extend flag")
	}
}

func TestEditDryRunBasic(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProvider(t, "sk-test")
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "make it blue", "--image", imgPath, "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "make it blue") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}

func TestEditDryRunOutpaint(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProvider(t, "sk-test")
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "more sky", "--image", imgPath, "--extend", "top=100", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run outpaint returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "more sky") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}

func TestEditDryRunInpaintRect(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProvider(t, "sk-test")
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "remove object", "--image", imgPath, "--mask-rect", "10,10,20,20", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run inpaint returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "remove object") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}

func TestEditMissingImageFile(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProvider(t, "sk-test")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", "/nonexistent.png", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("edit should error on missing image file")
	}
}

func TestEditMissingConfigError(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("edit should error when no config is provided")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestEditParseRectMask(t *testing.T) {
	x, y, w, h, err := parseRectMask("10,20,30,40")
	if err != nil {
		t.Fatalf("parseRectMask returned error: %v", err)
	}
	if x != 10 || y != 20 || w != 30 || h != 40 {
		t.Errorf("parseRectMask got x=%d y=%d w=%d h=%d, want 10 20 30 40", x, y, w, h)
	}
}

func TestEditParseRectMaskInvalid(t *testing.T) {
	_, _, _, _, err := parseRectMask("10,20,30")
	if err == nil {
		t.Fatal("parseRectMask should error on 3 parts")
	}
}

func TestEditParseCircleMask(t *testing.T) {
	cx, cy, r, err := parseCircleMask("25,25,10")
	if err != nil {
		t.Fatalf("parseCircleMask returned error: %v", err)
	}
	if cx != 25 || cy != 25 || r != 10 {
		t.Errorf("parseCircleMask got cx=%d cy=%d r=%d, want 25 25 10", cx, cy, r)
	}
}

func TestEditParseCircleMaskInvalid(t *testing.T) {
	_, _, _, err := parseCircleMask("25,25")
	if err == nil {
		t.Fatal("parseCircleMask should error on 2 parts")
	}
}

func TestPrepareEditImageNormalizesUserMaskFile(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 20, 20)

	maskPath := filepath.Join(dir, "mask.png")
	createTestPNG(t, maskPath, 5, 5)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("extend", "", "")
	flags.String("mask", maskPath, "")
	flags.String("mask-rect", "", "")
	flags.String("mask-circle", "", "")

	editPath, normalizedMaskPath, cleanup, err := prepareEditImage(imgPath, flags)
	if err != nil {
		t.Fatalf("prepareEditImage error: %v", err)
	}
	defer cleanup()

	if editPath != imgPath {
		t.Fatalf("edit path = %q, want original image path", editPath)
	}
	if normalizedMaskPath == maskPath {
		t.Fatalf("mask path should point to normalized temp mask, not raw user mask")
	}
	mask, _, err := img.ReadImage(normalizedMaskPath)
	if err != nil {
		t.Fatalf("read normalized mask: %v", err)
	}
	if mask.Bounds().Dx() != 20 || mask.Bounds().Dy() != 20 {
		t.Fatalf("normalized mask size = %dx%d, want 20x20", mask.Bounds().Dx(), mask.Bounds().Dy())
	}
}

func TestPrepareEditImageCleanupRemovesGeneratedMaskDir(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 20, 20)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("extend", "", "")
	flags.String("mask", "", "")
	flags.String("mask-rect", "1,1,5,5", "")
	flags.String("mask-circle", "", "")

	_, maskPath, cleanup, err := prepareEditImage(imgPath, flags)
	if err != nil {
		t.Fatalf("prepareEditImage error: %v", err)
	}
	tempDir := filepath.Dir(maskPath)
	if _, err := os.Stat(maskPath); err != nil {
		t.Fatalf("mask should exist before cleanup: %v", err)
	}
	cleanup()
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("temp dir should be removed after cleanup, stat err: %v", err)
	}
}

func TestPrepareEditImageCleanupRemovesOutpaintDir(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 20, 20)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("extend", "right=5", "")
	flags.String("mask", "", "")
	flags.String("mask-rect", "", "")
	flags.String("mask-circle", "", "")

	expandedPath, _, cleanup, err := prepareEditImage(imgPath, flags)
	if err != nil {
		t.Fatalf("prepareEditImage error: %v", err)
	}
	tempDir := filepath.Dir(expandedPath)
	cleanup()
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("outpaint temp dir should be removed after cleanup, stat err: %v", err)
	}
}

func TestEditCleansGeneratedMaskDirAfterUpload(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 20, 20)
	outputPath := filepath.Join(dir, "output.png")

	responseBytes, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("read response fixture: %v", err)
	}
	responseB64 := base64.StdEncoding.EncodeToString(responseBytes)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("prompt") != "remove object" {
			t.Fatalf("prompt = %q, want remove object", r.FormValue("prompt"))
		}
		if _, _, err := r.FormFile("image"); err != nil {
			t.Fatalf("image missing: %v", err)
		}
		if _, _, err := r.FormFile("mask"); err != nil {
			t.Fatalf("mask missing: %v", err)
		}
		fmt.Fprintf(w, `{"created":1,"data":[{"b64_json":%q}]}`, responseB64)
	}))
	defer server.Close()

	// Add provider, then run edit with --base-url pointing at mock server
	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)

	before, _ := filepath.Glob(filepath.Join(os.TempDir(), "potaco-mask-*"))

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{
		"edit",
		"--prompt", "remove object",
		"--image", imgPath,
		"--mask-rect", "1,1,5,5",
		"--output", outputPath,
		"--base-url", server.URL,
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("edit returned error: %v", err)
	}

	after, _ := filepath.Glob(filepath.Join(os.TempDir(), "potaco-mask-*"))
	if len(after) != len(before) {
		t.Fatalf("temp mask dirs leaked: before=%v after=%v", before, after)
	}
}

func TestEditCommandDryRunUsesAdapter(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProvider(t, "sk-test")

	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")
	createTestPNG(t, imgPath, 4, 4)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "make it blue", "--image", imgPath, "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "make it blue") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}

func TestEditWithAuthCredentials(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	// Add a provider via auth add
	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-from-auth", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "make it blue", "--image", imgPath, "--model", "gpt-image-2", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit --dry-run: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/edits") {
		t.Errorf("dry-run should contain edit endpoint, got: %q", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("dry-run should contain model, got: %q", output)
	}
}

func TestEditNoActiveProviderError(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("edit should error when no provider is configured")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestEditWithBaseUrlOverride(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")
	t.Setenv("POTACO_MODEL", "")

	var setupBuf bytes.Buffer
	rootCmd.SetOut(&setupBuf)
	rootCmd.SetErr(&setupBuf)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-stored", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "source.png")
	createTestPNG(t, imgPath, 50, 50)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--dry-run", "--base-url", "https://custom.api.com"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("edit with override: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom.api.com") {
		t.Errorf("dry-run should use overridden base-url, got: %q", output)
	}
}

func TestEditDryRunFalProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProviderForProvider(t, "fal", "fal-key", "fal-ai/flux/dev")

	tmpDir := t.TempDir()
	imgPath := tmpDir + "/test.png"
	createTestPNG(t, imgPath, 4, 4)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--dry-run", "--provider", "fal", "--base-url", "https://fal.run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Key [REDACTED]") {
		t.Errorf("dry-run should show 'Key [REDACTED]' for fal, got: %s", output)
	}
}

func TestEditDryRunVercelNotSupported(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProviderForProvider(t, "vercel", "vkey", "openai/gpt-image-2")

	tmpDir := t.TempDir()
	imgPath := tmpDir + "/test.png"
	createTestPNG(t, imgPath, 4, 4)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--dry-run", "--provider", "vercel"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for vercel edit, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention 'not supported', got: %v", err)
	}
	if !strings.Contains(err.Error(), "vercel") {
		t.Errorf("error should mention provider 'vercel', got: %v", err)
	}
}

func TestEditVercelNotSupportedNonDryRun(t *testing.T) {
	resetRootCmdFlags(t)
	resetEditCmdFlags(t)
	t.Cleanup(func() { resetEditCmdFlags(t) })
	setupAuthProviderForProvider(t, "vercel", "vkey", "openai/gpt-image-2")
	tmpDir := t.TempDir()
	imgPath := tmpDir + "/test.png"
	createTestPNG(t, imgPath, 4, 4)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	// No --dry-run: should still be rejected before touching the image.
	rootCmd.SetArgs([]string{"edit", "--prompt", "test", "--image", imgPath, "--provider", "vercel"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for vercel edit (non-dry-run), got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention 'not supported', got: %v", err)
	}
	if !strings.Contains(err.Error(), "vercel") {
		t.Errorf("error should mention provider 'vercel', got: %v", err)
	}
}

// resetEditCmdFlags restores edit subcommand flags to their defaults so that
// values set by earlier tests (e.g. --dry-run, --extend, --provider) do not
// leak in when tests run in shuffled order.
func resetEditCmdFlags(t *testing.T) {
	t.Helper()
	flags := editCmd.Flags()
	for _, name := range []string{"prompt", "image", "mask", "mask-rect", "mask-circle",
		"extend", "model", "size", "n", "response-format", "output", "output-format",
		"stdout", "provider", "base-url", "api-key", "retries", "timeout", "dry-run"} {
		flag := flags.Lookup(name)
		if flag == nil {
			t.Fatalf("resetEditCmdFlags: unknown flag %q", name)
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset edit flag %s: %v", name, err)
		}
		flag.Changed = false
	}
}

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}
