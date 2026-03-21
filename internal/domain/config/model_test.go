package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
)

// --- FindConnection ---

func TestConnectionsConfig_FindConnection_Found(t *testing.T) {
	cfg := makeTestConfig()

	entry, err := cfg.FindConnection("prod.write.main-db")
	require.NoError(t, err)
	assert.Equal(t, "main-db", entry.Name)
	assert.Equal(t, "mysql", entry.Type)
	assert.Equal(t, "db.prod.example.com", entry.Host)
}

func TestConnectionsConfig_FindConnection_NotFound(t *testing.T) {
	cfg := makeTestConfig()

	_, err := cfg.FindConnection("prod.write.nonexistent")
	assert.ErrorIs(t, err, config.ErrConnectionNotFound)
}

func TestConnectionsConfig_FindConnection_InvalidPath(t *testing.T) {
	cfg := makeTestConfig()

	_, err := cfg.FindConnection("prod.write")
	assert.Error(t, err)

	_, err = cfg.FindConnection("")
	assert.Error(t, err)
}

func TestConnectionsConfig_FindConnection_GroupNotFound(t *testing.T) {
	cfg := makeTestConfig()

	_, err := cfg.FindConnection("staging.write.main-db")
	assert.ErrorIs(t, err, config.ErrConnectionNotFound)
}

// --- ConnectionsByEnv ---

func TestConnectionsConfig_ConnectionsByEnv_Production(t *testing.T) {
	cfg := makeTestConfig()

	conns := cfg.ConnectionsByEnv(config.Production)
	require.Len(t, conns, 1)
	assert.Equal(t, "main-db", conns[0].Name)
}

func TestConnectionsConfig_ConnectionsByEnv_Staging(t *testing.T) {
	cfg := makeTestConfig()

	conns := cfg.ConnectionsByEnv(config.Staging)
	require.Len(t, conns, 1)
	assert.Equal(t, "replica", conns[0].Name)
}

func TestConnectionsConfig_ConnectionsByEnv_Empty(t *testing.T) {
	cfg := makeTestConfig()

	conns := cfg.ConnectionsByEnv(config.Test)
	assert.Empty(t, conns)
}

func TestConnectionsConfig_ConnectionsByEnv_UnknownEnv(t *testing.T) {
	cfg := makeTestConfig()

	conns := cfg.ConnectionsByEnv(config.Environment("custom-env"))
	require.Len(t, conns, 1)
	assert.Equal(t, "custom-conn", conns[0].Name)
}

// --- ValidateReferences ---

func TestConnectionsConfig_ValidateReferences_Valid(t *testing.T) {
	cfg := makeTestConfig()
	// Add a tunnel and a connection referencing it
	cfg.SSHTunnels["bastion"] = config.SSHTunnelEntry{
		Host: "bastion.example.com",
		Port: 22,
		User: "ec2-user",
	}
	cfg.Groups["prod"].Subgroups["write"].Connections["tunneled"] = config.ConnectionEntry{
		Name:      "tunneled",
		Type:      "mysql",
		Host:      "internal.db",
		Port:      3306,
		SSHTunnel: "bastion",
	}

	err := cfg.ValidateReferences()
	assert.NoError(t, err)
}

func TestConnectionsConfig_ValidateReferences_DanglingRef(t *testing.T) {
	cfg := makeTestConfig()
	cfg.Groups["prod"].Subgroups["write"].Connections["bad"] = config.ConnectionEntry{
		Name:      "bad",
		Type:      "mysql",
		Host:      "db.example.com",
		Port:      3306,
		SSHTunnel: "nonexistent-tunnel",
	}

	err := cfg.ValidateReferences()
	assert.ErrorIs(t, err, config.ErrDanglingTunnelRef)
}

func TestConnectionsConfig_ValidateReferences_NoTunnels_NoRefs_OK(t *testing.T) {
	cfg := makeTestConfig()
	err := cfg.ValidateReferences()
	assert.NoError(t, err)
}

// --- Environment constants ---

func TestEnvironment_Constants(t *testing.T) {
	assert.Equal(t, config.Environment("production"), config.Production)
	assert.Equal(t, config.Environment("staging"), config.Staging)
	assert.Equal(t, config.Environment("test"), config.Test)
}

func TestEnvironment_UnknownStoredAsIs(t *testing.T) {
	env := config.Environment("my-custom-env")
	assert.Equal(t, "my-custom-env", string(env))
}

// --- helpers ---

func makeTestConfig() *config.ConnectionsConfig {
	prodWrite := &config.Subgroup{
		Connections: map[string]config.ConnectionEntry{
			"main-db": {
				Name: "main-db",
				Type: "mysql",
				Host: "db.prod.example.com",
				Port: 3306,
				Env:  config.Production,
			},
		},
	}
	prodRead := &config.Subgroup{
		Connections: map[string]config.ConnectionEntry{
			"replica": {
				Name: "replica",
				Type: "mysql",
				Host: "replica.prod.example.com",
				Port: 3306,
				Env:  config.Staging,
			},
			"custom-conn": {
				Name: "custom-conn",
				Type: "postgresql",
				Host: "pg.example.com",
				Port: 5432,
				Env:  config.Environment("custom-env"),
			},
		},
	}

	return &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"prod": {
				Subgroups: map[string]*config.Subgroup{
					"write": prodWrite,
					"read":  prodRead,
				},
			},
		},
		SSHTunnels: map[string]config.SSHTunnelEntry{},
	}
}
