package cli

import (
	"fmt"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use [provider]",
	Short: "Switch the active provider and optionally set the model",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUse,
}

func init() {
	useCmd.Flags().String("model", "", "set the model for this provider")
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return configError(fmt.Errorf("specify a provider: potaco use <provider>"))
	}

	providerName := args[0]
	model, _ := cmd.Flags().GetString("model")

	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	if err := mgr.SetActiveProvider(providerName, model); err != nil {
		return configError(fmt.Errorf("switch provider: %w", err))
	}

	if model != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Switched to provider '%s' with model '%s'.\n", providerName, model)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Switched to provider '%s'.\n", providerName)
	}
	return nil
}
