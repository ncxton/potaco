package cli

import (
	"encoding/json"
	"fmt"
	"os"

	img "github.com/ncxton/potaco/internal/image"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [path]",
	Short: "Print metadata about an image file",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	path := args[0]

	stat, err := os.Stat(path)
	if err != nil {
		return imageError(fmt.Errorf("file: %w", err))
	}

	image, format, err := img.ReadImage(path)
	if err != nil {
		return imageError(fmt.Errorf("read image: %w", err))
	}

	bounds := image.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	fileSize := stat.Size()
	colorModel := fmt.Sprintf("%v", image.ColorModel())

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")

	if jsonMode {
		output := map[string]any{
			"path":        path,
			"format":      format,
			"width":       width,
			"height":      height,
			"file_size":   fileSize,
			"color_model": colorModel,
		}
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "File:       %s\n", path)
		fmt.Fprintf(cmd.OutOrStdout(), "Format:     %s\n", format)
		fmt.Fprintf(cmd.OutOrStdout(), "Dimensions: %dx%d\n", width, height)
		fmt.Fprintf(cmd.OutOrStdout(), "File size:  %d bytes\n", fileSize)
		fmt.Fprintf(cmd.OutOrStdout(), "Color:      %s\n", colorModel)
	}

	return nil
}
