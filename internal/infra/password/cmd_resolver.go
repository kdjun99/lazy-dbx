// Package password provides implementations of the domain password.Resolver interface.
package password

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
)

// CmdResolver resolves a password by executing a shell command.
type CmdResolver struct{}

// NewCmdResolver creates a new CmdResolver.
func NewCmdResolver() *CmdResolver {
	return &CmdResolver{}
}

// Resolve executes config.Cmd via "sh -c" and returns trimmed stdout.
func (r *CmdResolver) Resolve(ctx context.Context, config domainpw.Config) (string, error) {
	//nolint:gosec // G204: password_cmd comes from local config files (trusted source, no user interpolation)
	cmd := exec.CommandContext(ctx, "sh", "-c", config.Cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: command %q failed: %v", domainpw.ErrPasswordResolution, config.Cmd, err)
	}
	return strings.TrimSpace(string(out)), nil
}
