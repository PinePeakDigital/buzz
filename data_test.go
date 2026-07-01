package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunDataCommand(t *testing.T) {
	twoPoints := func(string) (*Goal, error) {
		return &Goal{Datapoints: []Datapoint{
			// Deliberately out of order to exercise the chronological sort. Both
			// rows carry comments and have different value widths ("3" vs "12.5")
			// so the "%-*s" column padding is visible: "3" pads to width 4.
			{Timestamp: 200, Daystamp: "20240102", Value: 12.5, Comment: "later"},
			{Timestamp: 100, Daystamp: "20240101", Value: 3, Comment: "first"},
		}}, nil
	}
	// A datapoint with no daystamp forces the time.Unix(...).UTC() fallback.
	// 1704153600 = 2024-01-02 00:00:00 UTC; a non-UTC render could shift the day.
	noDaystamp := func(string) (*Goal, error) {
		return &Goal{Datapoints: []Datapoint{
			{Timestamp: 1704153600, Daystamp: "", Value: 7},
		}}, nil
	}
	tests := []struct {
		name             string
		args             []string
		fn               func(string) (*Goal, error)
		wantCode         int
		wantOut, wantErr string
	}{
		{"missing arg", nil, nil, 1, "", "Missing required argument"},
		{"too many args", []string{"a", "b"}, nil, 1, "", "Too many arguments"},
		{"api error", []string{"g"}, func(string) (*Goal, error) { return nil, errors.New("boom") }, 1, "", "boom"},
		{"no datapoints", []string{"g"}, func(string) (*Goal, error) { return &Goal{}, nil }, 0, "No datapoints found for goal: g", ""},
		{"lists sorted and aligned", []string{"g"}, twoPoints, 0, "2024-01-01   3      first\n2024-01-02   12.5   later\n", ""},
		{"explicit --asc matches default", []string{"--asc", "g"}, twoPoints, 0, "2024-01-01   3      first\n2024-01-02   12.5   later\n", ""},
		{"--desc reverses to newest-first", []string{"g", "--desc"}, twoPoints, 0, "2024-01-02   12.5   later\n2024-01-01   3      first\n", ""},
		{"--asc and --desc are mutually exclusive", []string{"--asc", "--desc", "g"}, nil, 1, "", "mutually exclusive"},
		{"unknown flag", []string{"--nope", "g"}, nil, 2, "", "Error parsing flags"},
		{"daystamp fallback to utc timestamp", []string{"g"}, noDaystamp, 0, "2024-01-02   7\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := runDataCommand(tt.args, &FakeClient{FetchGoalWithDatapointsFunc: tt.fn}, &out, &errb)
			checkResult(t, code, out.String(), errb.String(), tt.wantCode, tt.wantOut, tt.wantErr)
		})
	}
}
