package password

import (
	"context"
	"fmt"
	"os"
	"strings"

	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
)

// EnvResolver resolves a password from an environment variable.
// The variable name is either config.Env (if set) or DB_PASSWORD_<NAME>
// where <NAME> is config.ConnectionName uppercased with hyphens replaced by underscores.
type EnvResolver struct{}

// NewEnvResolver creates a new EnvResolver.
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

// Resolve looks up the password from an environment variable.
func (r *EnvResolver) Resolve(_ context.Context, config domainpw.Config) (string, error) {
	varName := config.Env
	if varName == "" {
		varName = envVarName(config.ConnectionName)
	}

	val, ok := os.LookupEnv(varName)
	if !ok {
		return "", fmt.Errorf("%w: environment variable %q not set", domainpw.ErrPasswordResolution, varName)
	}
	return val, nil
}

// envVarName converts a connection name to a DB_PASSWORD_<NAME> env var name.
func envVarName(connectionName string) string {
	upper := strings.ToUpper(connectionName)
	normalized := strings.ReplaceAll(upper, "-", "_")
	return "DB_PASSWORD_" + normalized
}
