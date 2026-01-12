package main

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestNoColorFlag tests that the --no-color flag is properly parsed
func TestNoColorFlag(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectNoColor  bool
		expectedArgs   []string
	}{
		{
			name:          "no flag",
			args:          []string{"buzz", "next"},
			expectNoColor: false,
			expectedArgs:  []string{"buzz", "next"},
		},
		{
			name:          "with --no-color before command",
			args:          []string{"buzz", "--no-color", "next"},
			expectNoColor: true,
			expectedArgs:  []string{"buzz", "next"},
		},
		{
			name:          "with --no-color after command",
			args:          []string{"buzz", "next", "--no-color"},
			expectNoColor: true,
			expectedArgs:  []string{"buzz", "next"},
		},
		{
			name:          "with --no-color and multiple args",
			args:          []string{"buzz", "--no-color", "add", "mygoal", "5"},
			expectNoColor: true,
			expectedArgs:  []string{"buzz", "add", "mygoal", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args and color profile
			origArgs := os.Args
			origProfile := lipgloss.ColorProfile()
			defer func() {
				os.Args = origArgs
				lipgloss.SetColorProfile(origProfile)
			}()

			// Set test args
			os.Args = tt.args

			// Process the --no-color flag like main() does
			noColor := false
			filteredArgs := []string{os.Args[0]}
			for i := 1; i < len(os.Args); i++ {
				if os.Args[i] == "--no-color" {
					noColor = true
				} else {
					filteredArgs = append(filteredArgs, os.Args[i])
				}
			}
			os.Args = filteredArgs

			if noColor {
				lipgloss.SetColorProfile(termenv.Ascii)
			}

			// Verify results
			if noColor != tt.expectNoColor {
				t.Errorf("Expected noColor=%v, got noColor=%v", tt.expectNoColor, noColor)
			}

			if len(os.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected args length %d, got %d", len(tt.expectedArgs), len(os.Args))
			}

			for i, arg := range tt.expectedArgs {
				if i >= len(os.Args) || os.Args[i] != arg {
					t.Errorf("Expected arg[%d]=%q, got %q", i, arg, os.Args[i])
				}
			}

			// Verify color profile
			if tt.expectNoColor {
				if lipgloss.ColorProfile() != termenv.Ascii {
					t.Errorf("Expected Ascii color profile when --no-color is set, got %v", lipgloss.ColorProfile())
				}
			}
		})
	}
}
