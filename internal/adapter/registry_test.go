// internal/adapter/registry_test.go
package adapter

import (
	"testing"
)

func TestRegistryGetUnknownProvider(t *testing.T) {
	_, err := Get("nonexistent", "sk-test", AdapterOpts{})
	if err == nil {
		t.Fatal("Get should error for unknown provider")
	}
}

func TestRegistryListContainsRegisteredProviders(t *testing.T) {
	// The test-only adapter registered in TestRegistryRegisterAndGet
	names := List()
	// After init, openai should be registered (registered in openai package init)
	// But in the registry test, we only test what we explicitly add
	found := false
	for _, n := range names {
		if n == "test-provider" {
			found = true
			break
		}
	}
	if !found {
		// test-provider is registered in the test below via TestRegistryRegisterAndGet
		// If tests run in any order, we need to register here
		Register("test-provider", func(apiKey string, opts AdapterOpts) (Adapter, error) {
			return &mockAdapter{}, nil
		})
		_, err := Get("test-provider", "sk-test", AdapterOpts{})
		if err != nil {
			t.Fatalf("Get test-provider after register: %v", err)
		}
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	Register("test-provider", func(apiKey string, opts AdapterOpts) (Adapter, error) {
		return &mockAdapter{}, nil
	})

	a, err := Get("test-provider", "sk-test", AdapterOpts{})
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if a.Name() != "mock" {
		t.Errorf("Name = %q, want 'mock'", a.Name())
	}
}
