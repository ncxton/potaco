package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/tui"
)

var (
	autoUpdateCacheTTL      = 24 * time.Hour
	autoUpdateNow           = time.Now
	autoUpdateCheckLatest   = checkLatestVersion
	autoUpdateInstall       = installUpdate
	autoUpdatePrompt        = promptAutoUpdate
	autoUpdateIsInteractive = tui.IsInteractive
)

var rootCmd = &cobra.Command{
	Use:           "potaco",
	Short:         "Terminal image generation and editing CLI",
	Long:          `Potaco provides advanced image generation and editing inside the terminal.`,
	Version:       Version,
	Run:           func(cmd *cobra.Command, args []string) {},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		renderAnyError(os.Stderr, err)
		var ue *UserError
		if errors.As(err, &ue) {
			os.Exit(ue.ExitCode())
		}
		if ec, ok := err.(*ExitCoder); ok {
			os.Exit(ec.Code)
		}
		os.Exit(1)
	}
}

func runConfigMigrations() error {
	_, _, err := config.MigrateConfigFile(config.DefaultConfigPath(), time.Now)
	if err != nil {
		return configUserErr(
			"Could not migrate your config file.",
			"A backup is created before migrations. Check ~/.potaco/config.yaml and retry.",
			fmt.Errorf("migrate config: %w", err),
		)
	}
	return nil
}

func runAutoUpdateCheck(cmd *cobra.Command) error {
	if shouldSkipAutoUpdate(cmd) {
		return nil
	}

	cfg, err := config.LoadMultiProvider(config.DefaultConfigPath())
	if err != nil {
		verbosef(cmd, "auto-update: read config: %v\n", err)
		return nil
	}
	if !cfg.AutoUpdateEnabled() {
		return nil
	}

	cache, err := config.LoadUpdateCache(config.DefaultUpdateCachePath())
	if err != nil {
		verbosef(cmd, "auto-update: read cache: %v\n", err)
		cache = &config.UpdateCache{}
	}

	now := autoUpdateNow()
	latest := cache.LatestVersion
	if cache.LastUpdateCheck.IsZero() || now.Sub(cache.LastUpdateCheck) >= autoUpdateCacheTTL {
		latest, err = autoUpdateCheckLatest()
		if err != nil {
			verbosef(cmd, "auto-update: check latest: %v\n", err)
			return nil
		}
		cache.LastUpdateCheck = now
		cache.LatestVersion = latest
		if err := config.SaveUpdateCache(config.DefaultUpdateCachePath(), cache); err != nil {
			verbosef(cmd, "auto-update: write cache: %v\n", err)
		}
	}

	if latest == "" || !versionNewer(latest, Version) || cache.DismissedVersion == latest {
		return nil
	}

	yes, err := autoUpdatePrompt(fmt.Sprintf("Potaco %s is available. Update now? [Y/n] ", latest))
	if err != nil {
		verbosef(cmd, "auto-update: prompt: %v\n", err)
		return nil
	}
	if !yes {
		cache.DismissedVersion = latest
		if err := config.SaveUpdateCache(config.DefaultUpdateCachePath(), cache); err != nil {
			verbosef(cmd, "auto-update: write dismissal: %v\n", err)
		}
		return nil
	}

	return autoUpdateInstall(cmd, latest)
}

func shouldSkipAutoUpdate(cmd *cobra.Command) bool {
	if Version == "unknown" {
		return true
	}
	if !autoUpdateIsInteractive() {
		return true
	}
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonMode {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "update", "version", "uninstall", "help", "completion":
			return true
		}
	}
	if cmd.Name() == "set" && cmd.Parent() != nil && cmd.Parent().Name() == "config" {
		args := cmd.Flags().Args()
		if len(args) > 0 && args[0] == "auto_update" {
			return true
		}
	}
	return false
}

func promptAutoUpdate(prompt string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "" || answer == "y" || answer == "yes", nil
}

func versionNewer(latest, current string) bool {
	l, lok := parseVersion(latest)
	c, cok := parseVersion(current)
	if !lok || !cok {
		return latest != "" && latest != current
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, part := range parts {
		if part == "" {
			return out, false
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func verbosef(cmd *cobra.Command, format string, args ...any) {
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.DisplayName}} {{.Version}}\n")

	rootCmd.PersistentFlags().Bool("json", false, "output JSON metadata to stdout")
	rootCmd.PersistentFlags().Bool("verbose", false, "print retry attempts and debug info to stderr")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "force non-interactive mode (skip TUI)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ni, err := cmd.Flags().GetBool("non-interactive")
		if err != nil {
			return fmt.Errorf("read non-interactive flag: %w", err)
		}
		tui.SetNonInteractive(ni)
		if err := runConfigMigrations(); err != nil {
			return err
		}
		return runAutoUpdateCheck(cmd)
	}
}
