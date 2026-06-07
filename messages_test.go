package main

import (
	"context"
	"errors"
	"testing"
)

// loadGoalsCmd: calls client.FetchGoals, sorts the result, and packs goals
// or err into goalsLoadedMsg. The fake client lets us assert both the
// success and failure branches without an HTTP server.

func TestLoadGoalsCmdSuccess(t *testing.T) {
	// SortGoals orders by losedate ascending, so the second goal should end up
	// first after sorting.
	goal1 := Goal{Slug: "later", Losedate: 200}
	goal2 := Goal{Slug: "earlier", Losedate: 100}
	fake := &FakeClient{
		FetchGoalsFunc: func() ([]Goal, error) {
			return []Goal{goal1, goal2}, nil
		},
	}

	msg, ok := loadGoalsCmd(context.Background(), fake)().(goalsLoadedMsg)
	if !ok {
		t.Fatalf("loadGoalsCmd produced %T, want goalsLoadedMsg", msg)
	}
	if msg.err != nil {
		t.Fatalf("loadGoalsCmd err = %v, want nil", msg.err)
	}
	if len(msg.goals) != 2 {
		t.Fatalf("loadGoalsCmd returned %d goals, want 2", len(msg.goals))
	}
	if msg.goals[0].Slug != "earlier" {
		t.Errorf("goals not sorted by losedate: got first=%q, want %q", msg.goals[0].Slug, "earlier")
	}
}

func TestLoadGoalsCmdError(t *testing.T) {
	wantErr := errors.New("network down")
	fake := &FakeClient{
		FetchGoalsFunc: func() ([]Goal, error) { return nil, wantErr },
	}

	msg, ok := loadGoalsCmd(context.Background(), fake)().(goalsLoadedMsg)
	if !ok {
		t.Fatalf("loadGoalsCmd produced %T, want goalsLoadedMsg", msg)
	}
	if !errors.Is(msg.err, wantErr) {
		t.Errorf("loadGoalsCmd err = %v, want %v", msg.err, wantErr)
	}
	if msg.goals != nil {
		t.Errorf("loadGoalsCmd returned goals=%v on error path, want nil", msg.goals)
	}
}

// submitDatapointCmd forwards its args verbatim to client.CreateDatapoint
// (with empty requestid) and wraps the error in datapointSubmittedMsg.

func TestSubmitDatapointCmdPassesArgs(t *testing.T) {
	var gotSlug, gotTimestamp, gotValue, gotComment, gotRequestID string
	fake := &FakeClient{
		CreateDatapointFunc: func(slug, ts, value, comment, requestID string) (*Datapoint, error) {
			gotSlug, gotTimestamp, gotValue, gotComment, gotRequestID = slug, ts, value, comment, requestID
			return &Datapoint{ID: "1"}, nil
		},
	}

	msg, ok := submitDatapointCmd(context.Background(), fake, "exercise", "1700000000", "1.5", "morning run")().(datapointSubmittedMsg)
	if !ok {
		t.Fatalf("submitDatapointCmd produced %T, want datapointSubmittedMsg", msg)
	}
	if msg.err != nil {
		t.Fatalf("submitDatapointCmd err = %v, want nil", msg.err)
	}
	if gotSlug != "exercise" || gotTimestamp != "1700000000" || gotValue != "1.5" || gotComment != "morning run" {
		t.Errorf("client called with (%q, %q, %q, %q), want (exercise, 1700000000, 1.5, morning run)",
			gotSlug, gotTimestamp, gotValue, gotComment)
	}
	if gotRequestID != "" {
		t.Errorf("submitDatapointCmd should leave requestid empty, got %q", gotRequestID)
	}
}

func TestSubmitDatapointCmdError(t *testing.T) {
	wantErr := errors.New("rate limited")
	fake := &FakeClient{
		CreateDatapointFunc: func(_, _, _, _, _ string) (*Datapoint, error) { return nil, wantErr },
	}

	msg := submitDatapointCmd(context.Background(), fake, "any", "0", "1", "")().(datapointSubmittedMsg)
	if !errors.Is(msg.err, wantErr) {
		t.Errorf("submitDatapointCmd err = %v, want %v", msg.err, wantErr)
	}
}

