package bookmarks

import (
	"context"
	"time"
)

// Bookmark represents a single bookmark entry
type Bookmark struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Source    string    `json:"source"` // Which provider this came from
	Timestamp time.Time `json:"timestamp"`
}

// Provider defines the interface for bookmark providers
type Provider interface {
	// Name returns the unique name of this provider
	Name() string

	// GetBookmarks retrieves bookmarks from this provider
	GetBookmarks(ctx context.Context) ([]Bookmark, error)

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// Configure allows the provider to be configured with settings
	Configure(config map[string]interface{}) error
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
}
