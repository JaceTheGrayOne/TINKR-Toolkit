package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath cleans and normalizes a user-provided path.
// Handles quotes, whitespace, environment variables, and ~ expansion.
func NormalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.Trim(path, "'")
	path = os.ExpandEnv(path)

	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("couldn't expand ~: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("couldn't convert to absolute path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)
	cleanPath = strings.TrimSuffix(cleanPath, string(filepath.Separator))

	return cleanPath, nil
}

// GetExecutableDir returns the directory containing the running executable.
func GetExecutableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
