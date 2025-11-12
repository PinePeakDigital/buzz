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
	LastCheck      time.Time `json:"last_check"`
	LatestVersion  string    `json:"latest_version"`
	UpdateAvailable bool     `json:"update_available"`
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

	// Simple string comparison for semantic versions
	// This works for most cases (v0.30.1 vs v0.31.0)
	return newVersion > currentVersion
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
	if cache != nil && time.Since(cache.LastCheck) < checkInterval {
		return cache.UpdateAvailable, cache.LatestVersion, nil
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
	}
	_ = saveVersionCache(newCache) // Ignore errors when saving cache

	return updateAvailable, latestVersion, nil
}

// getUpdateMessage returns a message if an update is available
func getUpdateMessage() string {
	updateAvailable, latestVersion, err := checkForUpdates()
	if err != nil {
		// Silently ignore errors - don't disrupt user's workflow
		return ""
	}

	if updateAvailable {
		return fmt.Sprintf("\nℹ️  Update available: %s → %s\n   Visit https://github.com/pinepeakdigital/buzz/releases/latest to upgrade\n", version, latestVersion)
	}

	return ""
}
