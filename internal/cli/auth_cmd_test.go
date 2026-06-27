package cli

import (
	"strings"
	"testing"
)

func TestAuthCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "auth" || strings.HasPrefix(cmd.Use, "auth ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'auth' subcommand")
	}
}

func TestAuthAddCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "add" || strings.HasPrefix(cmd.Use, "add ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'add' subcommand")
	}
}

func TestAuthRemoveCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "remove" || strings.HasPrefix(cmd.Use, "remove ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'remove' subcommand")
	}
}

func TestAuthListCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "list" || strings.HasPrefix(cmd.Use, "list ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'list' subcommand")
	}
}
