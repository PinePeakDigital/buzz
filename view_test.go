package main

import (
	"flag"
	"testing"
)

// TestViewCommandFlagParsing tests that the --web flag can be parsed correctly
func TestViewCommandFlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantWeb  bool
		wantSlug string
		wantErr  bool
	}{
		{
			name:     "no flags",
			args:     []string{"mygoal"},
			wantWeb:  false,
			wantSlug: "mygoal",
			wantErr:  false,
		},
		{
			name:     "with --web flag before slug",
			args:     []string{"--web", "mygoal"},
			wantWeb:  true,
			wantSlug: "mygoal",
			wantErr:  false,
		},
		{
			name:     "no goal slug provided",
			args:     []string{},
			wantWeb:  false,
			wantSlug: "",
			wantErr:  true,
		},
		{
			name:     "with --web flag and no slug",
			args:     []string{"--web"},
			wantWeb:  true,
			wantSlug: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new flag set for each test to avoid pollution
			viewFlags := flag.NewFlagSet("view", flag.ContinueOnError)
			web := viewFlags.Bool("web", false, "Open the goal in the browser")

			// Parse the arguments
			err := viewFlags.Parse(tt.args)

			// Check for parsing errors
			if err != nil {
				if !tt.wantErr {
					t.Errorf("unexpected parse error: %v", err)
				}
				return
			}

			// Get remaining args (goal slug)
			args := viewFlags.Args()

			// Check if we got a slug when we should
			if tt.wantErr && len(args) == 0 {
				// Expected error case (no slug provided)
				return
			}

			// Check web flag value
			if *web != tt.wantWeb {
				t.Errorf("web flag = %v, want %v", *web, tt.wantWeb)
			}

			// Check goal slug
			if len(args) > 0 {
				gotSlug := args[0]
				if gotSlug != tt.wantSlug {
					t.Errorf("goal slug = %v, want %v", gotSlug, tt.wantSlug)
				}
			} else if tt.wantSlug != "" {
				t.Errorf("expected goal slug %v, got none", tt.wantSlug)
			}
		})
	}
}
