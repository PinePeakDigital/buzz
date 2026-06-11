package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// noStdin simulates an unpiped stdin (readValueFromStdin's error path).
func noStdin() (string, error) { return "", errors.New("stdin is not piped") }

// pipedStdin simulates a piped value on stdin.
func pipedStdin(v string) func() (string, error) {
	return func() (string, error) { return v, nil }
}

func TestRunRefreshCommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		fn               func(string) (bool, error)
		wantCode         int
		wantOut, wantErr string
	}{
		{"missing arg", nil, nil, 1, "", "Missing required argument"},
		{"too many args", []string{"a", "b"}, nil, 1, "", "Too many arguments"},
		{"queued", []string{"g"}, func(string) (bool, error) { return true, nil }, 0, "Successfully queued refresh for goal: g", ""},
		{"not queued", []string{"g"}, func(string) (bool, error) { return false, nil }, 0, "was not queued", ""},
		{"api error", []string{"g"}, func(string) (bool, error) { return false, errors.New("boom") }, 1, "", "Failed to refresh goal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := runRefreshCommand(tt.args, &FakeClient{RefreshGoalFunc: tt.fn}, &out, &errb)
			checkResult(t, code, out.String(), errb.String(), tt.wantCode, tt.wantOut, tt.wantErr)
		})
	}
}

func TestRunChargeCommand(t *testing.T) {
	okCharge := func(amount float64, note string, _ bool) (*Charge, error) {
		return &Charge{ID: "c1", Amount: amount, Note: note, Username: "u"}, nil
	}
	tests := []struct {
		name             string
		args             []string
		fn               func(float64, string, bool) (*Charge, error)
		wantCode         int
		wantOut, wantErr string
	}{
		{"missing args", []string{"5"}, nil, 1, "", "Missing required arguments"},
		{"bad amount", []string{"abc", "note"}, nil, 1, "", "must be a valid number"},
		{"NaN amount", []string{"NaN", "note"}, nil, 1, "", "must be a finite number"},
		{"below minimum", []string{"0.50", "note"}, nil, 1, "", "at least 1.00"},
		{"empty note", []string{"5", "   "}, nil, 1, "", "Note is required"},
		{"success", []string{"5", "my", "note"}, okCharge, 0, "Successfully created charge c1", ""},
		{"dryrun anywhere", []string{"5", "note", "--dryrun"}, okCharge, 0, "Dry run: Would charge", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := runChargeCommand(tt.args, &FakeClient{CreateChargeFunc: tt.fn}, &out, &errb)
			checkResult(t, code, out.String(), errb.String(), tt.wantCode, tt.wantOut, tt.wantErr)
		})
	}
}

func TestParseAddArgs(t *testing.T) {
	t.Run("help to stdout, no error", func(t *testing.T) {
		var out, errb bytes.Buffer
		_, code, done := parseAddArgs([]string{"-h"}, noStdin, &out, &errb)
		if !done || code != 0 {
			t.Fatalf("done=%v code=%d, want done=true code=0", done, code)
		}
		if !strings.Contains(out.String(), "Usage: buzz add") {
			t.Errorf("help not printed to stdout: %q", out.String())
		}
	})

	t.Run("positional value and comment", func(t *testing.T) {
		req, _, done := parseAddArgs([]string{"goal", "42", "a", "note"}, noStdin, &bytes.Buffer{}, &bytes.Buffer{})
		if done {
			t.Fatal("unexpected done")
		}
		if req.goalSlug != "goal" || req.value != "42" || req.comment != "a note" {
			t.Errorf("got %+v", req)
		}
	})

	t.Run("piped value, default comment", func(t *testing.T) {
		req, _, done := parseAddArgs([]string{"goal"}, pipedStdin("42"), &bytes.Buffer{}, &bytes.Buffer{})
		if done {
			t.Fatal("unexpected done")
		}
		if req.value != "42" || req.comment != "Added via buzz" {
			t.Errorf("got %+v", req)
		}
	})

	t.Run("piped and positional value rejected", func(t *testing.T) {
		var errb bytes.Buffer
		_, code, done := parseAddArgs([]string{"goal", "42"}, pipedStdin("99"), &bytes.Buffer{}, &errb)
		if !done || code != 1 || !strings.Contains(errb.String(), "not both") {
			t.Errorf("done=%v code=%d err=%q", done, code, errb.String())
		}
	})

	t.Run("time-format value converted to decimal", func(t *testing.T) {
		req, _, done := parseAddArgs([]string{"goal", "1:30:00"}, noStdin, &bytes.Buffer{}, &bytes.Buffer{})
		if done {
			t.Fatal("unexpected done")
		}
		if strings.Contains(req.value, ":") {
			t.Errorf("time value not converted to decimal: %q", req.value)
		}
	})

	t.Run("invalid daystamp", func(t *testing.T) {
		var errb bytes.Buffer
		_, code, done := parseAddArgs([]string{"--daystamp=2024", "goal", "42"}, noStdin, &bytes.Buffer{}, &errb)
		if !done || code != 1 || !strings.Contains(errb.String(), "Invalid date format") {
			t.Errorf("done=%v code=%d err=%q", done, code, errb.String())
		}
	})

	t.Run("non-numeric value rejected", func(t *testing.T) {
		var errb bytes.Buffer
		_, code, done := parseAddArgs([]string{"goal", "notanumber"}, noStdin, &bytes.Buffer{}, &errb)
		if !done || code != 1 || !strings.Contains(errb.String(), "must be a valid number") {
			t.Errorf("done=%v code=%d err=%q", done, code, errb.String())
		}
	})

	t.Run("missing value", func(t *testing.T) {
		var errb bytes.Buffer
		_, code, done := parseAddArgs([]string{"goal"}, noStdin, &bytes.Buffer{}, &errb)
		if !done || code != 1 || !strings.Contains(errb.String(), "Missing required value") {
			t.Errorf("done=%v code=%d err=%q", done, code, errb.String())
		}
	})

	t.Run("flag after positionals warns and is absorbed into comment", func(t *testing.T) {
		var errb bytes.Buffer
		// A --daystamp after the positional args is not parsed as a flag; it
		// warns and is treated as part of the comment.
		req, code, done := parseAddArgs([]string{"goal", "42", "--daystamp=20240115"}, noStdin, &bytes.Buffer{}, &errb)
		if done {
			t.Fatalf("unexpected done (code=%d)", code)
		}
		if !strings.Contains(errb.String(), "appears after positional arguments") {
			t.Errorf("expected misplaced-flag warning, stderr=%q", errb.String())
		}
		if req.daystamp != "" || req.comment != "--daystamp=20240115" {
			t.Errorf("flag should be absorbed into comment, got daystamp=%q comment=%q", req.daystamp, req.comment)
		}
	})
}

