package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
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
	editCmd.Flags().Bool("stdout", false, "pipe raw image bytes to stdout")

	// Provider override flags
	editCmd.Flags().String("provider", "", "provider preset (openai, fal, vercel)")
	editCmd.Flags().String("base-url", "", "override API base URL")
	editCmd.Flags().String("api-key", "", "override API key")
	editCmd.Flags().Int("retries", 0, "max retry attempts")
	editCmd.Flags().String("timeout", "", "request timeout in seconds")

	// Mode flags
	editCmd.Flags().Bool("dry-run", false, "validate and print request payload without calling API")

	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	prompt := flagString(cmd, "prompt")
	if prompt == "" {
		return configUserErr("A prompt is required.", "Use 'potaco edit --prompt \"your description\"'.", fmt.Errorf("prompt cannot be empty"))
	}

	imagePath := flagString(cmd, "image")
	if imagePath == "" {
		return configUserErr("An image file is required.", "Use --image to specify the source image path.", fmt.Errorf("image path is required"))
	}

	if _, err := os.Stat(imagePath); err != nil {
		return configUserErr(
			fmt.Sprintf("The file '%s' does not exist.", imagePath),
			"Check the path and try again.",
			fmt.Errorf("image file: %w", err),
		)
	}

	if !flagBool(cmd, "stdout") {
		outputPath := flagString(cmd, "output")
		if ue := validateOutputPath(outputPath); ue != nil {
			return ue
		}
	}

	resolved, err := resolveAdapterForCommand(cmd)
	if err != nil {
		return err
	}
	model := resolved.Model

	if resolved.Adapter.Name() == "vercel" {
		return apiUserErr(
			"Image editing is not supported by the Vercel AI Gateway provider.",
			"Use 'potaco use openai' or 'potaco use fal' to switch to a provider that supports editing.",
			fmt.Errorf("image editing is not supported by the Vercel AI Gateway provider"),
		)
	}

	dryRun := flagBool(cmd, "dry-run")
	editImagePath := imagePath
	maskPath := ""

	if dryRun {
		authHeader := resolved.Adapter.AuthHeader("[REDACTED]")
		return printEditDryRun(cmd, resolved.BaseURL, resolved.Adapter.Name(), authHeader, prompt, model, imagePath, cmd.Flags())
	}

	cleanup := noopCleanup
	editImagePath, maskPath, cleanup, err = prepareEditImage(imagePath, cmd.Flags())
	if err != nil {
		return imageError(err)
	}
	defer cleanup()

	req := adapter.EditRequest{
		Prompt:         prompt,
		Model:          model,
		N:              flagInt(cmd, "n"),
		Size:           flagString(cmd, "size"),
		ResponseFormat: flagString(cmd, "response-format"),
		ImagePath:      editImagePath,
		MaskPath:       maskPath,
	}

	sp := startSpinner(cmd, "Editing image...")
	start := time.Now()
	resp, err := resolved.Adapter.Edit(context.Background(), req)
	sp.stop()
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return apiUserErr(
			"Image editing failed.",
			"Check your API key, network connection, and model name.",
			fmt.Errorf("edit: %w", err),
		)
	}

	if err := processAndOutput(cmd, outputContext{
		resp:  resp,
		model: model,
		params: map[string]any{
			"mode":            "edit",
			"image":           imagePath,
			"size":            req.Size,
			"n":               req.N,
			"response_format": req.ResponseFormat,
		},
		latency: latency,
	}); err != nil {
		return imageError(err)
	}
	return nil
}
