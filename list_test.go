package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// TestFormatListRate tests the formatListRate function
func TestFormatListRate(t *testing.T) {
	tests := []struct {
		name     string
		rate     *float64
		runits   string
		expected string
	}{
		{
			name:     "nil rate",
			rate:     nil,
			runits:   "d",
			expected: "-",
		},
		{
			name:     "zero rate",
			rate:     float64Ptr(0.0),
			runits:   "d",
			expected: "0/d",
		},
		{
			name:     "integer rate",
			rate:     float64Ptr(1.0),
			runits:   "d",
			expected: "1/d",
		},
		{
			name:     "decimal rate",
			rate:     float64Ptr(0.5),
			runits:   "w",
			expected: "0.5/w",
		},
		{
			name:     "decimal rate with multiple digits",
			rate:     float64Ptr(2.75),
			runits:   "d",
			expected: "2.75/d",
		},
		{
			name:     "large integer rate",
			rate:     float64Ptr(100.0),
			runits:   "m",
			expected: "100/m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatListRate(tt.rate, tt.runits)
			if result != tt.expected {
				t.Errorf("formatListRate(%v, %q) = %q, want %q", tt.rate, tt.runits, result, tt.expected)
			}
		})
	}
}

// float64Ptr is a helper function to create a pointer to a float64
func float64Ptr(f float64) *float64 {
	return &f
}

// TestParseListArgs covers the `buzz list` argument-parsing branches: the
// default and --archived success paths, help, an unknown flag, and an
// unexpected positional argument.
func TestParseListArgs(t *testing.T) {
	t.Run("no args defaults to active", func(t *testing.T) {
		var out, errOut bytes.Buffer
		archived, code, done := parseListArgs(nil, &out, &errOut)
		if archived || code != 0 || done {
			t.Fatalf("got archived=%v code=%d done=%v, want false/0/false", archived, code, done)
		}
	})

	t.Run("--archived selects archived", func(t *testing.T) {
		var out, errOut bytes.Buffer
		archived, code, done := parseListArgs([]string{"--archived"}, &out, &errOut)
		if !archived || code != 0 || done {
			t.Fatalf("got archived=%v code=%d done=%v, want true/0/false", archived, code, done)
		}
	})

	t.Run("help prints usage and stops cleanly", func(t *testing.T) {
		var out, errOut bytes.Buffer
		archived, code, done := parseListArgs([]string{"-h"}, &out, &errOut)
		if archived || code != 0 || !done {
			t.Fatalf("got archived=%v code=%d done=%v, want false/0/true", archived, code, done)
		}
		if !strings.Contains(out.String(), "Usage: buzz list") {
			t.Errorf("expected usage on stdout, got: %q", out.String())
		}
		// Help goes to stdout only; flag's built-in usage must not leak to stderr.
		if errOut.Len() != 0 {
			t.Errorf("expected nothing on stderr for help, got: %q", errOut.String())
		}
	})

	t.Run("unknown flag errors with exit code 2", func(t *testing.T) {
		var out, errOut bytes.Buffer
		_, code, done := parseListArgs([]string{"--bogus"}, &out, &errOut)
		if code != 2 || !done {
			t.Fatalf("got code=%d done=%v, want 2/true", code, done)
		}
		if !strings.Contains(errOut.String(), "Usage: buzz list") {
			t.Errorf("expected usage on stderr, got: %q", errOut.String())
		}
		// Only our explicit message should print — flag's auto-output is
		// suppressed, so usage appears exactly once and nothing leaks to stdout.
		if n := strings.Count(errOut.String(), "Usage:"); n != 1 {
			t.Errorf("expected usage printed once, got %d times: %q", n, errOut.String())
		}
		if out.Len() != 0 {
			t.Errorf("expected nothing on stdout for parse error, got: %q", out.String())
		}
	})

	t.Run("unexpected positional arg errors with exit code 2", func(t *testing.T) {
		var out, errOut bytes.Buffer
		_, code, done := parseListArgs([]string{"extra"}, &out, &errOut)
		if code != 2 || !done {
			t.Fatalf("got code=%d done=%v, want 2/true", code, done)
		}
		if !strings.Contains(errOut.String(), "Unknown arguments") {
			t.Errorf("expected unknown-arguments message on stderr, got: %q", errOut.String())
		}
	})
}

