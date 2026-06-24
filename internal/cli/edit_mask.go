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

func printEditDryRun(cmd *cobra.Command, baseURL, prompt, model, imagePath string, flags *pflag.FlagSet) error {
	extendFlag, _ := flags.GetString("extend")
	maskFlag, _ := flags.GetString("mask")
	maskRectFlag, _ := flags.GetString("mask-rect")
	maskCircleFlag, _ := flags.GetString("mask-circle")

	var mode string
	var extra map[string]any

	if extendFlag != "" {
		extendCfg, err := img.ParseExtend(extendFlag)
		if err != nil {
			return fmt.Errorf("parse extend: %w", err)
		}
		mode = "outpaint"
		extra = map[string]any{"extend": extendCfg}
	} else if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" {
		mode = "inpaint"
		extra = map[string]any{
			"mask":        maskFlag,
			"mask_rect":   maskRectFlag,
			"mask_circle": maskCircleFlag,
		}
	} else {
		mode = "basic"
		extra = nil
	}

	body := map[string]any{
		"prompt": prompt,
		"model":  model,
		"image":  imagePath,
		"mode":   mode,
	}
	for k, v := range extra {
		body[k] = v
	}

	return printDryRun(cmd, "POST", baseURL+"/v1/images/edits", "multipart/form-data", body)
}

func prepareEditImage(imagePath string, flags *pflag.FlagSet) (string, string, error) {
	extendFlag, _ := flags.GetString("extend")
	maskFlag, _ := flags.GetString("mask")
	maskRectFlag, _ := flags.GetString("mask-rect")
	maskCircleFlag, _ := flags.GetString("mask-circle")

	if extendFlag != "" {
		extendCfg, err := img.ParseExtend(extendFlag)
		if err != nil {
			return "", "", fmt.Errorf("parse extend: %w", err)
		}
		expandedPath, maskPath, err := img.PrepareOutpaint(imagePath, extendCfg)
		if err != nil {
			return "", "", fmt.Errorf("prepare outpaint: %w", err)
		}
		return expandedPath, maskPath, nil
	}

	if maskFlag != "" || maskRectFlag != "" || maskCircleFlag != "" {
		if maskFlag != "" {
			if _, err := os.Stat(maskFlag); err != nil {
				return "", "", fmt.Errorf("mask file: %w", err)
			}
			return imagePath, maskFlag, nil
		}

		maskPath, err := generateMaskFile(imagePath, maskRectFlag, maskCircleFlag)
		return imagePath, maskPath, err
	}

	return imagePath, "", nil
}

func generateMaskFile(imagePath, maskRectFlag, maskCircleFlag string) (string, error) {
	srcImg, _, err := img.ReadImage(imagePath)
	if err != nil {
		return "", fmt.Errorf("read source image: %w", err)
	}
	bounds := srcImg.Bounds()

	var maskImg image.Image
	if maskRectFlag != "" {
		x, y, w, h, err := parseRectMask(maskRectFlag)
		if err != nil {
			return "", fmt.Errorf("parse mask-rect: %w", err)
		}
		maskImg, err = img.RectMask(bounds.Dx(), bounds.Dy(), x, y, w, h)
		if err != nil {
			return "", fmt.Errorf("generate rect mask: %w", err)
		}
	} else if maskCircleFlag != "" {
		cx, cy, r, err := parseCircleMask(maskCircleFlag)
		if err != nil {
			return "", fmt.Errorf("parse mask-circle: %w", err)
		}
		maskImg, err = img.CircleMask(bounds.Dx(), bounds.Dy(), cx, cy, r)
		if err != nil {
			return "", fmt.Errorf("generate circle mask: %w", err)
		}
	}

	dir, err := os.MkdirTemp("", "potaco-mask-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	maskPath := filepath.Join(dir, "mask.png")
	if err := img.WriteMask(maskImg, maskPath); err != nil {
		return "", fmt.Errorf("write mask: %w", err)
	}
	return maskPath, nil
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
