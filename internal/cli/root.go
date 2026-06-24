package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "potaco",
	Short: "Terminal image generation and editing CLI",
	Long:  `Potaco provides advanced image generation and editing inside the terminal. Connect to any OpenAI-compatible provider supporting /v1/images/generations and /v1/images/edits.`,
	Run:   func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "output JSON metadata to stdout")
	rootCmd.PersistentFlags().Bool("verbose", false, "print retry attempts and debug info to stderr")
}
