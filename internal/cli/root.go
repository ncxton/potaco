package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ncxton/potaco/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:           "potaco",
	Short:         "Terminal image generation and editing CLI",
	Long:          `Potaco provides advanced image generation and editing inside the terminal. Connect to any OpenAI-compatible provider supporting /v1/images/generations and /v1/images/edits.`,
	Run:           func(cmd *cobra.Command, args []string) {},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if ec, ok := err.(*ExitCoder); ok {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(ec.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "output JSON metadata to stdout")
	rootCmd.PersistentFlags().Bool("verbose", false, "print retry attempts and debug info to stderr")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "force non-interactive mode (skip TUI)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ni, err := cmd.Flags().GetBool("non-interactive")
		if err != nil {
			return fmt.Errorf("read non-interactive flag: %w", err)
		}
		tui.SetNonInteractive(ni)
		return nil
	}
}
