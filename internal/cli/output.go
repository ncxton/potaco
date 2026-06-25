package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OutputOptions controls how results are formatted for display.
type OutputOptions struct {
	JSON         bool
	Stdout       bool
	OutputPath   string
	OutputFormat string
}

// OutputResult holds the metadata about generated/edited images.
type OutputResult struct {
	Paths     []string
	Format    string
	Widths    []int
	Heights   []int
	Model     string
	Params    map[string]any
	LatencyMs int64
}

// FormatResult formats the result for stdout display based on the output options.
// In stdout mode, returns empty string (raw bytes are handled separately).
func FormatResult(result OutputResult, opts OutputOptions) (string, error) {
	if opts.Stdout {
		return "", nil // raw bytes handled separately
	}

	if opts.JSON {
		return formatJSON(result)
	}

	// Default: human-friendly text
	var lines []string
	for _, path := range result.Paths {
		lines = append(lines, fmt.Sprintf("Saved to: %s", path))
	}
	return strings.Join(lines, "\n"), nil
}

// formatJSON produces JSON output. For a single image, an object; for
// multiple images, an array of objects.
func formatJSON(result OutputResult) (string, error) {
	if len(result.Paths) == 1 {
		obj := map[string]any{
			"path":       result.Paths[0],
			"format":     result.Format,
			"width":      result.Widths[0],
			"height":     result.Heights[0],
			"model":      result.Model,
			"params":     result.Params,
			"latency_ms": result.LatencyMs,
		}
		b, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal JSON: %w", err)
		}
		return string(b), nil
	}

	// Multiple images: array
	arr := make([]map[string]any, len(result.Paths))
	for i, path := range result.Paths {
		arr[i] = map[string]any{
			"path":       path,
			"format":     result.Format,
			"width":      result.Widths[i],
			"height":     result.Heights[i],
			"model":      result.Model,
			"params":     result.Params,
			"latency_ms": result.LatencyMs,
		}
	}
	b, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON array: %w", err)
	}
	return string(b), nil
}
