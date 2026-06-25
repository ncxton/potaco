package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
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
//	--model flag    > POTACO_MODEL env   > active_model from config
//	--base-url flag > POTACO_BASE_URL env > provider preset default
func resolveAdapterForCommand(cmd *cobra.Command) (*resolvedConfig, error) {
	mgr, err := auth.New()
	if err != nil {
		return nil, configError(fmt.Errorf("init auth: %w", err))
	}

	providerName, err := resolveProvider(cmd, mgr)
	if err != nil {
		return nil, err
	}

	apiKey, err := resolveAPIKey(cmd, mgr)
	if err != nil {
		return nil, err
	}

	model := resolveModel(cmd, mgr)
	baseURL := resolveBaseURL(cmd)

	// Fall back to the provider preset's BaseURL for dry-run output when
	// no explicit override is given. The adapter has its own default, but
	// dry-run constructs the URL from BaseURL directly.
	if baseURL == "" {
		if preset, ok := getProviderPreset(providerName); ok {
			baseURL = preset.BaseURL
		}
	}

	retries, timeout := resolveRetriesTimeout(cmd, mgr, providerName)

	opts := adapter.AdapterOpts{
		BaseURL: baseURL,
		Retries: retries,
	}
	if timeout > 0 {
		opts.Timeout = timeout.String()
	}

	ad, err := adapter.Get(providerName, apiKey, opts)
	if err != nil {
		return nil, configError(fmt.Errorf("create adapter: %w", err))
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
		return "", configError(fmt.Errorf("no active provider: %w", err))
	}
	if p == "" {
		return "", configError(fmt.Errorf("no active provider configured. Use 'potaco auth add <provider>' to connect one"))
	}
	return p, nil
}

func resolveAPIKey(cmd *cobra.Command, mgr *auth.AuthManager) (string, error) {
	if cmd.Flags().Changed("api-key") {
		return flagString(cmd, "api-key"), nil
	}
	if v := os.Getenv("POTACO_API_KEY"); v != "" {
		return v, nil
	}
	k, err := mgr.GetActiveAPIKey()
	if err != nil {
		return "", configError(fmt.Errorf("get API key: %w", err))
	}
	return k, nil
}

func resolveModel(cmd *cobra.Command, mgr *auth.AuthManager) string {
	if cmd.Flags().Changed("model") {
		return flagString(cmd, "model")
	}
	if v := os.Getenv("POTACO_MODEL"); v != "" {
		return v
	}
	// GetActiveProvider error is already handled by resolveProvider above;
	// if we reach here, the provider is valid and the error is unreachable.
	_, m, _ := mgr.GetActiveProvider()
	return m
}

func resolveBaseURL(cmd *cobra.Command) string {
	if cmd.Flags().Changed("base-url") {
		return flagString(cmd, "base-url")
	}
	if v := os.Getenv("POTACO_BASE_URL"); v != "" {
		return v
	}
	return ""
}

func resolveRetriesTimeout(cmd *cobra.Command, mgr *auth.AuthManager, providerName string) (int, time.Duration) {
	retries := 2
	timeout := 120 * time.Second

	cfg, _ := mgr.LoadConfig()
	if pc, ok := cfg.Providers[providerName]; ok {
		if pc.Retries > 0 {
			retries = pc.Retries
		}
		if pc.Timeout > 0 {
			timeout = pc.Timeout
		}
	}

	if v := os.Getenv("POTACO_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retries = n
		}
	}
	if v := os.Getenv("POTACO_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	if cmd.Flags().Changed("retries") {
		retries = flagInt(cmd, "retries")
	}
	if cmd.Flags().Changed("timeout") {
		timeout, _ = cmd.Flags().GetDuration("timeout")
	}
	return retries, timeout
}
