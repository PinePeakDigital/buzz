package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
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

// TestRefreshFlagTimestamp tests the timestamp-based refresh flag operations
func TestRefreshFlagTimestamp(t *testing.T) {
	// Clean up any existing flag file before test
	deleteRefreshFlag()

	t.Run("getRefreshFlagTimestamp returns 0 when flag does not exist", func(t *testing.T) {
		// Ensure flag doesn't exist
		deleteRefreshFlag()

		timestamp := getRefreshFlagTimestamp()
		if timestamp != 0 {
			t.Errorf("getRefreshFlagTimestamp() = %d, want 0 when flag doesn't exist", timestamp)
		}
	})

	t.Run("createRefreshFlag writes valid timestamp", func(t *testing.T) {
		// Clean up first
		deleteRefreshFlag()

		// Create the flag
		beforeTime := time.Now().Unix()
		if err := createRefreshFlag(); err != nil {
			t.Fatalf("createRefreshFlag() error = %v", err)
		}
		afterTime := time.Now().Unix()

		// Get the timestamp
		timestamp := getRefreshFlagTimestamp()
		if timestamp < beforeTime || timestamp > afterTime {
			t.Errorf("getRefreshFlagTimestamp() = %d, want timestamp between %d and %d", timestamp, beforeTime, afterTime)
		}

		// Clean up
		deleteRefreshFlag()
	})

	t.Run("multiple instances can read same timestamp", func(t *testing.T) {
		// Clean up first
		deleteRefreshFlag()

		// Create the flag
		if err := createRefreshFlag(); err != nil {
			t.Fatalf("createRefreshFlag() error = %v", err)
		}

		// Read timestamp multiple times (simulating multiple instances)
		timestamp1 := getRefreshFlagTimestamp()
		timestamp2 := getRefreshFlagTimestamp()
		timestamp3 := getRefreshFlagTimestamp()

		if timestamp1 == 0 {
			t.Error("First read should return valid timestamp")
		}
		if timestamp1 != timestamp2 || timestamp1 != timestamp3 {
			t.Errorf("All reads should return same timestamp: got %d, %d, %d", timestamp1, timestamp2, timestamp3)
		}

		// Clean up
		deleteRefreshFlag()
	})

	t.Run("timestamp updates on new createRefreshFlag call", func(t *testing.T) {
		// Clean up first
		deleteRefreshFlag()

		// Create first flag
		if err := createRefreshFlag(); err != nil {
			t.Fatalf("createRefreshFlag() error = %v", err)
		}
		timestamp1 := getRefreshFlagTimestamp()

		// Wait 1 second to ensure different Unix timestamp
		time.Sleep(1 * time.Second)

		// Create second flag (overwrites first)
		if err := createRefreshFlag(); err != nil {
			t.Fatalf("createRefreshFlag() error = %v", err)
		}
		timestamp2 := getRefreshFlagTimestamp()

		if timestamp2 <= timestamp1 {
			t.Errorf("Second timestamp (%d) should be greater than first (%d)", timestamp2, timestamp1)
		}

		// Clean up
		deleteRefreshFlag()
	})

	// Clean up after all tests
	deleteRefreshFlag()
}

