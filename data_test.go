package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunDataCommand(t *testing.T) {
	twoPoints := func(string) (*Goal, error) {
		return &Goal{Datapoints: []Datapoint{
			// Deliberately out of order to exercise the chronological sort.
			{Timestamp: 200, Daystamp: "20240102", Value: 12.5, Comment: "later"},
			{Timestamp: 100, Daystamp: "20240101", Value: 3, Comment: ""},
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
		{"lists sorted", []string{"g"}, twoPoints, 0, "2024-01-01   3\n2024-01-02   12.5   later\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := runDataCommand(tt.args, &FakeClient{FetchGoalWithDatapointsFunc: tt.fn}, &out, &errb)
			checkResult(t, code, out.String(), errb.String(), tt.wantCode, tt.wantOut, tt.wantErr)
		})
	}
}
