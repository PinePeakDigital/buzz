package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubReleasesAPI = "https://api.github.com/repos/pinepeakdigital/buzz/releases/latest"
	checkInterval     = 24 * time.Hour // Check for updates once per day
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// VersionCache stores the last update check
type VersionCache struct {
	LastCheck       time.Time `json:"last_check"`
	LatestVersion   string    `json:"latest_version"`
	UpdateAvailable bool      `json:"update_available"`
	CurrentVersion  string    `json:"current_version"`
}

// getVersionCachePath returns the path to the version cache file
func getVersionCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".buzz_version_cache"), nil
}

// loadVersionCache loads the version cache from disk
func loadVersionCache() (*VersionCache, error) {
	cachePath, err := getVersionCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache file exists yet
		}
		return nil, err
	}

	var cache VersionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// saveVersionCache saves the version cache to disk
func saveVersionCache(cache *VersionCache) error {
	cachePath, err := getVersionCachePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}

// fetchLatestVersion fetches the latest version from GitHub
func fetchLatestVersion() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", githubReleasesAPI, nil)
	if err != nil {
		return "", err
	}

	// Set User-Agent to avoid GitHub rate limiting
	req.Header.Set("User-Agent", "buzz-cli")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// compareVersions compares two semantic versions
// Returns true if newVersion > currentVersion
func compareVersions(currentVersion, newVersion string) bool {
	// Remove 'v' prefix if present
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")

	// Handle dev version - always consider any release as newer
	if currentVersion == "dev" {
		return newVersion != "dev"
	}

	// If versions are identical, no update available
	if currentVersion == newVersion {
		return false
	}

	// Proper semantic version comparison
	return isNewerSemver(currentVersion, newVersion)
}

// isNewerSemver returns true if newVersion > currentVersion according to semantic versioning
func isNewerSemver(currentVersion, newVersion string) bool {
	parse := func(v string) (major, minor, patch int) {
		parts := strings.Split(v, ".")
		if len(parts) < 2 {
			return 0, 0, 0
		}
		// Parse major
		fmt.Sscanf(parts[0], "%d", &major)
		// Parse minor
		fmt.Sscanf(parts[1], "%d", &minor)
		// Parse patch (optional)
		if len(parts) > 2 {
			fmt.Sscanf(parts[2], "%d", &patch)
		}
		return
	}
	maj1, min1, pat1 := parse(currentVersion)
	maj2, min2, pat2 := parse(newVersion)
	if maj2 != maj1 {
		return maj2 > maj1
	}
	if min2 != min1 {
		return min2 > min1
	}
	return pat2 > pat1
}

// checkForUpdates checks if a new version is available
func checkForUpdates() (bool, string, error) {
	// Load cache
	cache, err := loadVersionCache()
	if err != nil {
		// If we can't load cache, proceed with check but don't fail
		cache = nil
	}

	// If cache exists and is fresh, use cached result
	// But first verify that the current version hasn't changed
	if cache != nil && time.Since(cache.LastCheck) < checkInterval {
		// Invalidate cache if current version has changed (user upgraded/downgraded)
		if cache.CurrentVersion != version {
			cache = nil
		} else {
			return cache.UpdateAvailable, cache.LatestVersion, nil
		}
	}

	// Fetch latest version from GitHub
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		// If we have cached data, use it even if stale
		if cache != nil {
			return cache.UpdateAvailable, cache.LatestVersion, nil
		}
		return false, "", err
	}

	// Compare versions
	updateAvailable := compareVersions(version, latestVersion)

	// Save to cache
	newCache := &VersionCache{
		LastCheck:       time.Now(),
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
		CurrentVersion:  version,
	}
	_ = saveVersionCache(newCache) // Ignore errors when saving cache

	return updateAvailable, latestVersion, nil
}

// InstallMethod represents how buzz was installed
type InstallMethod int

const (
	InstallMethodUnknown InstallMethod = iota
	InstallMethodBin
	InstallMethodBrew
)

// detectInstallMethod determines how buzz was installed based on the executable path
func detectInstallMethod() InstallMethod {
	execPath, err := os.Executable()
	if err != nil {
		return InstallMethodUnknown
	}

	// Resolve any symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	// Check for Homebrew installation
	// Homebrew-managed binaries are symlinked to /opt/homebrew/bin/ (Apple Silicon) or /usr/local/bin/ (Intel)
	// The /Cellar/ path is unique to Homebrew installations
	if strings.Contains(realPath, "/Cellar/") ||
		strings.HasPrefix(realPath, "/opt/homebrew/bin/") ||
		strings.HasPrefix(realPath, "/usr/local/bin/") {
		return InstallMethodBrew
	}

	// Check for bin installation
	// bin typically installs to ~/.bin/
	if strings.Contains(realPath, "/.bin/") {
		return InstallMethodBin
	}

	return InstallMethodUnknown
}

// getUpdateCommand returns the appropriate update command based on install method
func getUpdateCommand(method InstallMethod) string {
	switch method {
	case InstallMethodBrew:
		return "brew upgrade narthur/tap/buzz"
	case InstallMethodBin:
		return "bin update buzz"
	default:
		return ""
	}
}

// getUpdateMessage returns a message if an update is available
func getUpdateMessage() string {
	updateAvailable, latestVersion, err := checkForUpdates()
	if err != nil {
		// Silently ignore errors - don't disrupt user's workflow
		return ""
	}

	if updateAvailable {
		method := detectInstallMethod()
		updateCmd := getUpdateCommand(method)

		if updateCmd != "" {
			return fmt.Sprintf("\nℹ️  Update available: %s → %s\n   Run: %s\n", version, latestVersion, updateCmd)
		}
		return fmt.Sprintf("\nℹ️  Update available: %s → %s\n   See https://github.com/pinepeakdigital/buzz#installation for update instructions\n", version, latestVersion)
	}

	return ""
}
