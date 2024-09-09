package common

import (
	"os"
	"path/filepath"
)

// GetRelativePath determines the file path to the configuration file (config.yaml).
// It first checks if the config file exists in the current working directory.
// If the file is not found, it then checks within a "client" subdirectory.
// Returns the full path to the config.yaml file as a string.
func GetRelativePath(path string) string {
	pwd, _ := os.Getwd()
	configPath := filepath.Join(pwd, path)
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	return filepath.Join(pwd, "client", path)
}
