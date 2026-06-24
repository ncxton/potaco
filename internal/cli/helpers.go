package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/ngct/potaco/internal/config"
	img "github.com/ngct/potaco/internal/image"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)

// buildMergeOptions creates MergeOptions from CLI flags.
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

	return opts
}

// flagString reads a string flag, returning the flag value.
func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// flagInt reads an int flag, returning the flag value.
func flagInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

// flagFloat64 reads a float64 flag, returning the flag value.
func flagFloat64(cmd *cobra.Command, name string) float64 {
	v, _ := cmd.Flags().GetFloat64(name)
	return v
}

// flagBool reads a bool flag, returning the flag value.
func flagBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

// printDryRun prints the request payload as JSON to stdout without making an API call.
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

// processAndOutput decodes the API response images, saves them, and prints output.
func processAndOutput(cmd *cobra.Command, resp *provider.ImageResponse, model string, params map[string]any, latency int64) error {
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	stdoutMode := flagBool(cmd, "stdout")
	viewMode := flagBool(cmd, "view")
	outputPath := flagString(cmd, "output")
	outputFormat := flagString(cmd, "output-format")

	paths := make([]string, len(resp.Data))
	widths := make([]int, len(resp.Data))
	heights := make([]int, len(resp.Data))

	for i, imgData := range resp.Data {
		if imgData.B64JSON != "" {
			decoded, err := img.DecodeBase64Image(imgData.B64JSON)
			if err != nil {
				return fmt.Errorf("decode image %d: %w", i, err)
			}
			bounds := decoded.Bounds()
			widths[i] = bounds.Dx()
			heights[i] = bounds.Dy()

			path := outputPath
			if path == "" {
				path = img.AutoFilename()
			} else if len(resp.Data) > 1 {
				path = fmt.Sprintf("%s-%d%s", trimExt(outputPath), i, extOf(outputPath))
			}

			if stdoutMode && !viewMode {
				// Write raw bytes to stdout
				var buf bytes.Buffer
				switch outputFormat {
				case "jpeg", "jpg":
					jpeg.Encode(&buf, decoded, &jpeg.Options{Quality: 90})
				default:
					png.Encode(&buf, decoded)
				}
				os.Stdout.Write(buf.Bytes())
			}

			if err := img.WriteImage(decoded, path, outputFormat); err != nil {
				return fmt.Errorf("write image %d: %w", i, err)
			}
			paths[i] = path

			if viewMode {
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
		Model:     model,
		Params:    params,
		LatencyMs: latency,
	}

	outOpts := OutputOptions{
		JSON:         jsonMode,
		Stdout:       stdoutMode,
		View:         viewMode,
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

// trimExt removes the file extension from a path.
func trimExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx > 0 {
		return path[:idx]
	}
	return path
}

// extOf returns the file extension of a path, including the dot.
func extOf(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx >= 0 {
		return path[idx:]
	}
	return ""
}
