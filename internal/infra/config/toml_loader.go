package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
)

// TOMLLoader implements domain/config.ConfigLoader using go-toml/v2.
type TOMLLoader struct{}

// NewTOMLLoader creates a new TOMLLoader.
func NewTOMLLoader() *TOMLLoader {
	return &TOMLLoader{}
}

// rawConnections is the intermediate struct for TOML unmarshalling.
type rawConnections struct {
	SSHTunnels map[string]rawTunnel                           `toml:"ssh_tunnels"`
	Groups     map[string]map[string]map[string]rawConnection `toml:"groups"`
}

type rawTunnel struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Key      string `toml:"key"`
	UseAgent bool   `toml:"use_agent"`
}

type rawConnection struct {
	Type            string `toml:"type"`
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	User            string `toml:"user"`
	Database        string `toml:"database"`
	SSHTunnel       string `toml:"ssh_tunnel"`
	Env             string `toml:"env"`
	PasswordCmd     string `toml:"password_cmd"`
	PasswordEnv     string `toml:"password_env"`
	Password        string `toml:"password"`
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime string `toml:"conn_max_lifetime"`
	ConnectTimeout  string `toml:"connect_timeout"`
}

// rawSettings is the intermediate struct for settings.toml unmarshalling.
type rawSettings struct {
	Editor  rawEditorSettings  `toml:"editor"`
	Safety  rawSafetySettings  `toml:"safety"`
	Audit   rawAuditSettings   `toml:"audit"`
	UI      rawUISettings      `toml:"ui"`
	Logging rawLoggingSettings `toml:"logging"`
}

type rawEditorSettings struct {
	TabSize  int    `toml:"tab_size"`
	WordWrap bool   `toml:"word_wrap"`
	Theme    string `toml:"theme"`
}

type rawSafetySettings struct {
	ConfirmDML        bool `toml:"confirm_dml"`
	ConfirmDDL        bool `toml:"confirm_ddl"`
	ConfirmDrop       bool `toml:"confirm_drop"`
	ReadonlyByDefault bool `toml:"readonly_by_default"`
}

type rawAuditSettings struct {
	Enabled bool   `toml:"enabled"`
	LogPath string `toml:"log_path"`
}

type rawUISettings struct {
	MouseEnabled bool   `toml:"mouse_enabled"`
	RefreshRate  string `toml:"refresh_rate"`
}

type rawLoggingSettings struct {
	LogPath string `toml:"log_path"`
	Level   string `toml:"level"`
}

// LoadConnections parses a connections.toml file.
func (l *TOMLLoader) LoadConnections(path string) (domain.Result[*domainconfig.ConnectionsConfig], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Result[*domainconfig.ConnectionsConfig]{
				Error: fmt.Errorf("%w: %s", domainconfig.ErrConfigNotFound, path),
			}, nil
		}
		return domain.Result[*domainconfig.ConnectionsConfig]{
			Error: fmt.Errorf("%w: %s", domainconfig.ErrConfigNotFound, path),
		}, nil
	}

	var raw rawConnections
	if err := toml.Unmarshal(data, &raw); err != nil {
		return domain.Result[*domainconfig.ConnectionsConfig]{
			Error: fmt.Errorf("%w: %s", domainconfig.ErrConfigParse, err.Error()),
		}, nil
	}

	cfg, convertErr := l.convertConnections(&raw)
	if convertErr != nil {
		return domain.Result[*domainconfig.ConnectionsConfig]{Error: convertErr}, nil //nolint:nilerr
	}

	if validateErr := cfg.ValidateReferences(); validateErr != nil {
		return domain.Result[*domainconfig.ConnectionsConfig]{Error: validateErr}, nil //nolint:nilerr
	}

	return domain.Result[*domainconfig.ConnectionsConfig]{Data: cfg}, nil
}

// LoadSettings parses a settings.toml file.
func (l *TOMLLoader) LoadSettings(path string) (domain.Result[*domainconfig.SettingsConfig], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Result[*domainconfig.SettingsConfig]{
				Error: fmt.Errorf("%w: %s", domainconfig.ErrConfigNotFound, path),
			}, nil
		}
		return domain.Result[*domainconfig.SettingsConfig]{
			Error: fmt.Errorf("%w: %s", domainconfig.ErrConfigNotFound, path),
		}, nil
	}

	var raw rawSettings
	if err := toml.Unmarshal(data, &raw); err != nil {
		return domain.Result[*domainconfig.SettingsConfig]{
			Error: fmt.Errorf("%w: %s", domainconfig.ErrConfigParse, err.Error()),
		}, nil
	}

	cfg := convertSettings(&raw)
	return domain.Result[*domainconfig.SettingsConfig]{Data: cfg}, nil
}

