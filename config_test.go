package main

import (
	"encoding/json"
	"os"
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

// TestLastDatapointFunctions tests the last datapoint info file operations
func TestLastDatapointFunctions(t *testing.T) {
	// Helper to clean up the last datapoint file
	cleanupLastDatapoint := func() {
		path, _ := getLastDatapointPath()
		if path != "" {
			os.Remove(path)
		}
	}

	// Clean up before and after tests
	cleanupLastDatapoint()
	defer cleanupLastDatapoint()

	t.Run("getLastDatapointPath returns valid path", func(t *testing.T) {
		path, err := getLastDatapointPath()
		if err != nil {
			t.Fatalf("getLastDatapointPath() error = %v", err)
		}
		if path == "" {
			t.Error("getLastDatapointPath() returned empty path")
		}
	})

	t.Run("LoadLastDatapoint returns error when file does not exist", func(t *testing.T) {
		cleanupLastDatapoint()

		_, err := LoadLastDatapoint()
		if err == nil {
			t.Error("LoadLastDatapoint() should error when file doesn't exist")
		}
		if !os.IsNotExist(err) {
			t.Errorf("LoadLastDatapoint() error should be IsNotExist, got: %v", err)
		}
	})

	t.Run("SaveLastDatapoint and LoadLastDatapoint work correctly", func(t *testing.T) {
		cleanupLastDatapoint()

		// Save test data
		goalSlug := "testgoal"
		datapointID := "abc123"
		beforeTime := time.Now().Unix()

		if err := SaveLastDatapoint(goalSlug, datapointID); err != nil {
			t.Fatalf("SaveLastDatapoint() error = %v", err)
		}

		afterTime := time.Now().Unix()

		// Load and verify
		info, err := LoadLastDatapoint()
		if err != nil {
			t.Fatalf("LoadLastDatapoint() error = %v", err)
		}

		if info.GoalSlug != goalSlug {
			t.Errorf("GoalSlug = %q, want %q", info.GoalSlug, goalSlug)
		}
		if info.DatapointID != datapointID {
			t.Errorf("DatapointID = %q, want %q", info.DatapointID, datapointID)
		}
		if info.Timestamp < beforeTime || info.Timestamp > afterTime {
			t.Errorf("Timestamp = %d, want between %d and %d", info.Timestamp, beforeTime, afterTime)
		}
	})

	t.Run("SaveLastDatapoint overwrites existing data", func(t *testing.T) {
		cleanupLastDatapoint()

		// Save first datapoint
		if err := SaveLastDatapoint("goal1", "id1"); err != nil {
			t.Fatalf("First SaveLastDatapoint() error = %v", err)
		}

		// Wait to ensure different timestamp
		time.Sleep(1 * time.Second)

		// Save second datapoint (should overwrite)
		if err := SaveLastDatapoint("goal2", "id2"); err != nil {
			t.Fatalf("Second SaveLastDatapoint() error = %v", err)
		}

		// Load and verify we got the second one
		info, err := LoadLastDatapoint()
		if err != nil {
			t.Fatalf("LoadLastDatapoint() error = %v", err)
		}

		if info.GoalSlug != "goal2" {
			t.Errorf("GoalSlug = %q, want %q (should be overwritten)", info.GoalSlug, "goal2")
		}
		if info.DatapointID != "id2" {
			t.Errorf("DatapointID = %q, want %q (should be overwritten)", info.DatapointID, "id2")
		}
	})

	t.Run("LastDatapointInfo JSON marshaling", func(t *testing.T) {
		info := &LastDatapointInfo{
			GoalSlug:    "mygoal",
			DatapointID: "xyz789",
			Timestamp:   1234567890,
		}

		// Marshal to JSON
		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Failed to marshal LastDatapointInfo: %v", err)
		}

		// Unmarshal back
		var decoded LastDatapointInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal LastDatapointInfo: %v", err)
		}

		// Verify fields
		if decoded.GoalSlug != info.GoalSlug {
			t.Errorf("GoalSlug = %q, want %q", decoded.GoalSlug, info.GoalSlug)
		}
		if decoded.DatapointID != info.DatapointID {
			t.Errorf("DatapointID = %q, want %q", decoded.DatapointID, info.DatapointID)
		}
		if decoded.Timestamp != info.Timestamp {
			t.Errorf("Timestamp = %d, want %d", decoded.Timestamp, info.Timestamp)
		}
	})
}
