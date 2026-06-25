package cli

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ncxton/potaco/internal/adapter"
	_ "github.com/ncxton/potaco/internal/adapter/openai" // register openai adapter
	"github.com/ncxton/potaco/internal/config"
	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/cobra"
)

// providerPreset holds known defaults for a provider preset. This is a
// temporary Phase 1 shim that mirrors internal/provider/presets.go so the
// CLI layer no longer imports the provider package. It will be replaced
// in Phase 2 when adapters expose their own preset metadata.
type providerPreset struct {
	BaseURL      string
	DefaultModel string
}

var providerPresets = map[string]providerPreset{
	"openai":   {BaseURL: "https://api.openai.com", DefaultModel: "gpt-image-2"},
	"together": {BaseURL: "https://api.together.ai", DefaultModel: "black-forest-labs/flux-1"},
	"fal":      {BaseURL: "https://fal.run", DefaultModel: "fal-ai/flux"},
}

func getProviderPreset(name string) (providerPreset, bool) {
	p, ok := providerPresets[name]
	return p, ok
}

func buildMergeOptions(cmd *cobra.Command) config.MergeOptions {
	opts := config.MergeOptions{}

	if cmd.Flags().Changed("base-url") {
		v, _ := cmd.Flags().GetString("base-url")
		opts.BaseURL = &v
	}
	if cmd.Flags().Changed("api-key") {
		v, _ := cmd.Flags().GetString("api-key")
		opts.APIKey = &v
	}
	if cmd.Flags().Changed("model") {
		v, _ := cmd.Flags().GetString("model")
		opts.Model = &v
	}
	if cmd.Flags().Changed("retries") {
		v, _ := cmd.Flags().GetInt("retries")
		opts.Retries = &v
	}
	if cmd.Flags().Changed("timeout") {
		v, _ := cmd.Flags().GetDuration("timeout")
		opts.Timeout = &v
	}
	if cmd.Flags().Changed("provider") {
		v, _ := cmd.Flags().GetString("provider")
		opts.Provider = &v
	}

	// Apply provider preset defaults for BaseURL and Model when not
	// already set by higher-precedence CLI flags (--base-url, --model).
	// This keeps provider knowledge in the CLI layer rather than the
	// config package.
	if opts.Provider != nil {
		preset, ok := getProviderPreset(*opts.Provider)
		if ok {
			if opts.BaseURL == nil {
				opts.BaseURL = &preset.BaseURL
			}
			if opts.Model == nil {
				opts.Model = &preset.DefaultModel
			}
		}
	}

	return opts
}

func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func flagInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

func flagFloat64(cmd *cobra.Command, name string) float64 {
	v, _ := cmd.Flags().GetFloat64(name)
	return v
}

func flagBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func printDryRun(cmd *cobra.Command, method, url, contentType string, body any) error {
	bodyJSON, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run body: %w", err)
	}

	dryRunOutput := map[string]any{
		"method":       method,
		"url":          url,
		"content_type": contentType,
		"headers": map[string]string{
			"Authorization": "Bearer [REDACTED]",
		},
		"body": json.RawMessage(bodyJSON),
	}

	output, err := json.MarshalIndent(dryRunOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run output: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(output))
	return nil
}

type outputContext struct {
	resp    *adapter.GenerateResponse
	model   string
	params  map[string]any
	latency int64
}

// adapterForProvider resolves the adapter for the given provider name and
// config. In Phase 1, this is a transition helper that maps the old config
// model to the adapter registry.
func adapterForProvider(cfg *config.Config) (adapter.Adapter, error) {
	providerName := cfg.Provider
	if providerName == "" {
		providerName = "openai" // default to openai for backward compat
	}

	opts := adapter.AdapterOpts{
		BaseURL: cfg.BaseURL,
	}
	if cfg.Timeout > 0 {
		opts.Timeout = cfg.Timeout.String()
	}

	return adapter.Get(providerName, cfg.APIKey, opts)
}

func processAndOutput(cmd *cobra.Command, octx outputContext) error {
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	stdoutMode := flagBool(cmd, "stdout")
	viewMode := flagBool(cmd, "view")
	outputPath := flagString(cmd, "output")
	outputFormat := flagString(cmd, "output-format")
	explicitOutput := outputPath != ""
	effectiveView := viewMode && !jsonMode && !stdoutMode

	paths := make([]string, len(octx.resp.Data))
	widths := make([]int, len(octx.resp.Data))
	heights := make([]int, len(octx.resp.Data))
	autoPath := ""
	if !explicitOutput && !stdoutMode && len(octx.resp.Data) > 0 {
		autoPath = img.AutoFilename()
	}

	for i, imgData := range octx.resp.Data {
		if imgData.B64JSON != "" {
			decoded, err := img.DecodeBase64Image(imgData.B64JSON)
			if err != nil {
				return fmt.Errorf("decode image %d: %w", i, err)
			}
			bounds := decoded.Bounds()
			widths[i] = bounds.Dx()
			heights[i] = bounds.Dy()

			path := outputPath
			if stdoutMode && !explicitOutput {
				path = "stdout"
			} else if path == "" {
				path = autoPath
				if len(octx.resp.Data) > 1 {
					path = fmt.Sprintf("%s-%d%s", trimExt(autoPath), i, extOf(autoPath))
				}
			} else if len(octx.resp.Data) > 1 {
				path = fmt.Sprintf("%s-%d%s", trimExt(outputPath), i, extOf(outputPath))
			}

			if stdoutMode {
				switch outputFormat {
				case "jpeg", "jpg":
					if err := jpeg.Encode(os.Stdout, decoded, &jpeg.Options{Quality: 90}); err != nil {
						return fmt.Errorf("encode image %d to stdout: %w", i, err)
					}
				default:
					if err := png.Encode(os.Stdout, decoded); err != nil {
						return fmt.Errorf("encode image %d to stdout: %w", i, err)
					}
				}
			}

			if !stdoutMode || explicitOutput {
				if err := img.WriteImage(decoded, path, outputFormat); err != nil {
					return fmt.Errorf("write image %d: %w", i, err)
				}
			}
			paths[i] = path

			if effectiveView {
				output := img.DisplayInTerminal(decoded, path)
				fmt.Fprintln(cmd.OutOrStdout(), output)
			}
		} else if imgData.URL != "" {
			paths[i] = imgData.URL
		}
	}

	result := OutputResult{
		Paths:     paths,
		Format:    outputFormat,
		Widths:    widths,
		Heights:   heights,
		Model:     octx.model,
		Params:    octx.params,
		LatencyMs: octx.latency,
	}

	outOpts := OutputOptions{
		JSON:         jsonMode,
		Stdout:       stdoutMode,
		View:         effectiveView,
		OutputPath:   outputPath,
		OutputFormat: outputFormat,
	}

	if !stdoutMode {
		output, err := FormatResult(result, outOpts)
		if err != nil {
			return fmt.Errorf("format output: %w", err)
		}
		if output != "" {
			fmt.Fprintln(cmd.OutOrStdout(), output)
		}
	}

	return nil
}

func trimExt(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path
	}
	return path[:len(path)-len(ext)]
}

func extOf(path string) string {
	return filepath.Ext(path)
}
