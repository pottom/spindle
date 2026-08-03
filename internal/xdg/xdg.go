// Package xdg resolves spindle's directories under the XDG Base Directory
// specification. Go's os.UserConfigDir and os.UserCacheDir follow platform
// convention instead, which on macOS means ~/Library — not where someone who
// keeps their dotfiles in order expects to find things.
package xdg

import (
	"fmt"
	"os"
	"path/filepath"
)

const appName = "spindle"

// ConfigDir is $XDG_CONFIG_HOME/spindle, or ~/.config/spindle. It is created if
// it does not exist, readable only by its owner.
func ConfigDir() (string, error) {
	return appDir("XDG_CONFIG_HOME", ".config", 0o700)
}

// CacheDir is $XDG_CACHE_HOME/spindle, or ~/.cache/spindle.
func CacheDir() (string, error) {
	return appDir("XDG_CACHE_HOME", ".cache", 0o755)
}

func appDir(env, fallback string, perm os.FileMode) (string, error) {
	base := os.Getenv(env)
	if base == "" || !filepath.IsAbs(base) {
		// The spec says a relative value is invalid and must be ignored.
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home directory: %w", err)
		}
		base = filepath.Join(home, fallback)
	}

	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, perm); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}
