package password

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
)

// PromptResolver resolves a password by reading from an io.Reader.
// In production, inject os.Stdin. In tests, inject a strings.Reader.
type PromptResolver struct {
	reader io.Reader
}

// NewPromptResolver creates a PromptResolver that reads from r.
func NewPromptResolver(r io.Reader) *PromptResolver {
	return &PromptResolver{reader: r}
}

// Resolve reads a line from the injected reader and trims whitespace.
func (r *PromptResolver) Resolve(_ context.Context, config domainpw.Config) (string, error) {
	scanner := bufio.NewScanner(r.reader)
	if scanner.Scan() {
		pw := strings.TrimSpace(scanner.Text())
		if pw == "" {
			return "", fmt.Errorf("%w: empty password entered for %q", domainpw.ErrPasswordResolution, config.ConnectionName)
		}
		return pw, nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("%w: reading password for %q: %v", domainpw.ErrPasswordResolution, config.ConnectionName, err)
	}
	return "", fmt.Errorf("%w: no input provided for %q", domainpw.ErrPasswordResolution, config.ConnectionName)
}
