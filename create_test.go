package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestRunCreateCommandSuccess verifies the happy path: prompts are answered,
// the entered fields are forwarded to CreateGoal, and the created slug is
// reported. Goal value and rate are provided (goal date left blank), satisfying
// the "exactly 2 of 3" rule.
func TestRunCreateCommandSuccess(t *testing.T) {
	var got struct{ slug, title, goalType, gunits, goaldate, goalval, rate string }
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			got.slug, got.title, got.goalType, got.gunits = slug, title, goalType, gunits
			got.goaldate, got.goalval, got.rate = goaldate, goalval, rate
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\nhustler\npages\n\n365\n1\n")
	var stdout, stderr bytes.Buffer
	code := runCreateCommand(stdin, client, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if got.slug != "reading" || got.title != "Daily Reading" || got.goalType != "hustler" || got.gunits != "pages" {
		t.Errorf("unexpected fields forwarded: %+v", got)
	}
	if got.goaldate != "" || got.goalval != "365" || got.rate != "1" {
		t.Errorf("unexpected 2-of-3 fields: date=%q val=%q rate=%q", got.goaldate, got.goalval, got.rate)
	}
	if !strings.Contains(stdout.String(), "Successfully created goal: reading") {
		t.Errorf("missing success message, got: %s", stdout.String())
	}
}

// TestRunCreateCommandDefaultGoalType verifies that leaving the goal type blank
// falls back to the default "hustler" rather than failing validation.
func TestRunCreateCommandDefaultGoalType(t *testing.T) {
	var gotType string
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			gotType = goalType
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\n\npages\n\n365\n1\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if gotType != defaultGoalType {
		t.Errorf("expected default goal type %q, got %q", defaultGoalType, gotType)
	}
}

// TestRunCreateCommandGoalTypeByNumber verifies that selecting a goal type by
// its menu number resolves to the canonical goal_type value (here, choice "2"
// → "drinker"), and that the menu lists each type with its plain-language label.
func TestRunCreateCommandGoalTypeByNumber(t *testing.T) {
	var gotType string
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			gotType = goalType
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("nojunk\nNo Junk Food\n2\nservings\n\n0\n-1\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if gotType != "drinker" {
		t.Errorf("expected goal type %q for menu choice 2, got %q", "drinker", gotType)
	}
	// The menu should explain types, not just name them.
	if out := stdout.String(); !strings.Contains(out, "Do Less") || !strings.Contains(out, "drinker") {
		t.Errorf("menu missing label/name for a goal type, got:\n%s", out)
	}
}

// TestRunCreateCommandGoalTypeCaseInsensitiveName verifies a canonical name
// typed in a different case still resolves (EqualFold) to the canonical value.
func TestRunCreateCommandGoalTypeCaseInsensitiveName(t *testing.T) {
	var gotType string
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			gotType = goalType
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\nHUSTLER\npages\n\n365\n1\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if gotType != "hustler" {
		t.Errorf("expected goal type %q for %q, got %q", "hustler", "HUSTLER", gotType)
	}
}

// TestRunCreateCommandGoalTypePassthrough verifies the forward-compat escape
// hatch: input that is neither a valid menu number nor a known name/label —
// including an out-of-range number — is forwarded to CreateGoal verbatim, so a
// goal_type buzz doesn't yet know about still works.
func TestRunCreateCommandGoalTypePassthrough(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"unknown name", "whittler", "whittler"},
		{"out-of-range number", "99", "99"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotType string
			client := &FakeClient{
				CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
					gotType = goalType
					return &Goal{Slug: slug}, nil
				},
			}

			stdin := strings.NewReader("reading\nDaily Reading\n" + tc.input + "\npages\n\n365\n1\n")
			var stdout, stderr bytes.Buffer
			if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
				t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
			}
			if gotType != tc.want {
				t.Errorf("expected goal type %q passed through, got %q", tc.want, gotType)
			}
		})
	}
}

// TestRunCreateCommandGoalTypeByLabel verifies that a human label typed directly
// (case-insensitively) resolves to the canonical goal_type value.
func TestRunCreateCommandGoalTypeByLabel(t *testing.T) {
	var gotType string
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			gotType = goalType
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\ndo more\npages\n\n365\n1\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if gotType != "hustler" {
		t.Errorf("expected goal type %q for label %q, got %q", "hustler", "do more", gotType)
	}
}

// TestRunCreateCommandGoalDateAndValue verifies the third accepted permutation
// of the 2-of-3 rule: goal date + goal value provided, rate left blank.
func TestRunCreateCommandGoalDateAndValue(t *testing.T) {
	var got struct{ goaldate, goalval, rate string }
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			got.goaldate, got.goalval, got.rate = goaldate, goalval, rate
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\nhustler\npages\n1700000000\n365\n\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if got.goaldate != "1700000000" || got.goalval != "365" || got.rate != "" {
		t.Errorf("unexpected 2-of-3 fields: date=%q val=%q rate=%q", got.goaldate, got.goalval, got.rate)
	}
}

