package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		newVersion     string
		expected       bool
	}{
		{
			name:           "newer version available",
			currentVersion: "v0.30.0",
			newVersion:     "v0.31.0",
			expected:       true,
		},
		{
			name:           "same version",
			currentVersion: "v0.30.0",
			newVersion:     "v0.30.0",
			expected:       false,
		},
		{
			name:           "older version",
			currentVersion: "v0.31.0",
			newVersion:     "v0.30.0",
			expected:       false,
		},
		{
			name:           "dev version with newer release",
			currentVersion: "dev",
			newVersion:     "v0.30.0",
			expected:       true,
		},
		{
			name:           "dev version same",
			currentVersion: "dev",
			newVersion:     "dev",
			expected:       false,
		},
		{
			name:           "versions without v prefix",
			currentVersion: "0.30.0",
			newVersion:     "0.31.0",
			expected:       true,
		},
		{
			name:           "minor version bump",
			currentVersion: "v0.30.1",
			newVersion:     "v0.30.2",
			expected:       true,
		},
		{
			name:           "major version bump",
			currentVersion: "v0.30.0",
			newVersion:     "v1.0.0",
			expected:       true,
		},
		{
			name:           "double digit minor version",
			currentVersion: "0.9.0",
			newVersion:     "0.10.0",
			expected:       true,
		},
		{
			name:           "triple digit minor version",
			currentVersion: "0.2.0",
			newVersion:     "0.100.0",
			expected:       true,
		},
		{
			name:           "double digit patch version",
			currentVersion: "0.30.9",
			newVersion:     "0.30.10",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.currentVersion, tt.newVersion)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %v, expected %v",
					tt.currentVersion, tt.newVersion, result, tt.expected)
			}
		})
	}
}

func TestVersionCacheSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Create a test cache
	testCache := &VersionCache{
		LastCheck:       time.Now(),
		LatestVersion:   "v0.31.0",
		UpdateAvailable: true,
	}

	// Save the cache
	err = saveVersionCache(testCache)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify the file exists
	cachePath, err := getVersionCachePath()
	if err != nil {
		t.Fatalf("Failed to get cache path: %v", err)
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("Cache file was not created at %s", cachePath)
	}

	// Load the cache
	loadedCache, err := loadVersionCache()
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	if loadedCache == nil {
		t.Fatal("Loaded cache is nil")
	}

	// Verify the loaded data matches
	if loadedCache.LatestVersion != testCache.LatestVersion {
		t.Errorf("LatestVersion mismatch: got %q, expected %q",
			loadedCache.LatestVersion, testCache.LatestVersion)
	}

	if loadedCache.UpdateAvailable != testCache.UpdateAvailable {
		t.Errorf("UpdateAvailable mismatch: got %v, expected %v",
			loadedCache.UpdateAvailable, testCache.UpdateAvailable)
	}
}

func TestLoadVersionCacheNoFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Try to load non-existent cache
	cache, err := loadVersionCache()
	if err != nil {
		t.Errorf("Expected no error when cache doesn't exist, got: %v", err)
	}

	if cache != nil {
		t.Error("Expected nil cache when file doesn't exist")
	}
}

func TestVersionCacheInvalidJSON(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Write invalid JSON to cache file
	cachePath := filepath.Join(tmpDir, ".buzz_version_cache")
	err = os.WriteFile(cachePath, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Try to load invalid cache
	_, err = loadVersionCache()
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestCheckForUpdatesWithFreshCache(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Create a fresh cache
	freshCache := &VersionCache{
		LastCheck:       time.Now(),
		LatestVersion:   "v0.31.0",
		UpdateAvailable: true,
	}

	err = saveVersionCache(freshCache)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Check for updates - should use cache
	updateAvailable, latestVersion, err := checkForUpdates()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !updateAvailable {
		t.Error("Expected update to be available based on cache")
	}

	if latestVersion != "v0.31.0" {
		t.Errorf("Expected version v0.31.0, got %s", latestVersion)
	}
}

func TestCheckForUpdatesWithStaleCache(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Create a stale cache (more than 24 hours old)
	staleCache := &VersionCache{
		LastCheck:       time.Now().Add(-25 * time.Hour),
		LatestVersion:   "v0.30.0",
		UpdateAvailable: false,
	}

	err = saveVersionCache(staleCache)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Check for updates - will try to fetch from GitHub
	// This test might fail in offline environments, so we don't assert specific results
	// Just verify it doesn't crash
	_, _, _ = checkForUpdates()
}

func TestGetUpdateMessageNoUpdate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Create a cache indicating no update
	cache := &VersionCache{
		LastCheck:       time.Now(),
		LatestVersion:   "v0.30.0",
		UpdateAvailable: false,
	}

	err = saveVersionCache(cache)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Get update message
	msg := getUpdateMessage()
	if msg != "" {
		t.Errorf("Expected empty message when no update available, got: %s", msg)
	}
}

func TestGetUpdateMessageWithUpdate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "buzz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the cache path for testing
	originalHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHomeDir)

	// Create a cache indicating update is available
	cache := &VersionCache{
		LastCheck:       time.Now(),
		LatestVersion:   "v0.31.0",
		UpdateAvailable: true,
	}

	err = saveVersionCache(cache)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Get update message
	msg := getUpdateMessage()
	if msg == "" {
		t.Error("Expected non-empty message when update available")
	}

	// Verify message contains version information
	if !strings.Contains(msg, "v0.31.0") {
		t.Errorf("Expected message to contain version v0.31.0, got: %s", msg)
	}
}

func TestVersionCacheJSON(t *testing.T) {
	cache := &VersionCache{
		LastCheck:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		LatestVersion:   "v0.31.0",
		UpdateAvailable: true,
	}

	// Marshal to JSON
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}

	// Unmarshal back
	var loaded VersionCache
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal cache: %v", err)
	}

	if loaded.LatestVersion != cache.LatestVersion {
		t.Errorf("LatestVersion mismatch: got %q, expected %q",
			loaded.LatestVersion, cache.LatestVersion)
	}

	if loaded.UpdateAvailable != cache.UpdateAvailable {
		t.Errorf("UpdateAvailable mismatch: got %v, expected %v",
			loaded.UpdateAvailable, cache.UpdateAvailable)
	}
}
