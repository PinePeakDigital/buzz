package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the Beeminder API credentials
type Config struct {
	Username  string `json:"username"`
	AuthToken string `json:"auth_token"`
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".buzzrc"), nil
}

// ConfigExists checks if the config file exists
func ConfigExists() bool {
	path, err := getConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// LoadConfig reads and parses the config file from ~/.buzzrc
func LoadConfig() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig writes the config to ~/.buzzrc with secure permissions
func SaveConfig(config *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// Write with 0600 permissions (read/write for owner only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	return nil
}

// getRefreshFlagPath returns the path to the refresh flag file
func getRefreshFlagPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".buzz-refresh"), nil
}

// createRefreshFlag creates the refresh flag file
func createRefreshFlag() error {
	path, err := getRefreshFlagPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte{}, 0600)
}

// deleteRefreshFlag deletes the refresh flag file
func deleteRefreshFlag() error {
	path, err := getRefreshFlagPath()
	if err != nil {
		return err
	}
	// Ignore error if file doesn't exist
	os.Remove(path)
	return nil
}

// refreshFlagExists checks if the refresh flag file exists
func refreshFlagExists() bool {
	path, err := getRefreshFlagPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