// TestRunCreateCommandGoalDateAndRate verifies the other accepted permutation
// of the 2-of-3 rule: goal date + rate provided, goal value left blank.
func TestRunCreateCommandGoalDateAndRate(t *testing.T) {
	var got struct{ goaldate, goalval, rate string }
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			got.goaldate, got.goalval, got.rate = goaldate, goalval, rate
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\nhustler\npages\n1700000000\n\n1\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if got.goaldate != "1700000000" || got.goalval != "" || got.rate != "1" {
		t.Errorf("unexpected 2-of-3 fields: date=%q val=%q rate=%q", got.goaldate, got.goalval, got.rate)
	}
}

// TestRunCreateCommandTrimsWhitespace verifies that surrounding whitespace and
// Windows CRLF line endings (\r\n) are stripped from each field, so piped or
// pasted input doesn't leak a stray \r or spaces into the API call.
func TestRunCreateCommandTrimsWhitespace(t *testing.T) {
	var got struct{ slug, title, gunits string }
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			got.slug, got.title, got.gunits = slug, title, gunits
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("  reading  \r\nDaily Reading\r\nhustler\r\n pages \r\n\r\n365\r\n1\r\n")
	var stdout, stderr bytes.Buffer
	if code := runCreateCommand(stdin, client, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if got.slug != "reading" || got.title != "Daily Reading" || got.gunits != "pages" {
		t.Errorf("fields not trimmed: slug=%q title=%q gunits=%q", got.slug, got.title, got.gunits)
	}
}

// TestRunCreateCommandTruncatedInput verifies graceful failure when stdin ends
// before all prompts are answered (e.g. a short pipe): the missing required
// fields fail validation, no API call is made, and the exit code is non-zero.
func TestRunCreateCommandTruncatedInput(t *testing.T) {
	called := false
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			called = true
			return &Goal{Slug: slug}, nil
		},
	}

	// Only slug and a partial (newline-less) title are supplied; the remaining
	// prompts read empty strings at EOF.
	stdin := strings.NewReader("reading\nDaily Reading")
	var stdout, stderr bytes.Buffer
	code := runCreateCommand(stdin, client, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if called {
		t.Error("CreateGoal should not be called when required input is missing")
	}
}

// TestRunCreateCommandValidationError verifies that invalid input (here, all
// three of goaldate/goalval/rate provided, violating the 2-of-3 rule) is
// rejected before any API call and surfaces a non-zero exit code.
func TestRunCreateCommandValidationError(t *testing.T) {
	called := false
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			called = true
			return &Goal{Slug: slug}, nil
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\nhustler\npages\n1700000000\n365\n1\n")
	var stdout, stderr bytes.Buffer
	code := runCreateCommand(stdin, client, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if called {
		t.Error("CreateGoal should not be called when validation fails")
	}
	if !strings.Contains(stderr.String(), "Exactly 2 out of 3") {
		t.Errorf("expected validation error on stderr, got: %s", stderr.String())
	}
}

// TestParseCreateArgsNonInteractive verifies flags are parsed into a request,
// title defaults to the slug when omitted (#335), and --deadline is threaded
// through to UpdateGoalDeadline (#332).
func TestParseCreateArgsNonInteractive(t *testing.T) {
	req, code, done := parseCreateArgs(
		[]string{"--slug=reading", "--units=pages", "--goalval=365", "--rate=1", "--deadline=-3600"},
		&bytes.Buffer{}, &bytes.Buffer{},
	)
	if done || code != 0 {
		t.Fatalf("unexpected parse result: code=%d done=%v", code, done)
	}
	if req.slug != "reading" || req.gunits != "pages" || req.goalType != defaultGoalType {
		t.Errorf("unexpected fields: %+v", req)
	}
	if !req.setDeadline || req.deadline != -3600 {
		t.Errorf("deadline not captured: set=%v val=%d", req.setDeadline, req.deadline)
	}

	var gotTitle string
	var gotDeadline int
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			gotTitle = title
			return &Goal{Slug: slug}, nil
		},
		UpdateGoalDeadlineFunc: func(goalSlug string, deadline int) (*Goal, error) {
			gotDeadline = deadline
			return &Goal{Slug: goalSlug}, nil
		},
	}

	var stdout, stderr bytes.Buffer
	if c := doCreate(req, client, &stdout, &stderr); c != 0 {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", c, stderr.String())
	}
	if gotTitle != "reading" {
		t.Errorf("title should default to slug, got %q", gotTitle)
	}
	if gotDeadline != -3600 {
		t.Errorf("deadline not forwarded, got %d", gotDeadline)
	}
}

// TestParseCreateArgsHelp verifies --help prints usage and signals done without
// error.
func TestParseCreateArgsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_, code, done := parseCreateArgs([]string{"--help"}, &stdout, &stderr)
	if !done || code != 0 {
		t.Fatalf("expected help to finish cleanly: code=%d done=%v", code, done)
	}
	if !strings.Contains(stdout.String(), "buzz create") {
		t.Errorf("usage not printed, got: %s", stdout.String())
	}
}

// TestRunCreateCommandAPIError verifies that an error from CreateGoal is
// reported and produces a non-zero exit code.
func TestRunCreateCommandAPIError(t *testing.T) {
	client := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			return nil, errors.New("boom")
		},
	}

	stdin := strings.NewReader("reading\nDaily Reading\nhustler\npages\n\n365\n1\n")
	var stdout, stderr bytes.Buffer
	code := runCreateCommand(stdin, client, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Failed to create goal") {
		t.Errorf("expected API error on stderr, got: %s", stderr.String())
	}
}
