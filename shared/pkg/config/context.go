package config

import "context"

// configKey is the context key for storing Config.
type configKey struct{}

// WithConfig stores config in context for request-scoped access.
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey{}, cfg)
}

// GetConfig retrieves config from context.
// Returns nil if config is not found in context.
func GetConfig(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configKey{}).(*Config); ok {
		return cfg
	}
	return nil
}