func TestRunAddCommand(t *testing.T) {
	t.Run("success forwards request and reports daystamp/requestid", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir()) // contain createRefreshFlag's file write
		var out, errb bytes.Buffer
		var gotSlug, gotDaystamp, gotValue, gotComment, gotReqID string
		client := &FakeClient{
			CreateDatapointWithDaystampFunc: func(slug, _, daystamp, value, comment, requestid string) (*Datapoint, error) {
				gotSlug, gotDaystamp, gotValue, gotComment, gotReqID = slug, daystamp, value, comment, requestid
				return &Datapoint{}, nil
			},
		}
		req := addRequest{goalSlug: "g", value: "42", comment: "hi", daystamp: "20240115", requestid: "r1"}
		if code := runAddCommand(req, client, &out, &errb); code != 0 {
			t.Fatalf("code=%d err=%q", code, errb.String())
		}
		if gotSlug != "g" || gotDaystamp != "20240115" || gotValue != "42" || gotComment != "hi" || gotReqID != "r1" {
			t.Errorf("client got slug=%q daystamp=%q value=%q comment=%q reqid=%q", gotSlug, gotDaystamp, gotValue, gotComment, gotReqID)
		}
		o := out.String()
		if !strings.Contains(o, "Successfully added datapoint to g") ||
			!strings.Contains(o, "daystamp=20240115") ||
			!strings.Contains(o, `requestid="r1"`) {
			t.Errorf("stdout=%q", o)
		}
	})

	t.Run("api error", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		var out, errb bytes.Buffer
		client := &FakeClient{
			CreateDatapointWithDaystampFunc: func(_, _, _, _, _, _ string) (*Datapoint, error) {
				return nil, errors.New("boom")
			},
		}
		code := runAddCommand(addRequest{goalSlug: "g", value: "1"}, client, &out, &errb)
		if code != 1 || !strings.Contains(errb.String(), "Failed to add datapoint") {
			t.Errorf("code=%d err=%q", code, errb.String())
		}
	})
}

