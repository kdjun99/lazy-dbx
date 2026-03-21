// Package config provides infrastructure implementations for configuration loading.
package config

import (
	"os"
	"strings"
)

// ExpandPath resolves a leading tilde (~) to the user's home directory.
// Paths without a leading tilde are returned unchanged.
func ExpandPath(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	if p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home + p[1:], nil
	}
	return p, nil
}
