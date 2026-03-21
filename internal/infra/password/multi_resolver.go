package password

import (
	"context"
	"fmt"
	"os"
	"strings"

	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
)

// MultiResolver selects the appropriate password resolver based on priority:
// cmd > env > prompt > plaintext. A failing chosen method is a hard error.
type MultiResolver struct {
	cmd       *CmdResolver
	env       *EnvResolver
	plaintext *PlaintextResolver
}

// NewMultiResolver creates a MultiResolver with all sub-resolvers wired.
func NewMultiResolver(log domainlogger.Logger) *MultiResolver {
	return &MultiResolver{
		cmd:       NewCmdResolver(),
		env:       NewEnvResolver(),
		plaintext: NewPlaintextResolver(log),
	}
}

// Resolve picks the highest-priority configured method and executes it.
// If the chosen method fails it returns a hard error (no fallback).
func (r *MultiResolver) Resolve(ctx context.Context, config domainpw.Config) (string, error) {
	// Priority 1: password_cmd
	if config.Cmd != "" {
		return r.cmd.Resolve(ctx, config)
	}

	// Priority 2: env var (either explicit config.Env or derived DB_PASSWORD_<NAME>)
	envName := config.Env
	if envName == "" {
		upper := strings.ToUpper(config.ConnectionName)
		envName = "DB_PASSWORD_" + strings.ReplaceAll(upper, "-", "_")
	}
	if _, ok := os.LookupEnv(envName); ok {
		return r.env.Resolve(ctx, config)
	}

	// Priority 3: plaintext
	if config.Plaintext != "" {
		return r.plaintext.Resolve(ctx, config)
	}

	return "", fmt.Errorf("%w: no password method configured for %q", domainpw.ErrNoPasswordMethod, config.ConnectionName)
}
