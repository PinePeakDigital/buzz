package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds the Beeminder API credentials
type Config struct {
	Username  string `json:"username"`
	AuthToken string `json:"auth_token"`
	BaseURL   string `json:"base_url,omitempty"` // Optional base URL for API, defaults to https://www.beeminder.com
	LogFile   string `json:"log_file,omitempty"` // Optional path to log file
	// Accounts holds *additional* Beeminder logins beyond the primary
	// Username/AuthToken above. Keeping the primary in its original top-level
	// fields means every existing ~/.buzzrc keeps working untouched — there is
	// no migration, and a single-account config is byte-identical to before.
	Accounts []Account `json:"accounts,omitempty"`
}

// Account is one additional Beeminder login. BaseURL and LogFile are not
// per-account: they configure how buzz talks to Beeminder and where it logs,
// not who it talks as.
type Account struct {
	Username  string `json:"username"`
	AuthToken string `json:"auth_token"`
}

// accountConfigs returns one *Config per configured account, primary first,
// each carrying the shared BaseURL/LogFile. Every account therefore gets an
// HTTPClient that behaves exactly as the single-account one always has.
//
// An unset primary is skipped rather than returned as a blank-credentialled
// account, so a hand-written ~/.buzzrc that lists only "accounts" works, and a
// config emptied by `buzz auth logout` of the last account returns nothing at
// all. Duplicate usernames are dropped, keeping the invariant setAccount
// maintains in memory — one entry per username — true for a hand-edited file
// too; without this, a username listed twice makes every one of its goals look
// ambiguous to multiClient, with no qualifier able to resolve it.
func (c *Config) accountConfigs() []*Config {
	var configs []*Config
	seen := make(map[string]bool)
	add := func(username, authToken string) {
		if username == "" || seen[username] {
			return
		}
		seen[username] = true
		configs = append(configs, &Config{Username: username, AuthToken: authToken, BaseURL: c.BaseURL, LogFile: c.LogFile})
	}
	add(c.Username, c.AuthToken)
	for _, a := range c.Accounts {
		add(a.Username, a.AuthToken)
	}
	return configs
}

// hasCredentials reports whether any account is configured. `buzz auth logout`
// of the last account leaves a valid, parseable ~/.buzzrc with nothing in it,
// so "the file exists and parses" is no longer enough to mean "authenticated".
func (c *Config) hasCredentials() bool {
	return len(c.accountConfigs()) > 0
}

// checkAccounts validates the config against the global --account filter. It is
// the single gate every entry point runs before building a client, so an
// unconfigured --account username is rejected once, up front, rather than
// silently widening to act as every account.
func (c *Config) checkAccounts() error {
	configs := c.accountConfigs()
	if len(configs) == 0 {
		return fmt.Errorf("no accounts configured. Please run 'buzz auth login' to authenticate")
	}
	if accountFilter == "" {
		return nil
	}
	names := make([]string, len(configs))
	for i, cfg := range configs {
		names[i] = cfg.Username
		if cfg.Username == accountFilter {
			return nil
		}
	}
	return fmt.Errorf("no such account: %s (configured: %s)", accountFilter, strings.Join(names, ", "))
}

// setAccount adds a login, or replaces the stored token if that username is
// already configured. Re-authenticating as a user you already have is a token
// refresh, not a second copy of the same account.
func (c *Config) setAccount(username, authToken string) {
	if c.Username == "" || c.Username == username {
		c.Username, c.AuthToken = username, authToken
		return
	}
	for i := range c.Accounts {
		if c.Accounts[i].Username == username {
			c.Accounts[i].AuthToken = authToken
			return
		}
	}
	c.Accounts = append(c.Accounts, Account{Username: username, AuthToken: authToken})
}

// removeAccount drops a login by username, promoting the first additional
// account to primary if the primary itself is removed. Reports whether the
// username was found.
func (c *Config) removeAccount(username string) bool {
	if c.Username == username {
		if len(c.Accounts) == 0 {
			c.Username, c.AuthToken = "", ""
			return true
		}
		c.Username, c.AuthToken = c.Accounts[0].Username, c.Accounts[0].AuthToken
		c.Accounts = c.Accounts[1:]
		return true
	}
	for i, a := range c.Accounts {
		if a.Username == username {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			return true
		}
	}
	return false
}

// accountLabel names the configured account(s) for the grid header. With
// several accounts the header would otherwise claim the goals belong to the
// primary alone, when the grid is showing everyone's.
func accountLabel(c *Config) string {
	// --account scopes the whole session to one account, so that is whose goals
	// the grid is showing — listing the others would misstate what's on screen.
	if accountFilter != "" {
		return accountFilter
	}
	configs := c.accountConfigs()
	names := make([]string, len(configs))
	for i, cfg := range configs {
		names[i] = cfg.Username
	}
	return strings.Join(names, ", ")
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
