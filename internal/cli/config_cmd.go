package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ncxton/potaco/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage provider configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
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
	configSetCmd.Flags().String("base-url", "", "base URL for the active provider")
	configSetCmd.Flags().Int("retries", 0, "max retry attempts for the active provider")
	configSetCmd.Flags().String("timeout", "", "request timeout in seconds for the active provider (e.g., 120)")
	_ = configSetCmd.Flags().MarkHidden("model")
	_ = configSetCmd.Flags().MarkHidden("base-url")
	_ = configSetCmd.Flags().MarkHidden("retries")
	_ = configSetCmd.Flags().MarkHidden("timeout")

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

	if len(args) > 0 {
		if len(args) != 2 {
			return configError(fmt.Errorf("usage: potaco config set <key> <value>"))
		}
		if err := setConfigKeyValue(cfg, args[0], args[1]); err != nil {
			return configError(err)
		}
		if err := config.SaveMultiProvider(path, cfg); err != nil {
			return configError(err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to %s\n", path)
		return nil
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
	if cmd.Flags().Changed("base-url") {
		baseURL, _ := cmd.Flags().GetString("base-url")
		pc.BaseURL = strings.TrimRight(baseURL, "/")
		changed = true
	}
	if cmd.Flags().Changed("retries") {
		retries, _ := cmd.Flags().GetInt("retries")
		pc.Retries = retries
		changed = true
	}
	if cmd.Flags().Changed("timeout") {
		timeoutStr, _ := cmd.Flags().GetString("timeout")
		timeout, err := parseTimeoutString(timeoutStr)
		if err != nil {
			return configError(err)
		}
		pc.Timeout = int(timeout.Seconds())
		changed = true
	}

	if !changed {
		return configError(fmt.Errorf("no value specified. Use 'potaco config set <key> <value>'"))
	}

	cfg.Providers[cfg.ActiveProvider] = pc

	if err := config.SaveMultiProvider(path, cfg); err != nil {
		return configError(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to %s\n", path)
	return nil
}

func setConfigKeyValue(cfg *config.MultiProviderConfig, key, value string) error {
	if key == "auto_update" {
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("auto_update must be true or false, got %q", value)
		}
		cfg.AutoUpdate = &v
		return nil
	}

	if strings.HasPrefix(key, "providers.") {
		parts := strings.Split(key, ".")
		if len(parts) == 3 && parts[1] != "" {
			return setProviderConfigValue(cfg, parts[1], parts[2], value)
		}
		if len(parts) >= 5 && parts[1] != "" && parts[2] == "models" && parts[len(parts)-1] == "edit" {
			modelID := strings.Join(parts[3:len(parts)-1], ".")
			if modelID == "" {
				return fmt.Errorf("unknown config key %q", key)
			}
			return setProviderModelEditValue(cfg, parts[1], modelID, value)
		}
		if len(parts) != 3 || parts[1] == "" {
			return fmt.Errorf("unknown config key %q", key)
		}
	}

	switch key {
	case "model", "base_url", "retries", "timeout":
		if cfg.ActiveProvider == "" {
			return fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one")
		}
		return setProviderConfigValue(cfg, cfg.ActiveProvider, key, value)
	case "model.edit":
		if cfg.ActiveProvider == "" {
			return fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one")
		}
		pc := cfg.Providers[cfg.ActiveProvider]
		if pc.Model == "" {
			return fmt.Errorf("active provider %q has no model configured", cfg.ActiveProvider)
		}
		return setProviderModelEditValue(cfg, cfg.ActiveProvider, pc.Model, value)
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
}

func setProviderConfigValue(cfg *config.MultiProviderConfig, providerName, field, value string) error {
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	pc, ok := cfg.Providers[providerName]
	if !ok {
		return fmt.Errorf("provider %q is not configured. Use 'potaco auth add %s' first", providerName, providerName)
	}
	switch field {
	case "model":
		pc.Model = value
		if providerName == cfg.ActiveProvider {
			cfg.ActiveModel = value
		}
	case "base_url":
		pc.BaseURL = strings.TrimRight(value, "/")
	case "retries":
		retries, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("retries must be a number, got %q", value)
		}
		pc.Retries = retries
	case "timeout":
		timeout, err := parseTimeoutString(value)
		if err != nil {
			return err
		}
		pc.Timeout = int(timeout.Seconds())
	default:
		return fmt.Errorf("unknown config key %q", "providers."+providerName+"."+field)
	}
	cfg.Providers[providerName] = pc
	return nil
}

func setProviderModelEditValue(cfg *config.MultiProviderConfig, providerName, modelID, value string) error {
	edit, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("model edit must be true or false, got %q", value)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	pc, ok := cfg.Providers[providerName]
	if !ok {
		return fmt.Errorf("provider %q is not configured. Use 'potaco auth add %s' first", providerName, providerName)
	}
	if pc.Models == nil {
		pc.Models = make(map[string]config.ModelConfig)
	}
	mc := pc.Models[modelID]
	mc.Edit = edit
	pc.Models[modelID] = mc
	cfg.Providers[providerName] = pc
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
		fmt.Fprintf(out, "    model: %s\n", pc.Model)
		fmt.Fprintf(out, "    base_url: %s\n", formatBaseURL(pc.BaseURL))
		fmt.Fprintf(out, "    retries: %d\n", pc.Retries)
		fmt.Fprintf(out, "    timeout: %s\n", formatTimeout(pc.Timeout))
		if len(pc.Models) > 0 {
			models := make([]string, 0, len(pc.Models))
			for model := range pc.Models {
				models = append(models, model)
			}
			sort.Strings(models)
			fmt.Fprintln(out, "    models:")
			for _, model := range models {
				fmt.Fprintf(out, "      %s: edit=%t\n", model, pc.Models[model].Edit)
			}
		}
	}
	return nil
}

// formatTimeout renders the timeout (in seconds) for display, returning
// "default" for a zero value so the output is informative.
func formatTimeout(secs int) string {
	if secs <= 0 {
		return "default"
	}
	return fmt.Sprintf("%ds", secs)
}

// formatBaseURL renders the base URL for display, returning "default"
// for a zero value so providers without an override are easy to spot.
func formatBaseURL(baseURL string) string {
	if baseURL == "" {
		return "default"
	}
	return baseURL
}
