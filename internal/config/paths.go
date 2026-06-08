package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func DefaultPaths() (Paths, error) {
	dataDir, err := defaultDataDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		DataDir:    dataDir,
		ConfigFile: filepath.Join(dataDir, ConfigFileName),
		LogDir:     filepath.Join(dataDir, LogDirName),
		LogFile:    filepath.Join(dataDir, LogDirName, LogFileName),
	}, nil
}

func ExpandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if len(path) >= 2 && path[:2] == "~"+string(os.PathSeparator) {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func defaultDataDir() (string, error) {
	if runtime.GOOS == "windows" {
		configDir, err := os.UserConfigDir()
		if err == nil && configDir != "" {
			return filepath.Join(configDir, AppName), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".vietclaw"), nil
}
