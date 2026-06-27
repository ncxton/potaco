package cli

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func printEditDryRun(cmd *cobra.Command, baseURL, providerName, authHeader, prompt, model, imagePath string, flags *pflag.FlagSet) error {
	extendFlag, _ := flags.GetString("extend")
	maskFlag, _ := flags.GetString("mask")
	maskRectFlag, _ := flags.GetString("mask-rect")
	maskCircleFlag, _ := flags.GetString("mask-circle")

	var mode string

	if extendFlag != "" {
		if _, err := img.ParseExtend(extendFlag); err != nil {
			return fmt.Errorf("parse extend: %w", err)
		}
		mode = "outpaint"
	} else if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" {
		mode = "inpaint"
	} else {
		mode = "basic"
	}

	body := map[string]any{
		"prompt": prompt,
		"model":  model,
		"mode":   mode,
	}

	editURL := baseURL + "/v1/images/edits"
	contentType := "multipart/form-data"
	if providerName == "fal" {
		editURL = baseURL + "/" + model + "/image-to-image"
		contentType = "application/json"
		body["image_url"] = "<data:" + detectImageMIME(imagePath) + ";base64,...>"
	} else if providerName == "custom" {
		contentType = "application/json"
		if strings.HasSuffix(baseURL, "/v1") {
			editURL = baseURL + "/images/edits"
		}
		body["images"] = []map[string]any{
			{"image_url": "<data:" + detectImageMIME(imagePath) + ";base64,...>"},
		}
		// A mask is sent when any mask-producing flag is set: --mask,
		// --mask-rect, --mask-circle, or --extend (auto-generated).
		if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" || extendFlag != "" {
			if maskFlag != "" {
				body["mask"] = "<data:" + detectImageMIME(maskFlag) + ";base64,...>"
			} else {
				body["mask"] = "<data:png;base64,...>"
			}
		}
	} else {
		body["image"] = imagePath
		if strings.HasSuffix(baseURL, "/v1") {
			editURL = baseURL + "/images/edits"
		}
	}
	return printDryRun(cmd, "POST", editURL, contentType, authHeader, body)
}

// detectImageMIME returns the MIME subtype for an image file by reading
// its magic bytes. Used by dry-run output to show the data URL prefix
// without inlining the full base64 payload.
func detectImageMIME(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "octet-stream"
	}
	format := img.FormatFromBytes(data)
	if format == "" {
		return "octet-stream"
	}
	return format
}

func noopCleanup() {}

func prepareEditImage(imagePath string, flags *pflag.FlagSet) (string, string, func(), error) {
	extendFlag, _ := flags.GetString("extend")
	maskFlag, _ := flags.GetString("mask")
	maskRectFlag, _ := flags.GetString("mask-rect")
	maskCircleFlag, _ := flags.GetString("mask-circle")

	if extendFlag != "" {
		extendCfg, err := img.ParseExtend(extendFlag)
		if err != nil {
			return "", "", noopCleanup, imageUserErr(
				fmt.Sprintf("Invalid extend format: '%s'.", extendFlag),
				"Use top=N,bottom=N,left=N,right=N or all=N.",
				fmt.Errorf("parse extend: %w", err),
			)
		}
		expandedPath, maskPath, err := img.PrepareOutpaint(imagePath, extendCfg)
		if err != nil {
			return "", "", noopCleanup, imageUserErr(
				"Could not prepare the outpaint canvas.",
				"Check that the source image is a valid PNG or JPEG file.",
				fmt.Errorf("prepare outpaint: %w", err),
			)
		}
		return expandedPath, maskPath, func() { _ = os.RemoveAll(filepath.Dir(expandedPath)) }, nil
	}

	if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" {
		if maskFlag != "" {
			if _, err := os.Stat(maskFlag); err != nil {
				return "", "", noopCleanup, imageUserErr(
					fmt.Sprintf("The mask file '%s' does not exist.", maskFlag),
					"Check the path and try again.",
					fmt.Errorf("mask file: %w", err),
				)
			}
			maskPath, cleanup, err := normalizeMaskFile(imagePath, maskFlag)
			if err != nil {
				cleanup()
				return "", "", noopCleanup, err
			}
			return imagePath, maskPath, cleanup, nil
		}

		maskPath, cleanup, err := generateMaskFile(imagePath, maskRectFlag, maskCircleFlag)
		return imagePath, maskPath, cleanup, err
	}

	return imagePath, "", noopCleanup, nil
}

func normalizeMaskFile(imagePath, maskPath string) (string, func(), error) {
	srcImg, _, err := img.ReadImage(imagePath)
	if err != nil {
		return "", noopCleanup, fmt.Errorf("read source image: %w", err)
	}
	bounds := srcImg.Bounds()
	maskImg, err := img.LoadMaskFile(maskPath, bounds.Dx(), bounds.Dy())
	if err != nil {
		return "", noopCleanup, fmt.Errorf("load mask file: %w", err)
	}
	return writeTempMask(maskImg)
}

func generateMaskFile(imagePath, maskRectFlag, maskCircleFlag string) (string, func(), error) {
	srcImg, _, err := img.ReadImage(imagePath)
	if err != nil {
		return "", noopCleanup, fmt.Errorf("read source image: %w", err)
	}
	bounds := srcImg.Bounds()

	var maskImg image.Image
	if maskRectFlag != "" {
		x, y, w, h, err := parseRectMask(maskRectFlag)
		if err != nil {
			return "", noopCleanup, fmt.Errorf("parse mask-rect: %w", err)
		}
		maskImg, err = img.RectMask(bounds.Dx(), bounds.Dy(), x, y, w, h)
		if err != nil {
			return "", noopCleanup, fmt.Errorf("generate rect mask: %w", err)
		}
	} else if maskCircleFlag != "" {
		cx, cy, r, err := parseCircleMask(maskCircleFlag)
		if err != nil {
			return "", noopCleanup, fmt.Errorf("parse mask-circle: %w", err)
		}
		maskImg, err = img.CircleMask(bounds.Dx(), bounds.Dy(), cx, cy, r)
		if err != nil {
			return "", noopCleanup, fmt.Errorf("generate circle mask: %w", err)
		}
	}

	return writeTempMask(maskImg)
}

func writeTempMask(maskImg image.Image) (string, func(), error) {
	dir, err := os.MkdirTemp("", "potaco-mask-*")
	if err != nil {
		return "", noopCleanup, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	maskPath := filepath.Join(dir, "mask.png")
	if err := img.WriteMask(maskImg, maskPath); err != nil {
		cleanup()
		return "", noopCleanup, fmt.Errorf("write mask: %w", err)
	}
	return maskPath, cleanup, nil
}

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
