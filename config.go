package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds the Beeminder API credentials
type Config struct {
	Username  string `json:"username"`
	AuthToken string `json:"auth_token"`
	BaseURL   string `json:"base_url,omitempty"` // Optional base URL for API, defaults to https://www.beeminder.com
	LogFile   string `json:"log_file,omitempty"` // Optional path to log file
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

// createRefreshFlag creates the refresh flag file with current Unix timestamp
func createRefreshFlag() error {
	path, err := getRefreshFlagPath()
	if err != nil {
		return err
	}
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	return os.WriteFile(path, []byte(timestamp), 0600)
}

// deleteRefreshFlag deletes the refresh flag file
func deleteRefreshFlag() error {
	path, err := getRefreshFlagPath()
	if err != nil {
		return err
	}
	// Remove the file, but ignore "file not found" errors
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
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

// getRefreshFlagTimestamp reads and returns the timestamp from the refresh flag file
// Returns 0 if the file doesn't exist or contains invalid data
func getRefreshFlagTimestamp() int64 {
	path, err := getRefreshFlagPath()
	if err != nil {
		return 0
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	timestamp, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0
	}

	return timestamp
}

// logToFile writes a log entry to the configured log file
// If config.LogFile is empty, logging is disabled and this function does nothing
func logToFile(config *Config, message string) {
	if config == nil || config.LogFile == "" {
		return // Logging disabled
	}

	f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Fail silently if can't open log
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	// Intentionally ignore write errors to fail silently and not disrupt normal operations
	f.WriteString(logEntry)
}

// LogRequest logs HTTP request details to the configured log file
func LogRequest(config *Config, method, url string) {
	logToFile(config, fmt.Sprintf("REQUEST: %s %s", method, redactAuthToken(url)))
}

// LogResponse logs HTTP response details to the configured log file
func LogResponse(config *Config, statusCode int, url string) {
	logToFile(config, fmt.Sprintf("RESPONSE: %d %s", statusCode, redactAuthToken(url)))
}