// TestLoggingFunctionality tests the logging feature
func TestLoggingFunctionality(t *testing.T) {
t.Run("LogRequest does nothing when LogFile is empty", func(t *testing.T) {
config := &Config{
Username:  "test",
AuthToken: "token",
LogFile:   "", // Empty means disabled
}
// Should not panic or error
LogRequest(config, "GET", "http://example.com")
LogResponse(config, 200, "http://example.com")
})

t.Run("LogRequest does nothing when config is nil", func(t *testing.T) {
// Should not panic or error
LogRequest(nil, "GET", "http://example.com")
LogResponse(nil, 200, "http://example.com")
})

t.Run("LogRequest writes to file when LogFile is set", func(t *testing.T) {
// Create a temp file for testing
logFile := "/tmp/buzz_test_log.txt"
defer func() {
// Clean up
os.Remove(logFile)
}()

config := &Config{
Username:  "test",
AuthToken: "token",
LogFile:   logFile,
}

// Log a request
LogRequest(config, "GET", "http://example.com/api")

// Verify file exists
if _, err := os.Stat(logFile); os.IsNotExist(err) {
t.Error("Log file should exist after LogRequest")
}

// Read and verify content
data, err := os.ReadFile(logFile)
if err != nil {
t.Fatalf("Failed to read log file: %v", err)
}

content := string(data)
if !strings.Contains(content, "REQUEST: GET http://example.com/api") {
t.Errorf("Log content should contain request details, got: %s", content)
}
if !strings.Contains(content, "[20") { // Check for timestamp format [20XX-XX-XX ...]
t.Errorf("Log content should contain timestamp, got: %s", content)
}
})

t.Run("LogResponse writes to file when LogFile is set", func(t *testing.T) {
// Create a temp file for testing
logFile := "/tmp/buzz_test_log_response.txt"
defer func() {
// Clean up
os.Remove(logFile)
}()

config := &Config{
Username:  "test",
AuthToken: "token",
LogFile:   logFile,
}

// Log a response
LogResponse(config, 200, "http://example.com/api")

// Verify file exists
if _, err := os.Stat(logFile); os.IsNotExist(err) {
t.Error("Log file should exist after LogResponse")
}

// Read and verify content
data, err := os.ReadFile(logFile)
if err != nil {
t.Fatalf("Failed to read log file: %v", err)
}

content := string(data)
if !strings.Contains(content, "RESPONSE: 200 http://example.com/api") {
t.Errorf("Log content should contain response details, got: %s", content)
}
})

t.Run("Multiple log entries are appended", func(t *testing.T) {
// Create a temp file for testing
logFile := "/tmp/buzz_test_log_multiple.txt"
defer func() {
// Clean up
os.Remove(logFile)
}()

config := &Config{
Username:  "test",
AuthToken: "token",
LogFile:   logFile,
}

// Log multiple entries
LogRequest(config, "GET", "http://example.com/api/1")
LogResponse(config, 200, "http://example.com/api/1")
LogRequest(config, "POST", "http://example.com/api/2")
LogResponse(config, 201, "http://example.com/api/2")

// Read and verify content
data, err := os.ReadFile(logFile)
if err != nil {
t.Fatalf("Failed to read log file: %v", err)
}

content := string(data)
if !strings.Contains(content, "REQUEST: GET http://example.com/api/1") {
t.Error("Log should contain first request")
}
if !strings.Contains(content, "RESPONSE: 200 http://example.com/api/1") {
t.Error("Log should contain first response")
}
if !strings.Contains(content, "REQUEST: POST http://example.com/api/2") {
t.Error("Log should contain second request")
}
if !strings.Contains(content, "RESPONSE: 201 http://example.com/api/2") {
t.Error("Log should contain second response")
}
})
}

// TestConfigWithLogFile tests Config struct with LogFile field
func TestConfigWithLogFile(t *testing.T) {
t.Run("Config marshaling includes log_file", func(t *testing.T) {
config := &Config{
Username:  "myusername",
AuthToken: "myauthtoken",
LogFile:   "/path/to/log.txt",
}

// Marshal to JSON
data, err := json.Marshal(config)
if err != nil {
t.Fatalf("Failed to marshal config: %v", err)
}

// Verify JSON field name
var jsonMap map[string]interface{}
if err := json.Unmarshal(data, &jsonMap); err != nil {
t.Fatalf("Failed to unmarshal to map: %v", err)
}

if _, exists := jsonMap["log_file"]; !exists {
t.Error("JSON should have 'log_file' field")
}

// Unmarshal back and verify
var decoded Config
if err := json.Unmarshal(data, &decoded); err != nil {
t.Fatalf("Failed to unmarshal config: %v", err)
}

if decoded.LogFile != config.LogFile {
t.Errorf("LogFile = %q, want %q", decoded.LogFile, config.LogFile)
}
})

t.Run("Config unmarshaling handles missing log_file", func(t *testing.T) {
jsonData := `{"username":"myusername","auth_token":"myauthtoken"}`

var config Config
if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
t.Fatalf("Failed to unmarshal config: %v", err)
}

if config.LogFile != "" {
t.Errorf("LogFile should be empty when not in JSON, got %q", config.LogFile)
}
})
}
