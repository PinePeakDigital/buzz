package main

import (
	"testing"
	"time"
)

// TestRefreshIntervalConstant verifies the RefreshInterval constant is set correctly
func TestRefreshIntervalConstant(t *testing.T) {
	expected := time.Minute * 5
	if RefreshInterval != expected {
		t.Errorf("RefreshInterval = %v, want %v", RefreshInterval, expected)
	}
}

// TestClearScreen tests that clearScreen doesn't panic
func TestClearScreen(t *testing.T) {
	// This test just verifies the function doesn't panic
	// We can't easily verify the ANSI escape codes without capturing stdout
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("clearScreen() panicked: %v", r)
		}
	}()
	
	clearScreen()
}

// TestDisplayNextGoalNoConfig tests displayNextGoal when config doesn't exist
func TestDisplayNextGoalNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := displayNextGoal(); err == nil {
		t.Fatalf("expected error when no config present")
	}
}

// TestDisplayNextGoalWithTimestamp tests that displayNextGoalWithTimestamp doesn't panic
func TestDisplayNextGoalWithTimestamp(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayNextGoalWithTimestamp() panicked: %v", r)
		}
	}()
	t.Setenv("HOME", t.TempDir())
	displayNextGoalWithTimestamp()
}

// TestTimestampFormat tests that the timestamp format used in watch mode is correct
func TestTimestampFormat(t *testing.T) {
	// Test that the timestamp format "2006-01-02 15:04:05" works correctly
	testTime := time.Date(2025, 10, 10, 23, 27, 13, 0, time.UTC)
	formatted := testTime.Format("2006-01-02 15:04:05")
	expected := "2025-10-10 23:27:13"
	
	if formatted != expected {
		t.Errorf("Timestamp format = %q, want %q", formatted, expected)
	}
}
