package config

import "github.com/kdjun99/lazy-dbx/internal/domain"

// Loader defines how configuration files are loaded.
// Implementations live in internal/infra/config/.
type Loader interface {
	// LoadConnections parses a connections.toml file at the given path.
	LoadConnections(path string) (domain.Result[*ConnectionsConfig], error)
	// LoadSettings parses a settings.toml file at the given path.
	LoadSettings(path string) (domain.Result[*SettingsConfig], error)
}
