package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/ncxton/potaco/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage provider configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values for the active provider",
	RunE:  runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

func init() {
	configSetCmd.Flags().String("model", "", "model for the active provider")
	configSetCmd.Flags().Int("retries", 0, "max retry attempts for the active provider")
	configSetCmd.Flags().Duration("timeout", 0, "request timeout for the active provider (e.g., 120s)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		return configError(err)
	}

	if cfg.ActiveProvider == "" {
		return configError(fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one"))
	}

	pc, ok := cfg.Providers[cfg.ActiveProvider]
	if !ok {
		return configError(fmt.Errorf("active provider %q has no config entry. Use 'potaco auth add %s' first", cfg.ActiveProvider, cfg.ActiveProvider))
	}

	changed := false
	if cmd.Flags().Changed("model") {
		model, _ := cmd.Flags().GetString("model")
		pc.Model = model
		cfg.ActiveModel = model
		changed = true
	}
	if cmd.Flags().Changed("retries") {
		retries, _ := cmd.Flags().GetInt("retries")
		pc.Retries = retries
		changed = true
	}
	if cmd.Flags().Changed("timeout") {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		pc.Timeout = timeout
		changed = true
	}

	if !changed {
		return configError(fmt.Errorf("no flags specified. Use --model, --retries, or --timeout"))
	}

	cfg.Providers[cfg.ActiveProvider] = pc

	if err := config.SaveMultiProvider(path, cfg); err != nil {
		return configError(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to %s\n", path)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	cfg, err := config.LoadMultiProvider(path)
	if err != nil {
		return configError(err)
	}

	// LoadMultiProvider returns an empty config (not an error) when the
	// file does not exist. Treat an empty config as "not configured".
	if cfg.ActiveProvider == "" && len(cfg.Providers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No configuration found. Use 'potaco auth add <provider>' to connect.")
		return nil
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Config file: %s\n\n", path)
	fmt.Fprintf(out, "Active provider: %s\n", cfg.ActiveProvider)
	fmt.Fprintf(out, "Active model:    %s\n", cfg.ActiveModel)
	fmt.Fprintln(out)

	if len(cfg.Providers) == 0 {
		return nil
	}

	// Print providers in alphabetical order with an (active) marker.
	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintln(out, "Providers:")
	for _, name := range names {
		pc := cfg.Providers[name]
		active := ""
		if name == cfg.ActiveProvider {
			active = " (active)"
		}
		fmt.Fprintf(out, "  %s%s\n", name, active)
		fmt.Fprintf(out, "    model:   %s\n", pc.Model)
		fmt.Fprintf(out, "    retries: %d\n", pc.Retries)
		fmt.Fprintf(out, "    timeout: %s\n", formatTimeout(pc.Timeout))
	}
	return nil
}

// formatTimeout renders a duration for display, returning "default" for a
// zero value so the output is informative rather than showing "0s".
func formatTimeout(d time.Duration) string {
	if d <= 0 {
		return "default"
	}
	return d.String()
}
