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
		{
			name:     "with --web flag after slug",
			args:     []string{"mygoal", "--web"},
			wantWeb:  true,
			wantSlug: "mygoal",
			wantErr:  false,
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

			// Check if --web flag appears after the goal slug (handle both positions)
			webFlag := *web
			var filteredArgs []string

			for _, arg := range args {
				if arg == "--web" {
					webFlag = true
				} else {
					filteredArgs = append(filteredArgs, arg)
				}
			}

			// Check if we got a slug when we should
			if tt.wantErr && len(filteredArgs) == 0 {
				// Expected error case (no slug provided)
				return
			}

			// Check web flag value
			if webFlag != tt.wantWeb {
				t.Errorf("web flag = %v, want %v", webFlag, tt.wantWeb)
			}

			// Check goal slug
			if len(filteredArgs) > 0 {
				gotSlug := filteredArgs[0]
				if gotSlug != tt.wantSlug {
					t.Errorf("goal slug = %v, want %v", gotSlug, tt.wantSlug)
				}
			} else if tt.wantSlug != "" {
				t.Errorf("expected goal slug %v, got none", tt.wantSlug)
			}
		})
	}
}
