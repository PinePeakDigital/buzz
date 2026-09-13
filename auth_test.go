package main

import (
	"os"
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

	t.Run("rejects whitespace-only required fields", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if _, err := parseAndSaveCredentials(`{"username":"   ","auth_token":"secret"}`); err == nil {
			t.Error("expected error for whitespace-only username")
		}
		if _, err := parseAndSaveCredentials(`{"username":"alice","auth_token":"\t"}`); err == nil {
			t.Error("expected error for whitespace-only auth_token")
		}
		if ConfigExists() {
			t.Error("config should not be written for whitespace-only fields")
		}
	})

	t.Run("a second username is added alongside the first", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if _, err := parseAndSaveCredentials(`{"username":"alice","auth_token":"a"}`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		config, err := parseAndSaveCredentials(`{"username":"bob","auth_token":"b"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Username != "alice" {
			t.Errorf("the first account should stay primary, got %q", config.Username)
		}
		if len(config.Accounts) != 1 || config.Accounts[0].Username != "bob" {
			t.Errorf("bob should be added alongside alice, got %+v", config.Accounts)
		}
	})

	t.Run("re-auth refreshes the token and keeps other settings", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		if err := SaveConfig(&Config{Username: "alice", AuthToken: "old", LogFile: "/tmp/buzz.log", BaseURL: "https://example.test"}); err != nil {
			t.Fatal(err)
		}
		config, err := parseAndSaveCredentials(`{"username":"alice","auth_token":"new"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.AuthToken != "new" || len(config.Accounts) != 0 {
			t.Errorf("re-auth should refresh in place, got %+v", config)
		}
		if config.LogFile != "/tmp/buzz.log" || config.BaseURL != "https://example.test" {
			t.Errorf("non-credential settings should survive re-login, got %+v", config)
		}
	})

	t.Run("an unreadable config is moved aside rather than trapping the user", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		path := filepath.Join(tmpDir, ".buzzrc")
		if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
			t.Fatal(err)
		}

		// The TUI sends the user to the auth screen precisely because the config
		// won't load; refusing to save here would leave them no way back in.
		config, err := parseAndSaveCredentials(`{"username":"alice","auth_token":"a"}`)
		if err != nil {
			t.Fatalf("expected the credentials to save, got %v", err)
		}
		if config.Username != "alice" {
			t.Errorf("got %+v, want username=alice", config)
		}
		if _, err := os.Stat(path + ".bak"); err != nil {
			t.Errorf("the unreadable config should be kept as .bak: %v", err)
		}
	})

	t.Run("trims surrounding whitespace from saved fields", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		config, err := parseAndSaveCredentials(`{"username":"  alice  ","auth_token":" secret "}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Username != "alice" || config.AuthToken != "secret" {
			t.Errorf("got %+v, want trimmed username=alice auth_token=secret", config)
		}
	})
}