// TestRunListCommand exercises the testable core of `buzz list`, covering the
// active/archived split, sorting, the empty case, and fetch errors.
func TestRunListCommand(t *testing.T) {
	t.Run("lists active goals sorted by slug", func(t *testing.T) {
		client := &FakeClient{
			FetchGoalsFunc: func() ([]Goal, error) {
				return []Goal{
					{Slug: "zebra", Title: "Z Goal", Gunits: "reps", Rate: float64Ptr(2), Runits: "d", Pledge: 5},
					{Slug: "apple", Title: "A Goal", Gunits: "pages", Pledge: 0},
				}, nil
			},
			// Leaving FetchArchivedGoalsFunc nil ensures the active path never
			// touches the archived endpoint.
		}

		var out, errOut bytes.Buffer
		code := runListCommand(context.Background(), client, false, &out, &errOut)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		got := out.String()
		if !strings.Contains(got, "Total goals: 2") {
			t.Errorf("expected active-goals header, got:\n%s", got)
		}
		if i, j := strings.Index(got, "apple"), strings.Index(got, "zebra"); i == -1 || j == -1 || i > j {
			t.Errorf("expected goals sorted by slug (apple before zebra), got:\n%s", got)
		}
		if errOut.Len() != 0 {
			t.Errorf("expected nothing on stderr, got: %q", errOut.String())
		}
	})

	t.Run("lists archived goals", func(t *testing.T) {
		client := &FakeClient{
			FetchArchivedGoalsFunc: func() ([]Goal, error) {
				return []Goal{{Slug: "olddiet", Title: "Old Diet", Gunits: "lbs", Pledge: 30}}, nil
			},
		}

		var out, errOut bytes.Buffer
		code := runListCommand(context.Background(), client, true, &out, &errOut)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		got := out.String()
		if !strings.Contains(got, "Total archived goals: 1") {
			t.Errorf("expected archived-goals header, got:\n%s", got)
		}
		if !strings.Contains(got, "olddiet") {
			t.Errorf("expected archived goal slug in output, got:\n%s", got)
		}
		if errOut.Len() != 0 {
			t.Errorf("expected nothing on stderr, got: %q", errOut.String())
		}
	})

	t.Run("empty active goals", func(t *testing.T) {
		client := &FakeClient{FetchGoalsFunc: func() ([]Goal, error) { return nil, nil }}

		var out, errOut bytes.Buffer
		code := runListCommand(context.Background(), client, false, &out, &errOut)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		if got := out.String(); !strings.Contains(got, "No goals found.") {
			t.Errorf("expected empty-active message, got:\n%s", got)
		}
	})

	t.Run("empty archived goals", func(t *testing.T) {
		client := &FakeClient{FetchArchivedGoalsFunc: func() ([]Goal, error) { return nil, nil }}

		var out, errOut bytes.Buffer
		code := runListCommand(context.Background(), client, true, &out, &errOut)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		if got := out.String(); !strings.Contains(got, "No archived goals found.") {
			t.Errorf("expected empty-archived message, got:\n%s", got)
		}
	})

	t.Run("fetch error returns exit code 1", func(t *testing.T) {
		client := &FakeClient{
			FetchArchivedGoalsFunc: func() ([]Goal, error) {
				return nil, errors.New("boom")
			},
		}

		var out, errOut bytes.Buffer
		code := runListCommand(context.Background(), client, true, &out, &errOut)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		// The fetch error goes to stderr, keeping stdout clean for piping.
		if got := errOut.String(); !strings.Contains(got, "Failed to fetch archived goals") {
			t.Errorf("expected fetch-error message on stderr, got:\n%s", got)
		}
		if out.Len() != 0 {
			t.Errorf("expected nothing on stdout for fetch error, got: %q", out.String())
		}
	})
}

// TestGetDisplayUnits tests the getDisplayUnits function
func TestGetDisplayUnits(t *testing.T) {
	tests := []struct {
		name     string
		gunits   string
		expected string
	}{
		{
			name:     "empty string",
			gunits:   "",
			expected: "-",
		},
		{
			name:     "with units",
			gunits:   "hours",
			expected: "hours",
		},
		{
			name:     "single character",
			gunits:   "x",
			expected: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDisplayUnits(tt.gunits)
			if result != tt.expected {
				t.Errorf("getDisplayUnits(%q) = %q, want %q", tt.gunits, result, tt.expected)
			}
		})
	}
}
