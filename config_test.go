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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
