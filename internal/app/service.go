// Package app provides the application service layer that orchestrates
// config loading, password resolution, SSH tunneling, and database connection.
package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
	domaintunnel "github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

// Compile-time check: ConnectionService must satisfy connection.Manager.
var _ domainconn.Manager = (*ConnectionService)(nil)

// ValidationReport summarises the result of a config validation run.
type ValidationReport struct {
	Valid     bool
	Errors    []string
	ConnCount int
}

// Config holds all dependencies for ConnectionService.
type Config struct {
	// ConfigDir is the directory containing connections.toml and settings.toml.
	ConfigDir string
	// Loader parses config files.
	Loader domainconfig.Loader
	// PasswordResolver resolves database passwords.
	PasswordResolver domainpw.Resolver
	// TunnelManager manages SSH tunnels.
	TunnelManager domaintunnel.Manager
	// MySQLConnector connects to MySQL databases.
	MySQLConnector domainconn.Connector
	// PGConnector connects to PostgreSQL databases.
	PGConnector domainconn.Connector
	// Pool manages active database connections.
	Pool domainconn.Pool
	// Logger is the structured logger.
	Logger domainlogger.Logger
}

// ConnectionService orchestrates the full connection lifecycle:
// config → password → tunnel → connect → pool.
type ConnectionService struct {
	cfg Config
}

// NewConnectionService creates a ConnectionService with injected dependencies.
func NewConnectionService(cfg Config) *ConnectionService {
	return &ConnectionService{cfg: cfg}
}

// Connect performs: load config → find entry → resolve password →
// open tunnel (if needed) → connect DB → store in pool.
func (s *ConnectionService) Connect(ctx context.Context, path string) domain.Result[domainconn.Info] {
	s.cfg.Logger.Info(ctx, "ConnectionService", "Connect", "starting", domainlogger.F("path", path))

	connCfg, err := s.loadConnections(ctx)
	if err != nil {
		return domain.Result[domainconn.Info]{Error: err}
	}

	entry, err := connCfg.FindConnection(path)
	if err != nil {
		return domain.Result[domainconn.Info]{Error: fmt.Errorf("find connection: %w", err)}
	}

	password, err := s.cfg.PasswordResolver.Resolve(ctx, domainpw.Config{
		ConnectionName: entry.Name,
		Cmd:            entry.PasswordCmd,
		Env:            entry.PasswordEnv,
		Plaintext:      entry.Password,
	})
	if err != nil {
		return domain.Result[domainconn.Info]{Error: fmt.Errorf("resolve password: %w", err)}
	}

	var tunnelState *domaintunnel.State
	if entry.SSHTunnel != "" {
		tunnelEntry, ok := connCfg.SSHTunnels[entry.SSHTunnel]
		if !ok {
			return domain.Result[domainconn.Info]{
				Error: fmt.Errorf("connection %q references tunnel %q, but it is not defined in [ssh_tunnels]", path, entry.SSHTunnel),
			}
		}
		tunnelState, err = s.cfg.TunnelManager.GetOrOpen(ctx, entry.SSHTunnel, domaintunnel.Config{
			SSHHost:  tunnelEntry.Host,
			SSHPort:  tunnelEntry.Port,
			SSHUser:  tunnelEntry.User,
			KeyPath:  tunnelEntry.Key,
			UseAgent: tunnelEntry.UseAgent,
		}, entry.Host, entry.Port)
		if err != nil {
			return domain.Result[domainconn.Info]{Error: fmt.Errorf("open tunnel: %w", err)}
		}
	}

	connector := s.connectorFor(entry.Type)
	if connector == nil {
		return domain.Result[domainconn.Info]{
			Error: fmt.Errorf("%w: %q (supported: mysql, postgresql)", domainconn.ErrUnsupportedDriver, entry.Type),
		}
	}

	db, err := connector.Connect(ctx, *entry, password, tunnelState)
	if err != nil {
		if entry.SSHTunnel != "" {
			if releaseErr := s.cfg.TunnelManager.Release(entry.SSHTunnel); releaseErr != nil {
				s.cfg.Logger.Warn(ctx, "ConnectionService", "Connect", "tunnel release after connect error",
					domainlogger.F("error", releaseErr.Error()))
			}
		}
		return domain.Result[domainconn.Info]{Error: fmt.Errorf("connect: %w", err)}
	}

	if err := s.cfg.Pool.Put(path, db); err != nil {
		db.Close()
		return domain.Result[domainconn.Info]{Error: fmt.Errorf("pool put: %w", err)}
	}

	info := domainconn.Info{
		Path:     path,
		Group:    groupFromPath(path),
		Subgroup: subgroupFromPath(path),
		Name:     entry.Name,
		Type:     entry.Type,
		Host:     entry.Host,
		Env:      string(entry.Env),
		Status:   "connected",
	}

	s.cfg.Logger.Info(ctx, "ConnectionService", "Connect", "connected", domainlogger.F("path", path))
	return domain.Result[domainconn.Info]{Data: info}
}

