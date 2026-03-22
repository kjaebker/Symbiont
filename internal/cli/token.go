package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

// LoadToken resolves the API token using the priority order:
// 1. --token flag
// 2. SYMBIONT_TOKEN env var
// 3. ~/.config/symbiont/token file
func LoadToken(flags *pflag.FlagSet) (string, error) {
	// 1. Flag.
	if flags != nil {
		if t, _ := flags.GetString("token"); t != "" {
			return t, nil
		}
	}

	// 2. Environment variable.
	if t := os.Getenv("SYMBIONT_TOKEN"); t != "" {
		return t, nil
	}

	// 3. Config file.
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ".config", "symbiont", "token")
		data, err := os.ReadFile(path)
		if err == nil {
			t := strings.TrimSpace(string(data))
			if t != "" {
				return t, nil
			}
		}
	}

	return "", fmt.Errorf("no API token found — set SYMBIONT_TOKEN, use --token, or save to ~/.config/symbiont/token")
}

// SaveToken writes the token to ~/.config/symbiont/token.
func SaveToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "symbiont")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing token file: %w", err)
	}
	return nil
}
