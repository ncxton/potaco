package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update potaco to the latest release",
	Aliases: []string{"upgrade"},
	RunE:    runUpdate,
}

func init() {
	updateCmd.Flags().BoolP("force", "f", false, "force update even if already at latest version")
	rootCmd.AddCommand(updateCmd)
}

// installScriptURL returns the raw install.sh URL for a given release tag.
func installScriptURL(tag string) string {
	return fmt.Sprintf("https://github.com/ncxton/potaco/releases/download/%s/install.sh", tag)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// Check latest version (reuses cache from version command).
	latest, err := checkLatestVersion()
	if err != nil {
		return configUserErr(
			"Could not check for updates.",
			"Check your network connection and try again.",
			fmt.Errorf("check latest version: %w", err),
		)
	}

	// Compare versions unless --force or unknown local build.
	if !force && Version != "unknown" && Version == latest {
		fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (%s).\n", Version)
		return nil
	}

	if force && Version != "unknown" && Version == latest {
		fmt.Fprintf(cmd.OutOrStdout(), "Forcing update (already at %s)...\n", Version)
	}

	// Download install.sh from the release.
	installURL := installScriptURL(latest)
	tmpFile, err := os.CreateTemp("", "potaco-install-*.sh")
	if err != nil {
		return apiUserErr(
			"Could not create a temporary file for the installer.",
			"",
			fmt.Errorf("create temp file: %w", err),
		)
	}
	defer os.Remove(tmpFile.Name())

	resp, err := http.Get(installURL)
	if err != nil {
		return apiUserErr(
			"Could not download the installer.",
			"Check your network connection and try again.",
			fmt.Errorf("download install.sh: %w", err),
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apiUserErr(
			"Could not download the installer.",
			fmt.Sprintf("GitHub returned status %d for the install script.", resp.StatusCode),
			fmt.Errorf("install.sh download returned status %d", resp.StatusCode),
		)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return apiUserErr(
			"Could not save the installer to disk.",
			"",
			fmt.Errorf("write temp file: %w", err),
		)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return configUserErr(
			"Could not make the installer executable.",
			"",
			fmt.Errorf("chmod installer: %w", err),
		)
	}

	// Execute the installer, inheriting stdin/stdout/stderr.
	fmt.Fprintf(cmd.OutOrStdout(), "Running installer...\n")

	sc := exec.Command("sh", tmpFile.Name())
	sc.Stdin = os.Stdin
	sc.Stdout = os.Stdout
	sc.Stderr = os.Stderr

	// Pass non-interactive flag to the installer if set.
	if !tui.IsInteractive() {
		sc.Env = append(os.Environ(), "POTACO_NON_INTERACTIVE=1")
	} else {
		sc.Env = os.Environ()
	}

	if err := sc.Run(); err != nil {
		return apiUserErr(
			"Installer failed.",
			"Check the output above for details, or try running the installer manually.",
			fmt.Errorf("install.sh execution: %w", err),
		)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Update complete.\n")
	return nil
}