// Disconnect closes the pool entry and releases the tunnel.
func (s *ConnectionService) Disconnect(ctx context.Context, path string) domain.Result[struct{}] {
	s.cfg.Logger.Info(ctx, "ConnectionService", "Disconnect", "starting", domainlogger.F("path", path))

	connCfg, err := s.loadConnections(ctx)
	if err != nil {
		// Best-effort: still close the pool entry
		if poolErr := s.cfg.Pool.Close(path); poolErr != nil {
			s.cfg.Logger.Warn(ctx, "ConnectionService", "Disconnect", "pool close during error recovery",
				domainlogger.F("path", path), domainlogger.F("error", poolErr.Error()))
		}
		return domain.Result[struct{}]{Error: err}
	}

	entry, err := connCfg.FindConnection(path)
	if err != nil {
		if poolErr := s.cfg.Pool.Close(path); poolErr != nil {
			s.cfg.Logger.Warn(ctx, "ConnectionService", "Disconnect", "pool close during error recovery",
				domainlogger.F("path", path), domainlogger.F("error", poolErr.Error()))
		}
		return domain.Result[struct{}]{Error: fmt.Errorf("find connection: %w", err)}
	}

	if poolErr := s.cfg.Pool.Close(path); poolErr != nil {
		return domain.Result[struct{}]{Error: fmt.Errorf("pool close: %w", poolErr)}
	}

	if entry.SSHTunnel != "" {
		if releaseErr := s.cfg.TunnelManager.Release(entry.SSHTunnel); releaseErr != nil {
			s.cfg.Logger.Warn(ctx, "ConnectionService", "Disconnect", "tunnel release error",
				domainlogger.F("path", path), domainlogger.F("error", releaseErr.Error()))
		}
	}

	s.cfg.Logger.Info(ctx, "ConnectionService", "Disconnect", "disconnected", domainlogger.F("path", path))
	return domain.Result[struct{}]{Data: struct{}{}}
}

// Ping checks that the pooled connection is alive.
func (s *ConnectionService) Ping(ctx context.Context, path string) domain.Result[domainconn.PingResult] {
	s.cfg.Logger.Info(ctx, "ConnectionService", "Ping", "starting", domainlogger.F("path", path))

	if err := s.cfg.Pool.Ping(ctx, path); err != nil {
		return domain.Result[domainconn.PingResult]{Error: fmt.Errorf("ping: %w", err)}
	}

	return domain.Result[domainconn.PingResult]{
		Data: domainconn.PingResult{Path: path},
	}
}

// ListConnections returns a flat list of all configured connections.
func (s *ConnectionService) ListConnections(ctx context.Context) domain.Result[[]domainconn.Info] {
	connCfg, err := s.loadConnections(ctx)
	if err != nil {
		return domain.Result[[]domainconn.Info]{Error: err}
	}

	var infos []domainconn.Info
	for groupName, group := range connCfg.Groups {
		for subgroupName, subgroup := range group.Subgroups {
			for connName, entry := range subgroup.Connections {
				path := groupName + "." + subgroupName + "." + connName
				infos = append(infos, domainconn.Info{
					Path:     path,
					Group:    groupName,
					Subgroup: subgroupName,
					Name:     entry.Name,
					Type:     entry.Type,
					Host:     entry.Host,
					Env:      string(entry.Env),
					Status:   "configured",
				})
			}
		}
	}

	return domain.Result[[]domainconn.Info]{Data: infos}
}

// ValidateConfig parses and validates config without connecting.
func (s *ConnectionService) ValidateConfig(ctx context.Context) domain.Result[ValidationReport] {
	connCfg, err := s.loadConnections(ctx)
	if err != nil {
		return domain.Result[ValidationReport]{Error: err}
	}

	count := 0
	for _, group := range connCfg.Groups {
		for _, subgroup := range group.Subgroups {
			count += len(subgroup.Connections)
		}
	}

	report := ValidationReport{
		Valid:     true,
		ConnCount: count,
	}

	s.cfg.Logger.Info(ctx, "ConnectionService", "ValidateConfig", "valid",
		domainlogger.F("connections", count))

	return domain.Result[ValidationReport]{Data: report}
}

// GetPool returns the connection pool managed by this service.
func (s *ConnectionService) GetPool() domainconn.Pool {
	return s.cfg.Pool
}

// Shutdown gracefully closes all connections and tunnels.
func (s *ConnectionService) Shutdown(ctx context.Context) {
	s.cfg.Logger.Info(ctx, "ConnectionService", "Shutdown", "closing all connections")

	if err := s.cfg.Pool.CloseAll(); err != nil {
		s.cfg.Logger.Error(ctx, "ConnectionService", "Shutdown", "pool close error",
			domainlogger.F("error", err.Error()))
	}

	if err := s.cfg.TunnelManager.CloseAll(); err != nil {
		s.cfg.Logger.Error(ctx, "ConnectionService", "Shutdown", "tunnel close error",
			domainlogger.F("error", err.Error()))
	}
}

// loadConnections is a helper that loads and validates the connections config.
func (s *ConnectionService) loadConnections(ctx context.Context) (*domainconfig.ConnectionsConfig, error) {
	path := filepath.Join(s.cfg.ConfigDir, "connections.toml")
	result, err := s.cfg.Loader.LoadConnections(path)
	if err != nil {
		s.cfg.Logger.Error(ctx, "ConnectionService", "loadConnections", "load error",
			domainlogger.F("error", err.Error()))
		return nil, err
	}
	if result.Error != nil {
		s.cfg.Logger.Error(ctx, "ConnectionService", "loadConnections", "config error",
			domainlogger.F("error", result.Error.Error()))
		return nil, result.Error
	}
	return result.Data, nil
}

// connectorFor returns the appropriate connector for the given DB type.
func (s *ConnectionService) connectorFor(dbType string) domainconn.Connector {
	switch dbType {
	case "mysql":
		return s.cfg.MySQLConnector
	case "postgresql":
		return s.cfg.PGConnector
	default:
		return nil
	}
}

func groupFromPath(path string) string {
	parts := splitPath(path)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func subgroupFromPath(path string) string {
	parts := splitPath(path)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	parts = append(parts, path[start:])
	return parts
}
