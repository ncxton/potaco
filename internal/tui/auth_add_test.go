package tui

import (
	"testing"
)

func TestRunAuthAddUnknownProvider(t *testing.T) {
	err := RunAuthAdd("nonexistent-provider")
	if err == nil {
		t.Fatal("RunAuthAdd with unknown provider should return an error")
	}
}

func TestRunAuthAddEmptyProvider(t *testing.T) {
	err := RunAuthAdd("")
	if err == nil {
		t.Fatal("RunAuthAdd with empty provider should return an error")
	}
}
