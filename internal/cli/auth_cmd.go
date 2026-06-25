package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage provider credentials",
}

var authAddCmd = &cobra.Command{
	Use:   "add <provider>",
	Short: "Connect to a provider by storing its API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthAdd,
}

func init() {
	authAddCmd.Flags().String("api-key", "", "API key for the provider")
	authAddCmd.Flags().Bool("force", false, "skip provider verification")
	authAddCmd.Flags().String("model", "", "override the default model for this provider")

	authCmd.AddCommand(authAddCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Check provider is a known adapter.
	available := adapter.List()
	known := false
	for _, name := range available {
		if name == providerName {
			known = true
			break
		}
	}
	if !known {
		return configError(fmt.Errorf("unknown provider: %s (available: %v)", providerName, available))
	}

	// Get API key from --api-key flag or POTACO_API_KEY env var.
	apiKey, _ := cmd.Flags().GetString("api-key")
	if apiKey == "" {
		apiKey = os.Getenv("POTACO_API_KEY")
	}
	if apiKey == "" {
		return configError(fmt.Errorf("API key required: use --api-key or set POTACO_API_KEY"))
	}

	// Create auth manager backed by default credential/config paths.
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	// Verify provider connectivity unless --force.
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{})
		if err != nil {
			return configError(fmt.Errorf("create adapter: %w", err))
		}
		if err := ad.Verify(context.Background()); err != nil {
			return apiError(fmt.Errorf("verification failed: %w\nUse --force to add anyway", err))
		}
	}

	// Store the credential and set the provider as active.
	if err := mgr.Add(providerName, apiKey, force); err != nil {
		return configError(fmt.Errorf("add provider: %w", err))
	}

	// Override the default model if --model was specified.
	model, _ := cmd.Flags().GetString("model")
	if model != "" {
		if err := mgr.SetActiveProvider(providerName, model); err != nil {
			return configError(fmt.Errorf("set model: %w", err))
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' added successfully.\n", providerName)
	fmt.Fprintf(cmd.OutOrStdout(), "Use 'potaco use %s' to switch to it.\n", providerName)
	return nil
}
