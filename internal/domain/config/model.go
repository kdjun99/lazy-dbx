// Package config defines domain models for lazy-dbx configuration.
package config

import (
	"fmt"
	"strings"
)

// Environment is a string-based tag for a connection's environment.
// Known constants are Production, Staging, Test; unknown values are stored as-is.
type Environment string

const (
	Production Environment = "production"
	Staging    Environment = "staging"
	Test       Environment = "test"
)

// ConnectionsConfig is the top-level parsed representation of connections.toml.
type ConnectionsConfig struct {
	Groups     map[string]*Group
	SSHTunnels map[string]SSHTunnelEntry
}

// Group is the first level of the connection hierarchy.
type Group struct {
	Subgroups map[string]*Subgroup
}

// Subgroup is the second level of the connection hierarchy (free-form names).
type Subgroup struct {
	Connections map[string]ConnectionEntry
}

// ConnectionEntry holds all configuration for a single database connection.
type ConnectionEntry struct {
	Name            string
	Type            string // "mysql" or "postgresql"
	Host            string
	Port            int
	User            string
	Database        string
	SSHTunnel       string // name of an SSHTunnelEntry, empty if none
	Env             Environment
	PasswordCmd     string
	PasswordEnv     string
	Password        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime string // duration string e.g. "5m"
	ConnectTimeout  string // duration string e.g. "10s"
}

// SSHTunnelEntry holds SSH tunnel configuration.
type SSHTunnelEntry struct {
	Host     string
	Port     int
	User     string
	Key      string // path to private key file
	UseAgent bool
}

// SettingsConfig holds all settings from settings.toml.
type SettingsConfig struct {
	Editor  EditorSettings
	Safety  SafetySettings
	Audit   AuditSettings
	UI      UISettings
	Logging LoggingSettings
}

// EditorSettings holds editor-related settings.
type EditorSettings struct {
	TabSize  int
	WordWrap bool
	Theme    string
}

// SafetySettings holds safety-related settings.
type SafetySettings struct {
	ConfirmDML        bool
	ConfirmDDL        bool
	ConfirmDrop       bool
	ReadonlyByDefault bool
}

// AuditSettings holds audit log settings.
type AuditSettings struct {
	Enabled bool
	LogPath string
}

// UISettings holds UI-related settings.
type UISettings struct {
	MouseEnabled   bool
	RefreshRate    string
	ResultPageSize int
	MaxResultRows  int
}

// LoggingSettings holds logging settings.
type LoggingSettings struct {
	LogPath string
	Level   string
}

// FindConnection looks up a connection by its dot-separated path: "group.subgroup.name".
func (c *ConnectionsConfig) FindConnection(path string) (*ConnectionEntry, error) {
	parts := strings.SplitN(path, ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("invalid connection path %q: expected group.subgroup.name", path)
	}

	groupName, subgroupName, connName := parts[0], parts[1], parts[2]

	group, ok := c.Groups[groupName]
	if !ok {
		return nil, fmt.Errorf("%w: group %q not found", ErrConnectionNotFound, groupName)
	}

	subgroup, ok := group.Subgroups[subgroupName]
	if !ok {
		return nil, fmt.Errorf("%w: subgroup %q not found in group %q", ErrConnectionNotFound, subgroupName, groupName)
	}

	entry, ok := subgroup.Connections[connName]
	if !ok {
		return nil, fmt.Errorf("%w: connection %q not found in %s.%s", ErrConnectionNotFound, connName, groupName, subgroupName)
	}

	return &entry, nil
}

// ConnectionsByEnv returns all connections tagged with the given environment.
func (c *ConnectionsConfig) ConnectionsByEnv(env Environment) []*ConnectionEntry {
	var result []*ConnectionEntry
	for _, group := range c.Groups {
		for _, subgroup := range group.Subgroups {
			for i := range subgroup.Connections {
				entry := subgroup.Connections[i]
				if entry.Env == env {
					result = append(result, &entry)
				}
			}
		}
	}
	return result
}

// ValidateReferences checks that all SSHTunnel references point to defined tunnels.
func (c *ConnectionsConfig) ValidateReferences() error {
	for groupName, group := range c.Groups {
		for subgroupName, subgroup := range group.Subgroups {
			for connName, entry := range subgroup.Connections {
				if entry.SSHTunnel == "" {
					continue
				}
				if _, ok := c.SSHTunnels[entry.SSHTunnel]; !ok {
					return fmt.Errorf("%w: connection %s.%s.%s references undefined tunnel %q",
						ErrDanglingTunnelRef, groupName, subgroupName, connName, entry.SSHTunnel)
				}
			}
		}
	}
	return nil
}
