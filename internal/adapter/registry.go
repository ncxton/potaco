// internal/adapter/registry.go
package adapter

import (
	"fmt"
	"sort"
	"sync"
)

// AdapterOpts holds options for constructing an adapter instance.
type AdapterOpts struct {
	BaseURL string // override the adapter's default base URL
	Timeout string // override the adapter's default timeout
}

// AdapterFactory creates an Adapter instance for a given API key and options.
type AdapterFactory func(apiKey string, opts AdapterOpts) (Adapter, error)

var (
	registry   = make(map[string]AdapterFactory)
	registryMu sync.RWMutex
)

// Register adds a provider adapter factory to the registry.
// Called by each adapter package's init() function.
func Register(name string, factory AdapterFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Get retrieves and instantiates the adapter for the named provider.
func Get(name string, apiKey string, opts AdapterOpts) (Adapter, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %v)", name, List())
	}
	return factory(apiKey, opts)
}

// List returns the names of all registered providers, sorted alphabetically.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
