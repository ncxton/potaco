package cli

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ngct/potaco/internal/config"
	img "github.com/ngct/potaco/internal/image"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit an existing image",
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().StringP("prompt", "p", "", "text description of the edit")
	_ = editCmd.MarkFlagRequired("prompt")

	editCmd.Flags().String("image", "", "path to source image file")
	_ = editCmd.MarkFlagRequired("image")

	// Mask flags
	editCmd.Flags().String("mask", "", "path to mask image file (white=edit, black=keep)")
	editCmd.Flags().String("mask-rect", "", "rectangular mask: x,y,w,h in pixels")
	editCmd.Flags().String("mask-circle", "", "circular mask: x,y,r in pixels")
	editCmd.Flags().String("extend", "", "outpaint extension: top=N,bottom=N,left=N,right=N or all=N")

	// Shared flags from gen
	editCmd.Flags().String("model", "", "model to use")
	editCmd.Flags().String("size", "1024x1024", "image dimensions (WxH)")
	editCmd.Flags().Int("n", 1, "number of images to generate")
	editCmd.Flags().String("response-format", "b64_json", "response format (url or b64_json)")

	// Output flags
	editCmd.Flags().StringP("output", "o", "", "output file path")
	editCmd.Flags().String("output-format", "png", "output format (png or jpeg)")
	editCmd.Flags().Bool("view", false, "attempt terminal image display")
	editCmd.Flags().Bool("stdout", false, "pipe raw image bytes to stdout")

	// Provider override flags
	editCmd.Flags().String("provider", "", "provider preset (openai, together, fal)")
	editCmd.Flags().String("base-url", "", "override API base URL")
	editCmd.Flags().String("api-key", "", "override API key")
	editCmd.Flags().Int("retries", 0, "max retry attempts")
	editCmd.Flags().Duration("timeout", 0, "request timeout")

	// Mode flags
	editCmd.Flags().Bool("dry-run", false, "validate and print request payload without calling API")

	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	prompt := flagString(cmd, "prompt")
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	imagePath := flagString(cmd, "image")
	if imagePath == "" {
		return fmt.Errorf("image path is required")
	}

	// Check image file exists
	if _, err := os.Stat(imagePath); err != nil {
		return fmt.Errorf("image file: %w", err)
	}

	opts := buildMergeOptions(cmd)
	cfg, err := config.Merge(opts)
	if err != nil {
		return configError(fmt.Errorf("config: %w", err))
	}

	model := cfg.Model
	if cmd.Flags().Changed("model") {
		model = flagString(cmd, "model")
	}

	// Determine edit mode
	extendFlag := flagString(cmd, "extend")
	maskFlag := flagString(cmd, "mask")
	maskRectFlag := flagString(cmd, "mask-rect")
	maskCircleFlag := flagString(cmd, "mask-circle")

	dryRun := flagBool(cmd, "dry-run")

	// Prepare the image and mask for the edit request
	editImagePath := imagePath
	maskPath := ""

	if extendFlag != "" {
		// Outpaint mode: expand canvas and generate mask internally
		extendCfg, err := img.ParseExtend(extendFlag)
		if err != nil {
			return fmt.Errorf("parse extend: %w", err)
		}

		if dryRun {
			return printDryRun(cmd, "POST", cfg.BaseURL+"/v1/images/edits", "multipart/form-data", map[string]any{
				"prompt": prompt,
				"model":  model,
				"image":  imagePath,
				"extend": extendCfg,
				"mode":   "outpaint",
			})
		}

		expandedPath, generatedMaskPath, err := img.PrepareOutpaint(imagePath, extendCfg)
		if err != nil {
			return fmt.Errorf("prepare outpaint: %w", err)
		}
		editImagePath = expandedPath
		maskPath = generatedMaskPath
	} else if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" {
		// Inpaint mode: generate mask if geometric flags are used
		if dryRun {
			return printDryRun(cmd, "POST", cfg.BaseURL+"/v1/images/edits", "multipart/form-data", map[string]any{
				"prompt":      prompt,
				"model":       model,
				"image":       imagePath,
				"mask":        maskFlag,
				"mask_rect":   maskRectFlag,
				"mask_circle": maskCircleFlag,
				"mode":        "inpaint",
			})
		}

		if maskFlag != "" {
			maskPath = maskFlag
			// Verify mask file is a valid image
			if _, err := os.Stat(maskFlag); err != nil {
				return fmt.Errorf("mask file: %w", err)
			}
		} else {
			// Generate mask from geometric flags
			srcImg, _, err := img.ReadImage(imagePath)
			if err != nil {
				return fmt.Errorf("read source image: %w", err)
			}
			bounds := srcImg.Bounds()

			var maskImg image.Image
			if maskRectFlag != "" {
				x, y, w, h, err := parseRectMask(maskRectFlag)
				if err != nil {
					return fmt.Errorf("parse mask-rect: %w", err)
				}
				maskImg, err = img.RectMask(bounds.Dx(), bounds.Dy(), x, y, w, h)
				if err != nil {
					return fmt.Errorf("generate rect mask: %w", err)
				}
			} else if maskCircleFlag != "" {
				cx, cy, r, err := parseCircleMask(maskCircleFlag)
				if err != nil {
					return fmt.Errorf("parse mask-circle: %w", err)
				}
				maskImg, err = img.CircleMask(bounds.Dx(), bounds.Dy(), cx, cy, r)
				if err != nil {
					return fmt.Errorf("generate circle mask: %w", err)
				}
			}

			// Write mask to temp file
			dir, err := os.MkdirTemp("", "potaco-mask-*")
			if err != nil {
				return fmt.Errorf("create temp dir: %w", err)
			}
			maskPath = filepath.Join(dir, "mask.png")
			if err := img.WriteMask(maskImg, maskPath); err != nil {
				return fmt.Errorf("write mask: %w", err)
			}
		}
	} else {
		// Basic edit mode
		if dryRun {
			return printDryRun(cmd, "POST", cfg.BaseURL+"/v1/images/edits", "multipart/form-data", map[string]any{
				"prompt": prompt,
				"model":  model,
				"image":  imagePath,
				"mode":   "basic",
			})
		}
	}

	// Build edit request
	req := provider.EditRequest{
		Prompt:         prompt,
		Model:          model,
		N:              flagInt(cmd, "n"),
		Size:           flagString(cmd, "size"),
		ResponseFormat: flagString(cmd, "response-format"),
		ImagePath:      editImagePath,
		MaskPath:       maskPath,
	}

	client := provider.NewClient(provider.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Retries: cfg.Retries,
		Timeout: cfg.Timeout,
	})

	start := time.Now()
	resp, err := client.Edit(context.Background(), req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return apiError(fmt.Errorf("edit: %w", err))
	}

	if err := processAndOutput(cmd, resp, model, map[string]any{
		"mode":            "edit",
		"image":           imagePath,
		"size":            req.Size,
		"n":               req.N,
		"response_format": req.ResponseFormat,
	}, latency); err != nil {
		return imageError(err)
	}
	return nil
}

// parseRectMask parses "x,y,w,h" into four ints.
func parseRectMask(s string) (x, y, w, h int, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("expected x,y,w,h, got %d parts", len(parts))
	}
	x, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse x: %w", err)
	}
	y, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse y: %w", err)
	}
	w, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse w: %w", err)
	}
	h, err = strconv.Atoi(parts[3])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse h: %w", err)
	}
	return x, y, w, h, nil
}

// parseCircleMask parses "cx,cy,r" into three ints.
func parseCircleMask(s string) (cx, cy, r int, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected cx,cy,r, got %d parts", len(parts))
	}
	cx, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse cx: %w", err)
	}
	cy, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse cy: %w", err)
	}
	r, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse r: %w", err)
	}
	return cx, cy, r, nil
}
