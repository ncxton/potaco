package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/spf13/cobra"
)

// resolvedConfig holds the adapter, model, and base URL resolved from the
// auth manager, applying CLI flag and env var overrides. BaseURL is needed
// for dry-run output, which prints the full endpoint URL without calling
// the adapter.
type resolvedConfig struct {
	Adapter adapter.Adapter
	Model   string
	BaseURL string
}

// resolveAdapterForCommand resolves the active provider, credential, and
// model from the auth manager, applying CLI flag and env var overrides in
// this precedence order:
//
//	--provider flag > POTACO_PROVIDER env > active_provider from config
//	--api-key flag  > POTACO_API_KEY env  > credential store
//	--model flag    > POTACO_MODEL env   > provider model from config > active_model from config
//	--base-url flag > POTACO_BASE_URL env > config.providers[provider].base_url > built-in provider preset
func resolveAdapterForCommand(cmd *cobra.Command) (*resolvedConfig, error) {
	mgr, err := auth.New()
	if err != nil {
		return nil, configUserErr(
			"Could not load configuration.",
			"Check that ~/.potaco/ is readable.",
			fmt.Errorf("init auth: %w", err),
		)
	}

	providerName, err := resolveProvider(cmd, mgr)
	if err != nil {
		return nil, err
	}

	apiKey, err := resolveAPIKey(cmd, mgr, providerName)
	if err != nil {
		return nil, err
	}

	cfg, _ := mgr.LoadConfig()
	pc := config.ProviderConfig{}
	if cfg != nil {
		if configured, ok := cfg.Providers[providerName]; ok {
			pc = configured
		}
	}
	model := resolveModel(cmd, cfg, providerName)
	providerType := config.ResolveProviderType(providerName, pc)
	adapterType := config.AdapterType(providerType)
	baseURL := resolveBaseURL(cmd, providerName, cfg)

	retries, timeout, err := resolveRetriesTimeout(cmd, cfg, providerName)
	if err != nil {
		return nil, err
	}

	if authAddRequiresBaseURL(providerName, providerType) && baseURL == "" {
		return nil, configUserErr(
			"A base URL is required for this provider.",
			fmt.Sprintf("Use --base-url, set POTACO_BASE_URL, or run 'potaco config set providers.%s.base_url <url>'.", providerName),
			fmt.Errorf("base URL required for provider %s", providerName),
		)
	}

	opts := adapter.AdapterOpts{
		BaseURL: baseURL,
		Retries: retries,
	}
	if timeout > 0 {
		opts.Timeout = timeout
	}

	ad, err := adapter.Get(adapterType, apiKey, opts)
	if err != nil {
		return nil, configUserErr(
			fmt.Sprintf("Could not connect to provider '%s'.", providerName),
			"Check that the provider name is correct. Use 'potaco auth list' to see connected providers.",
			fmt.Errorf("create adapter: %w", err),
		)
	}

	return &resolvedConfig{
		Adapter: ad,
		Model:   model,
		BaseURL: baseURL,
	}, nil
}

func resolveProvider(cmd *cobra.Command, mgr *auth.AuthManager) (string, error) {
	if cmd.Flags().Changed("provider") {
		return flagString(cmd, "provider"), nil
	}
	if v := os.Getenv("POTACO_PROVIDER"); v != "" {
		return v, nil
	}
	p, _, err := mgr.GetActiveProvider()
	if err != nil {
		return "", configUserErr(
			"No active provider configured.",
			"Run 'potaco auth add <provider>' to connect one.",
			fmt.Errorf("no active provider: %w", err),
		)
	}
	if p == "" {
		return "", configUserErr(
			"No active provider configured.",
			"Run 'potaco auth add <provider>' to connect one.",
			fmt.Errorf("no active provider configured"),
		)
	}
	return p, nil
}

func resolveAPIKey(cmd *cobra.Command, mgr *auth.AuthManager, providerName string) (string, error) {
	if cmd.Flags().Changed("api-key") {
		return flagString(cmd, "api-key"), nil
	}
	if v := os.Getenv("POTACO_API_KEY"); v != "" {
		return v, nil
	}
	k, err := mgr.GetAPIKey(providerName)
	if err != nil {
		return "", configUserErr(
			fmt.Sprintf("No API key found for provider '%s'.", providerName),
			fmt.Sprintf("Run 'potaco auth add %s --api-key <key>' to set one.", providerName),
			fmt.Errorf("get API key: %w", err),
		)
	}
	return k, nil
}

func resolveModel(cmd *cobra.Command, cfg *config.MultiProviderConfig, providerName string) string {
	if cmd.Flags().Changed("model") {
		return flagString(cmd, "model")
	}
	if v := os.Getenv("POTACO_MODEL"); v != "" {
		return v
	}
	if cfg != nil {
		if pc, ok := cfg.Providers[providerName]; ok && pc.Model != "" {
			return pc.Model
		}
		return cfg.ActiveModel
	}
	return ""
}

func resolveBaseURL(cmd *cobra.Command, providerName string, cfg *config.MultiProviderConfig) string {
	if cmd.Flags().Changed("base-url") {
		return strings.TrimRight(flagString(cmd, "base-url"), "/")
	}
	if v := os.Getenv("POTACO_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	if cfg != nil {
		if pc, ok := cfg.Providers[providerName]; ok {
			if pc.BaseURL != "" {
				return strings.TrimRight(pc.BaseURL, "/")
			}
		}
	}
	if preset, ok := getProviderPreset(providerName); ok {
		return preset.BaseURL
	}
	return ""
}

func resolveRetriesTimeout(cmd *cobra.Command, cfg *config.MultiProviderConfig, providerName string) (int, time.Duration, error) {
	retries := 2
	timeout := 120 * time.Second

	// If config is corrupted, fall back to defaults rather than blocking generation.
	if cfg != nil {
		if pc, ok := cfg.Providers[providerName]; ok {
			if pc.Retries > 0 {
				retries = pc.Retries
			}
			if pc.Timeout > 0 {
				timeout = time.Duration(pc.Timeout) * time.Second
			}
		}
	}

	if v := os.Getenv("POTACO_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retries = n
		}
	}
	if v := os.Getenv("POTACO_TIMEOUT"); v != "" {
		if d, err := parseTimeoutString(v); err == nil {
			timeout = d
		}
	}
	if cmd.Flags().Changed("retries") {
		retries = flagInt(cmd, "retries")
	}
	if cmd.Flags().Changed("timeout") {
		timeoutStr, _ := cmd.Flags().GetString("timeout")
		d, err := parseTimeoutString(timeoutStr)
		if err != nil {
			return 0, 0, configUserErr(
				"Invalid timeout value.",
				"Use a number of seconds, e.g., --timeout 120.",
				err,
			)
		}
		timeout = d
	}
	return retries, timeout, nil
}
