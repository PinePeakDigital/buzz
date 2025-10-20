package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
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

// TestPrintVersionFormat tests that printVersion function produces correct output
func TestPrintVersionFormat(t *testing.T) {
	originalVersion := version
	defer func() { version = originalVersion }()
	
	testVersions := []string{
		"v0.21.0",
		"v1.0.0-beta",
		"dev",
		"v1.2.3-rc1",
	}
	
	for _, testVersion := range testVersions {
		t.Run(testVersion, func(t *testing.T) {
			version = testVersion
			
			// Capture stdout
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			printVersion()
			
			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)
			
			expected := fmt.Sprintf("buzz version %s\n", testVersion)
			if buf.String() != expected {
				t.Errorf("expected %q, got %q", expected, buf.String())
			}
		})
	}
}
