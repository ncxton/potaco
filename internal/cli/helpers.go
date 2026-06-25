package cli

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ncxton/potaco/internal/adapter"
	_ "github.com/ncxton/potaco/internal/adapter/fal"    // register fal adapter
	_ "github.com/ncxton/potaco/internal/adapter/openai" // register openai adapter
	_ "github.com/ncxton/potaco/internal/adapter/vercel" // register vercel adapter
	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/cobra"
)

// providerPreset holds known defaults for a provider preset. This is a
// temporary Phase 1 shim so the CLI layer no longer needs a separate
// provider package. It will be replaced in Phase 2 when adapters expose
// their own preset metadata.
type providerPreset struct {
	BaseURL      string
	DefaultModel string
}

var providerPresets = map[string]providerPreset{
	"openai": {BaseURL: "https://api.openai.com", DefaultModel: "gpt-image-2"},
	"fal":    {BaseURL: "https://fal.run", DefaultModel: "fal-ai/flux/dev"},
	"vercel": {BaseURL: "https://ai-gateway.vercel.sh/v1", DefaultModel: "openai/gpt-image-2"},
}

func getProviderPreset(name string) (providerPreset, bool) {
	p, ok := providerPresets[name]
	return p, ok
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

func printDryRun(cmd *cobra.Command, method, url, contentType, authHeader string, body any) error {
	bodyJSON, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dry-run body: %w", err)
	}

	dryRunOutput := map[string]any{
		"method":       method,
		"url":          url,
		"content_type": contentType,
		"headers": map[string]string{
			"Authorization": authHeader,
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

func processAndOutput(cmd *cobra.Command, octx outputContext) error {
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	stdoutMode := flagBool(cmd, "stdout")
	outputPath := flagString(cmd, "output")
	outputFormat := flagString(cmd, "output-format")
	explicitOutput := outputPath != ""

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
				return imageUserErr(
					"Could not decode the image returned by the provider.",
					"The provider may have returned an unsupported image format. Try a different model or check '~/.potaco/debug.log' for details.",
					fmt.Errorf("decode image %d: %w", i, err),
				)
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
						return imageUserErr(
							"Could not encode the image for stdout.",
							"",
							fmt.Errorf("encode image %d to stdout: %w", i, err),
						)
					}
				default:
					if err := png.Encode(os.Stdout, decoded); err != nil {
						return imageUserErr(
							"Could not encode the image for stdout.",
							"",
							fmt.Errorf("encode image %d to stdout: %w", i, err),
						)
					}
				}
			}

			if !stdoutMode || explicitOutput {
				if err := img.WriteImage(decoded, path, outputFormat); err != nil {
					return imageUserErr(
						fmt.Sprintf("Cannot save the image to '%s'.", path),
						"Make sure the path is a file (not a directory) and you have write permissions.",
						fmt.Errorf("write image %d: %w", i, err),
					)
				}
			}
			paths[i] = path
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
		OutputPath:   outputPath,
		OutputFormat: outputFormat,
	}

	if !stdoutMode {
		output, err := FormatResult(result, outOpts)
		if err != nil {
			return imageUserErr(
				"Could not format the output.",
				"",
				fmt.Errorf("format output: %w", err),
			)
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