func (l *TOMLLoader) convertConnections(raw *rawConnections) (*domainconfig.ConnectionsConfig, error) {
	cfg := &domainconfig.ConnectionsConfig{
		Groups:     make(map[string]*domainconfig.Group),
		SSHTunnels: make(map[string]domainconfig.SSHTunnelEntry),
	}

	// Convert SSH tunnels
	for name, rt := range raw.SSHTunnels {
		port := rt.Port
		if port == 0 {
			port = 22
		}
		keyPath, err := ExpandPath(rt.Key)
		if err != nil {
			return nil, fmt.Errorf("expanding tunnel key path: %w", err)
		}
		cfg.SSHTunnels[name] = domainconfig.SSHTunnelEntry{
			Host:     rt.Host,
			Port:     port,
			User:     rt.User,
			Key:      keyPath,
			UseAgent: rt.UseAgent,
		}
	}

	// Convert groups (map[groupName]map[subgroupName]map[connName]rawConnection)
	for groupName, subgroups := range raw.Groups {
		group := &domainconfig.Group{
			Subgroups: make(map[string]*domainconfig.Subgroup),
		}
		for subgroupName, connections := range subgroups {
			subgroup := &domainconfig.Subgroup{
				Connections: make(map[string]domainconfig.ConnectionEntry),
			}
			for connName, rc := range connections {
				entry, err := l.convertConnection(connName, &rc)
				if err != nil {
					return nil, fmt.Errorf("connection %s.%s.%s: %w", groupName, subgroupName, connName, err)
				}
				subgroup.Connections[connName] = *entry
			}
			group.Subgroups[subgroupName] = subgroup
		}
		cfg.Groups[groupName] = group
	}

	return cfg, nil
}

func (l *TOMLLoader) convertConnection(name string, rc *rawConnection) (*domainconfig.ConnectionEntry, error) {
	if rc.Host == "" {
		return nil, fmt.Errorf("missing required field: host")
	}

	dbType := rc.Type
	if dbType == "" {
		return nil, fmt.Errorf("missing required field: type")
	}
	if dbType != "mysql" && dbType != "postgresql" {
		return nil, fmt.Errorf("unknown database type %q: must be \"mysql\" or \"postgresql\"", dbType)
	}

	port := rc.Port
	if port == 0 {
		switch dbType {
		case "mysql":
			port = 3306
		case "postgresql":
			port = 5432
		}
	}

	maxOpen := rc.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 5
	}
	maxIdle := rc.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 2
	}
	lifetime := rc.ConnMaxLifetime
	if lifetime == "" {
		lifetime = "5m"
	}
	timeout := rc.ConnectTimeout
	if timeout == "" {
		timeout = "10s"
	}

	return &domainconfig.ConnectionEntry{
		Name:            name,
		Type:            dbType,
		Host:            rc.Host,
		Port:            port,
		User:            rc.User,
		Database:        rc.Database,
		SSHTunnel:       rc.SSHTunnel,
		Env:             domainconfig.Environment(rc.Env),
		PasswordCmd:     rc.PasswordCmd,
		PasswordEnv:     rc.PasswordEnv,
		Password:        rc.Password,
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: lifetime,
		ConnectTimeout:  timeout,
	}, nil
}

func convertSettings(raw *rawSettings) *domainconfig.SettingsConfig {
	return &domainconfig.SettingsConfig{
		Editor: domainconfig.EditorSettings{
			TabSize:  raw.Editor.TabSize,
			WordWrap: raw.Editor.WordWrap,
			Theme:    raw.Editor.Theme,
		},
		Safety: domainconfig.SafetySettings{
			ConfirmDML:        raw.Safety.ConfirmDML,
			ConfirmDDL:        raw.Safety.ConfirmDDL,
			ConfirmDrop:       raw.Safety.ConfirmDrop,
			ReadonlyByDefault: raw.Safety.ReadonlyByDefault,
		},
		Audit: domainconfig.AuditSettings{
			Enabled: raw.Audit.Enabled,
			LogPath: raw.Audit.LogPath,
		},
		UI: domainconfig.UISettings{
			MouseEnabled: raw.UI.MouseEnabled,
			RefreshRate:  raw.UI.RefreshRate,
		},
		Logging: domainconfig.LoggingSettings{
			LogPath: raw.Logging.LogPath,
			Level:   raw.Logging.Level,
		},
	}
}
