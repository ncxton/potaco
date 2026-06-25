package fal

import (
	"testing"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestFalAdapterName(t *testing.T) {
	ad := New("test-key", adapter.AdapterOpts{})
	if ad.Name() != "fal" {
		t.Errorf("Name() = %q, want %q", ad.Name(), "fal")
	}
}

func TestFalAuthHeader(t *testing.T) {
	ad := New("my-key", adapter.AdapterOpts{})
	got := ad.AuthHeader("my-key")
	want := "Key my-key"
	if got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

func TestFalNewWithDefaults(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{})
	fa, ok := ad.(*Adapter)
	if !ok {
		t.Fatalf("expected *fal.Adapter, got %T", ad)
	}
	if fa.baseURL != "https://fal.run" {
		t.Errorf("baseURL = %q, want %q", fa.baseURL, "https://fal.run")
	}
	if fa.retries != 2 {
		t.Errorf("retries = %d, want 2", fa.retries)
	}
}

func TestFalNewWithOverrides(t *testing.T) {
	ad := New("key", adapter.AdapterOpts{
		BaseURL: "https://custom.fal.run",
		Retries: 5,
	})
	fa := ad.(*Adapter)
	if fa.baseURL != "https://custom.fal.run" {
		t.Errorf("baseURL = %q, want %q", fa.baseURL, "https://custom.fal.run")
	}
	if fa.retries != 5 {
		t.Errorf("retries = %d, want 5", fa.retries)
	}
}

func TestFalRegisteredInRegistry(t *testing.T) {
	_, err := adapter.Get("fal", "key", adapter.AdapterOpts{})
	if err != nil {
		t.Fatalf("adapter.Get(\"fal\") failed: %v", err)
	}
}