// loadGoalDetailsCmd is a thin wrapper around client.FetchGoalWithDatapoints
// that packs the result into goalDetailsLoadedMsg.

func TestLoadGoalDetailsCmdPassesSlug(t *testing.T) {
	want := &Goal{Slug: "x", Datapoints: []Datapoint{{ID: "1", Value: 1}}}
	var gotSlug string
	fake := &FakeClient{
		FetchGoalWithDatapointsFunc: func(slug string) (*Goal, error) {
			gotSlug = slug
			return want, nil
		},
	}

	msg := loadGoalDetailsCmd(context.Background(), fake, "x")().(goalDetailsLoadedMsg)
	if gotSlug != "x" {
		t.Errorf("client called with slug=%q, want x", gotSlug)
	}
	if msg.goal != want {
		t.Errorf("loadGoalDetailsCmd goal = %v, want %v", msg.goal, want)
	}
	if msg.err != nil {
		t.Errorf("loadGoalDetailsCmd err = %v, want nil", msg.err)
	}
}

func TestLoadGoalDetailsCmdError(t *testing.T) {
	wantErr := errors.New("goal not found")
	fake := &FakeClient{
		FetchGoalWithDatapointsFunc: func(string) (*Goal, error) { return nil, wantErr },
	}

	msg := loadGoalDetailsCmd(context.Background(), fake, "missing")().(goalDetailsLoadedMsg)
	if !errors.Is(msg.err, wantErr) {
		t.Errorf("loadGoalDetailsCmd err = %v, want %v", msg.err, wantErr)
	}
	if msg.goal != nil {
		t.Errorf("loadGoalDetailsCmd returned goal=%v on error path, want nil", msg.goal)
	}
}

// createGoalCmd forwards every positional argument to client.CreateGoal and
// wraps the result in goalCreatedMsg. Use a single happy-path test to verify
// argument plumbing — the seven-arg signature is what matters here.

func TestCreateGoalCmdPassesArgs(t *testing.T) {
	wantGoal := &Goal{Slug: "newg"}
	var gotSlug, gotTitle, gotType, gotGunits, gotGoaldate, gotGoalval, gotRate string
	fake := &FakeClient{
		CreateGoalFunc: func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
			gotSlug = slug
			gotTitle = title
			gotType = goalType
			gotGunits = gunits
			gotGoaldate = goaldate
			gotGoalval = goalval
			gotRate = rate
			return wantGoal, nil
		},
	}

	msg := createGoalCmd(context.Background(), fake, "newg", "New Goal", "hustler", "pages", "20260101", "null", "5")().(goalCreatedMsg)
	if msg.goal != wantGoal {
		t.Errorf("createGoalCmd goal = %v, want %v", msg.goal, wantGoal)
	}
	if msg.err != nil {
		t.Errorf("createGoalCmd err = %v, want nil", msg.err)
	}
	if gotSlug != "newg" || gotTitle != "New Goal" || gotType != "hustler" || gotGunits != "pages" ||
		gotGoaldate != "20260101" || gotGoalval != "null" || gotRate != "5" {
		t.Errorf("createGoalCmd passed (%q, %q, %q, %q, %q, %q, %q), want (newg, New Goal, hustler, pages, 20260101, null, 5)",
			gotSlug, gotTitle, gotType, gotGunits, gotGoaldate, gotGoalval, gotRate)
	}
}

func TestCreateGoalCmdError(t *testing.T) {
	wantErr := errors.New("slug already exists")
	fake := &FakeClient{
		CreateGoalFunc: func(_, _, _, _, _, _, _ string) (*Goal, error) { return nil, wantErr },
	}

	msg := createGoalCmd(context.Background(), fake, "dup", "", "", "", "", "", "")().(goalCreatedMsg)
	if !errors.Is(msg.err, wantErr) {
		t.Errorf("createGoalCmd err = %v, want %v", msg.err, wantErr)
	}
	if msg.goal != nil {
		t.Errorf("createGoalCmd returned goal=%v on error path, want nil", msg.goal)
	}
}