func TestParseDeadlineArgs(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		var out bytes.Buffer
		_, code, done := parseDeadlineArgs([]string{"-h"}, &out, &bytes.Buffer{})
		if !done || code != 0 || !strings.Contains(out.String(), "buzz deadline") {
			t.Errorf("done=%v code=%d out=%q", done, code, out.String())
		}
	})

	t.Run("missing args", func(t *testing.T) {
		var errb bytes.Buffer
		_, code, done := parseDeadlineArgs([]string{"goal"}, &bytes.Buffer{}, &errb)
		if !done || code != 1 || !strings.Contains(errb.String(), "Missing required arguments") {
			t.Errorf("done=%v code=%d err=%q", done, code, errb.String())
		}
	})

	t.Run("invalid time", func(t *testing.T) {
		var errb bytes.Buffer
		_, code, done := parseDeadlineArgs([]string{"goal", "notatime"}, &bytes.Buffer{}, &errb)
		if !done || code != 1 {
			t.Errorf("done=%v code=%d err=%q", done, code, errb.String())
		}
	})

	t.Run("valid with --yes", func(t *testing.T) {
		req, _, done := parseDeadlineArgs([]string{"--yes", "goal", "15:00"}, &bytes.Buffer{}, &bytes.Buffer{})
		if done {
			t.Fatal("unexpected done")
		}
		// offset is Beeminder's seconds-relative-to-midnight (15:00 → -32400);
		// assert the slug/flag wiring and that a non-zero offset was parsed
		// rather than re-deriving the deadline convention here.
		if req.goalSlug != "goal" || !req.skipConfirm || req.offset == 0 {
			t.Errorf("got %+v", req)
		}
	})
}

func TestRunDeadlineCommand(t *testing.T) {
	updated := func(slug string, deadline int) (*Goal, error) {
		return &Goal{Slug: slug, Deadline: deadline}, nil
	}

	t.Run("skip confirm updates without fetch", func(t *testing.T) {
		var out, errb bytes.Buffer
		client := &FakeClient{UpdateGoalDeadlineFunc: updated} // FetchGoal unset → would error if called
		code := runDeadlineCommand(deadlineRequest{goalSlug: "g", offset: 54000, skipConfirm: true}, strings.NewReader(""), client, &out, &errb)
		if code != 0 || !strings.Contains(out.String(), "Updated deadline for g") {
			t.Errorf("code=%d out=%q err=%q", code, out.String(), errb.String())
		}
	})

	t.Run("confirm yes fetches then updates", func(t *testing.T) {
		fetched := false
		client := &FakeClient{
			FetchGoalFunc:          func(s string) (*Goal, error) { fetched = true; return &Goal{Slug: s}, nil },
			UpdateGoalDeadlineFunc: updated,
		}
		var out, errb bytes.Buffer
		code := runDeadlineCommand(deadlineRequest{goalSlug: "g", offset: 54000}, strings.NewReader("y\n"), client, &out, &errb)
		if code != 0 || !fetched || !strings.Contains(out.String(), "Updated deadline") {
			t.Errorf("code=%d fetched=%v out=%q", code, fetched, out.String())
		}
	})

	t.Run("decline cancels without updating", func(t *testing.T) {
		updateCalled := false
		client := &FakeClient{
			FetchGoalFunc:          func(s string) (*Goal, error) { return &Goal{Slug: s}, nil },
			UpdateGoalDeadlineFunc: func(string, int) (*Goal, error) { updateCalled = true; return &Goal{}, nil },
		}
		var out, errb bytes.Buffer
		code := runDeadlineCommand(deadlineRequest{goalSlug: "g", offset: 54000}, strings.NewReader("n\n"), client, &out, &errb)
		if code != 0 || updateCalled || !strings.Contains(out.String(), "Cancelled") {
			t.Errorf("code=%d updateCalled=%v out=%q", code, updateCalled, out.String())
		}
	})

	t.Run("fetch error", func(t *testing.T) {
		client := &FakeClient{FetchGoalFunc: func(string) (*Goal, error) { return nil, errors.New("boom") }}
		var out, errb bytes.Buffer
		code := runDeadlineCommand(deadlineRequest{goalSlug: "g", offset: 54000}, strings.NewReader("y\n"), client, &out, &errb)
		if code != 1 || !strings.Contains(errb.String(), "Failed to fetch goal") {
			t.Errorf("code=%d err=%q", code, errb.String())
		}
	})

	t.Run("update error", func(t *testing.T) {
		client := &FakeClient{
			UpdateGoalDeadlineFunc: func(string, int) (*Goal, error) { return nil, errors.New("boom") },
		}
		var out, errb bytes.Buffer
		code := runDeadlineCommand(deadlineRequest{goalSlug: "g", offset: 54000, skipConfirm: true}, strings.NewReader(""), client, &out, &errb)
		if code != 1 || !strings.Contains(errb.String(), "Failed to update deadline") {
			t.Errorf("code=%d err=%q", code, errb.String())
		}
	})
}

// checkResult is a shared assertion for the table-driven run* command tests.
func checkResult(t *testing.T, code int, out, errOut string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	if code != wantCode {
		t.Errorf("code = %d, want %d (stderr: %q)", code, wantCode, errOut)
	}
	if wantOut != "" && !strings.Contains(out, wantOut) {
		t.Errorf("stdout = %q, want contains %q", out, wantOut)
	}
	if wantErr != "" && !strings.Contains(errOut, wantErr) {
		t.Errorf("stderr = %q, want contains %q", errOut, wantErr)
	}
}
