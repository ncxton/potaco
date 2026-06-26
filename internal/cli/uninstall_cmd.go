package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the potaco binary and optionally its configuration",
	RunE:  runUninstall,
}

// findPotacoBinaryFn locates the running potaco binary path, resolving
// symlinks. It is a package-level variable so tests can override it with
// a fake path instead of resolving the test binary itself.
var findPotacoBinaryFn = findPotacoBinary

func init() {
	uninstallCmd.Flags().Bool("remove-config", false, "also remove ~/.potaco/ config directory")
	uninstallCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompts")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	removeConfig, _ := cmd.Flags().GetBool("remove-config")
	nonInteractive := !tui.IsInteractive()

	// Locate the binary
	binaryPath, findErr := findPotacoBinaryFn()
	if findErr != nil && nonInteractive {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: potaco binary not found: %v\n", findErr)
	}

	// Determine home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return configUserErr(
			"Could not determine your home directory.",
			"",
			fmt.Errorf("user home dir: %w", err),
		)
	}
	configDir := filepath.Join(home, ".potaco")

	if nonInteractive {
		return runUninstallNonInteractive(cmd, binaryPath, configDir, removeConfig)
	}

	// Interactive mode: use the TUI flow
	return runUninstallInteractive(cmd, binaryPath, configDir, yes, removeConfig)
}

func runUninstallNonInteractive(cmd *cobra.Command, binaryPath, configDir string, removeConfig bool) error {
	out := cmd.OutOrStdout()

	if binaryPath != "" {
		if err := os.Remove(binaryPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(out, "Binary not found at %s (may have been removed already).\n", binaryPath)
			} else {
				return configUserErr(
					"Cannot remove the binary.",
					fmt.Sprintf("Check file permissions or remove manually: %s", binaryPath),
					fmt.Errorf("remove binary: %w", err),
				)
			}
		} else {
			fmt.Fprintf(out, "Binary removed: %s\n", binaryPath)
		}
	}

	if removeConfig {
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Fprintf(out, "Warning: could not remove config directory: %v\n", err)
		} else {
			fmt.Fprintf(out, "Config directory removed: %s\n", configDir)
		}
	}

	fmt.Fprintf(out, "Uninstall complete.\n")
	return nil
}

func runUninstallInteractive(cmd *cobra.Command, binaryPath, configDir string, yes, removeConfigFlag bool) error {
	out := cmd.OutOrStdout()

	// Step 1: Confirm binary removal
	if !yes {
		confirmed, err := tui.ConfirmAction(fmt.Sprintf("This will remove the potaco binary at '%s'. Continue?", binaryPath))
		if err != nil {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
		if !confirmed {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Step 2: Ask about config removal
	removeConfig := false
	if !yes {
		removeConfig, _ = tui.ConfirmAction(fmt.Sprintf("Also remove configuration and credentials at '%s'?", configDir))
	} else {
		// --yes auto-answers; use --remove-config flag value.
		removeConfig = removeConfigFlag
	}

	// Step 3: Final confirmation
	if !yes {
		summary := "Confirm: remove binary"
		if removeConfig {
			summary += " and config directory"
		}
		summary += "?"
		finalConfirm, err := tui.ConfirmAction(summary)
		if err != nil {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
		if !finalConfirm {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Execute removal
	if binaryPath != "" {
		if err := os.Remove(binaryPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(out, "Binary not found at %s (may have been removed already).\n", binaryPath)
			} else {
				return configUserErr(
					"Cannot remove the binary.",
					fmt.Sprintf("Check file permissions or remove manually: %s", binaryPath),
					fmt.Errorf("remove binary: %w", err),
				)
			}
		} else {
			fmt.Fprintf(out, "Binary removed: %s\n", binaryPath)
		}
	}

	if removeConfig {
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Fprintf(out, "Warning: could not remove config directory: %v\n", err)
		} else {
			fmt.Fprintf(out, "Config directory removed: %s\n", configDir)
		}
	}

	fmt.Fprintf(out, "Uninstall complete.\n")
	return nil
}

// findPotacoBinary locates the running potaco binary path, resolving
// symlinks. Falls back to searching PATH if os.Executable() fails.
func findPotacoBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		path, lookupErr := exec.LookPath("potaco")
		if lookupErr != nil {
			return "", fmt.Errorf("could not locate potaco binary: %w", lookupErr)
		}
		return path, nil
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}
