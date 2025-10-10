package main

import (
	"encoding/json"
	"testing"
)

// TestConfigStructMarshaling tests the Config struct JSON marshaling
func TestConfigStructMarshaling(t *testing.T) {
	config := &Config{
		Username:  "myusername",
		AuthToken: "myauthtoken",
	}

	// Marshal to JSON
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back
	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify fields
	if decoded.Username != config.Username {
		t.Errorf("Username = %q, want %q", decoded.Username, config.Username)
	}
	if decoded.AuthToken != config.AuthToken {
		t.Errorf("AuthToken = %q, want %q", decoded.AuthToken, config.AuthToken)
	}

	// Verify JSON field names
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := jsonMap["username"]; !exists {
		t.Error("JSON should have 'username' field")
	}
	if _, exists := jsonMap["auth_token"]; !exists {
		t.Error("JSON should have 'auth_token' field")
	}
}

// TestRefreshFlagFunctions tests the refresh flag file operations
func TestRefreshFlagFunctions(t *testing.T) {
	// Clean up any existing flag file before test
	deleteRefreshFlag()

	t.Run("getRefreshFlagPath returns valid path", func(t *testing.T) {
		path, err := getRefreshFlagPath()
		if err != nil {
			t.Fatalf("getRefreshFlagPath() error = %v", err)
		}
		if path == "" {
			t.Error("getRefreshFlagPath() returned empty path")
		}
	})

	t.Run("refreshFlagExists returns false when flag does not exist", func(t *testing.T) {
		// Ensure flag doesn't exist
		deleteRefreshFlag()

		if refreshFlagExists() {
			t.Error("refreshFlagExists() = true, want false when flag doesn't exist")
		}
	})

	t.Run("createRefreshFlag creates flag file", func(t *testing.T) {
		// Clean up first
		deleteRefreshFlag()

		if err := createRefreshFlag(); err != nil {
			t.Fatalf("createRefreshFlag() error = %v", err)
		}

		if !refreshFlagExists() {
			t.Error("Flag file should exist after createRefreshFlag()")
		}

		// Clean up
		deleteRefreshFlag()
	})

	t.Run("deleteRefreshFlag removes flag file", func(t *testing.T) {
		// Create flag first
		createRefreshFlag()

		if err := deleteRefreshFlag(); err != nil {
			t.Fatalf("deleteRefreshFlag() error = %v", err)
		}

		if refreshFlagExists() {
			t.Error("Flag file should not exist after deleteRefreshFlag()")
		}
	})

	t.Run("deleteRefreshFlag does not error when flag doesn't exist", func(t *testing.T) {
		// Ensure flag doesn't exist
		deleteRefreshFlag()

		// Should not error even if flag doesn't exist
		if err := deleteRefreshFlag(); err != nil {
			t.Errorf("deleteRefreshFlag() error = %v, want nil when flag doesn't exist", err)
		}
	})

	t.Run("multiple create/delete cycles", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			if err := createRefreshFlag(); err != nil {
				t.Fatalf("cycle %d: createRefreshFlag() error = %v", i, err)
			}

			if !refreshFlagExists() {
				t.Errorf("cycle %d: flag should exist after create", i)
			}

			if err := deleteRefreshFlag(); err != nil {
				t.Fatalf("cycle %d: deleteRefreshFlag() error = %v", i, err)
			}

			if refreshFlagExists() {
				t.Errorf("cycle %d: flag should not exist after delete", i)
			}
		}
	})

	// Clean up after all tests
	deleteRefreshFlag()
}
