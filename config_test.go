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
