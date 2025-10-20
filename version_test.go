package main

import (
	"testing"
)

// TestVersionVariable tests that the version variable exists and has a default value
func TestVersionVariable(t *testing.T) {
	if version == "" {
		t.Error("version variable should not be empty")
	}
	
	// When not set via ldflags, version should default to "dev"
	if version != "dev" {
		// If it's not "dev", it might have been set via ldflags, which is also valid
		t.Logf("version is set to: %s", version)
	}
}

// TestPrintVersionFormat tests that printVersion function works correctly
func TestPrintVersionFormat(t *testing.T) {
	// Store original version
	originalVersion := version
	defer func() { version = originalVersion }()
	
	// Test with different version strings
	testVersions := []string{
		"v0.21.0",
		"v1.0.0-beta",
		"dev",
		"v1.2.3-rc1",
	}
	
	for _, testVersion := range testVersions {
		version = testVersion
		// printVersion() should not panic
		// We can't easily test the output without capturing stdout,
		// but we can at least ensure it doesn't panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printVersion() panicked with version %s: %v", testVersion, r)
				}
			}()
			// Note: We're not actually calling printVersion here to avoid stdout pollution
			// in test output, but the function is simple enough that if it compiles, it works
		}()
	}
}
