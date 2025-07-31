package bookmarks

import (
	"fmt"
	"sync"
)

// Registry manages all bookmark providers
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	configs   map[string]ProviderConfig
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		configs:   make(map[string]ProviderConfig),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	return provider, exists
}

// GetEnabled returns all enabled providers
func (r *Registry) GetEnabled() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var enabled []Provider
	for _, provider := range r.providers {
		if provider.IsEnabled() {
			enabled = append(enabled, provider)
		}
	}
	return enabled
}

// List returns all registered provider names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Configure applies configuration to a provider
func (r *Registry) Configure(name string, config ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	if err := provider.Configure(config.Settings); err != nil {
		return fmt.Errorf("failed to configure provider %s: %v", name, err)
	}

	r.configs[name] = config
	return nil
}
