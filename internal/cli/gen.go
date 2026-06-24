package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/ngct/potaco/internal/config"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate images from a text prompt",
	RunE:  runGen,
}

func init() {
	genCmd.Flags().StringP("prompt", "p", "", "text description of the desired image(s)")
	_ = genCmd.MarkFlagRequired("prompt")

	genCmd.Flags().String("model", "", "model to use (e.g., dall-e-3)")
	genCmd.Flags().String("size", "1024x1024", "image dimensions (WxH)")
	genCmd.Flags().String("quality", "standard", "image quality (standard or hd)")
	genCmd.Flags().Int("n", 1, "number of images to generate")
	genCmd.Flags().String("style", "", "visual style (vivid or natural)")

	genCmd.Flags().Int("seed", 0, "reproducibility seed")
	genCmd.Flags().Float64("guidance-scale", 0, "guidance scale")
	genCmd.Flags().String("negative-prompt", "", "negative prompt")
	genCmd.Flags().String("response-format", "b64_json", "response format (url or b64_json)")

	genCmd.Flags().StringP("output", "o", "", "output file path")
	genCmd.Flags().String("output-format", "png", "output format (png or jpeg)")
	genCmd.Flags().Bool("view", false, "attempt terminal image display")
	genCmd.Flags().Bool("stdout", false, "pipe raw image bytes to stdout")

	genCmd.Flags().String("provider", "", "provider preset (openai, together, fal)")
	genCmd.Flags().String("base-url", "", "override API base URL")
	genCmd.Flags().String("api-key", "", "override API key")
	genCmd.Flags().Int("retries", 0, "max retry attempts")
	genCmd.Flags().Duration("timeout", 0, "request timeout")

	genCmd.Flags().Bool("dry-run", false, "validate and print request payload without calling API")

	rootCmd.AddCommand(genCmd)
}

func runGen(cmd *cobra.Command, args []string) error {
	prompt := flagString(cmd, "prompt")
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	opts := buildMergeOptions(cmd)
	cfg, err := config.Merge(opts)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	model := cfg.Model
	if cmd.Flags().Changed("model") {
		model = flagString(cmd, "model")
	}

	req := provider.GenerateRequest{
		Prompt:         prompt,
		Model:          model,
		Size:           flagString(cmd, "size"),
		Quality:        flagString(cmd, "quality"),
		N:              flagInt(cmd, "n"),
		Style:          flagString(cmd, "style"),
		ResponseFormat: flagString(cmd, "response-format"),
		Seed:           flagInt(cmd, "seed"),
		GuidanceScale:  flagFloat64(cmd, "guidance-scale"),
		NegativePrompt: flagString(cmd, "negative-prompt"),
	}

	dryRun := flagBool(cmd, "dry-run")
	if dryRun {
		return printDryRun(cmd, "POST", cfg.BaseURL+"/v1/images/generations", "application/json", req)
	}

	client := provider.NewClient(provider.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Retries: cfg.Retries,
		Timeout: cfg.Timeout,
	})

	start := time.Now()
	resp, err := client.Generate(context.Background(), req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return processAndOutput(cmd, resp, model, map[string]any{
		"size":            req.Size,
		"quality":         req.Quality,
		"n":               req.N,
		"style":           req.Style,
		"response_format": req.ResponseFormat,
	}, latency)
}
