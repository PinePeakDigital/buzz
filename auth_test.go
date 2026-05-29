package main

import (
	"path/filepath"
	"testing"
)

// TestParseAndSaveCredentials covers the shared credentials parsing/validation/
// save helper used by both the TUI auth screen and `buzz auth login`. HOME is
// redirected to a temp dir so the real ~/.buzzrc is never touched.
func TestParseAndSaveCredentials(t *testing.T) {
	t.Run("valid credentials are parsed and saved", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		config, err := parseAndSaveCredentials(`{"username":"alice","auth_token":"secret"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Username != "alice" || config.AuthToken != "secret" {
			t.Errorf("got %+v, want username=alice auth_token=secret", config)
		}

		// The config file should now exist and round-trip.
		if !ConfigExists() {
			t.Fatalf("expected config file at %s", filepath.Join(tmpDir, ".buzzrc"))
		}
		loaded, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if loaded.Username != "alice" || loaded.AuthToken != "secret" {
			t.Errorf("loaded %+v, want username=alice auth_token=secret", loaded)
		}
	})

	t.Run("surrounding whitespace is trimmed", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if _, err := parseAndSaveCredentials("  \n{\"username\":\"bob\",\"auth_token\":\"t\"}\n  "); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects empty input", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if _, err := parseAndSaveCredentials("   \n  "); err == nil {
			t.Error("expected error for empty input")
		}
		if ConfigExists() {
			t.Error("config should not be written for empty input")
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if _, err := parseAndSaveCredentials("not json"); err == nil {
			t.Error("expected error for invalid JSON")
		}
		if ConfigExists() {
			t.Error("config should not be written for invalid JSON")
		}
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if _, err := parseAndSaveCredentials(`{"username":"alice"}`); err == nil {
			t.Error("expected error when auth_token is missing")
		}
		if _, err := parseAndSaveCredentials(`{"auth_token":"secret"}`); err == nil {
			t.Error("expected error when username is missing")
		}
		if ConfigExists() {
			t.Error("config should not be written when required fields are missing")
		}
	})
}
