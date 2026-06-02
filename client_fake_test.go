package main

import (
	"context"
	"encoding/json"
	"errors"
)

// FakeClient is a test double for the Client interface. Each API method is
// backed by an optional function field — set the ones a given test needs and
// leave the rest unset; unset methods return errFakeNotConfigured so a test
// that exercises an unexpected path fails loudly rather than silently
// returning a zero value.
//
// Lives in a _test.go file so it never ships in the production binary; both
// command tests and Bubble Tea handler tests can use it because everything is
// in `package main`.
//
// The *Func signatures intentionally omit context.Context — every Client
// method takes one, but the fake drops it before invoking *Func, so the
// callback can't observe it. Tests that need to assert
// cancellation/deadline propagation should drive HTTPClient against
// httptest (see TestHTTPClientCancellation) or extend this fake with an
// explicit context-capture field when the need arises.
type FakeClient struct {
	FetchGoalsFunc                  func() ([]Goal, error)
	FetchGoalFunc                   func(goalSlug string) (*Goal, error)
	FetchGoalWithDatapointsFunc     func(goalSlug string) (*Goal, error)
	FetchGoalRawJSONFunc            func(goalSlug string, includeDatapoints bool) (json.RawMessage, error)
	GetLastDatapointValueFunc       func(goalSlug string) (float64, error)
	CreateDatapointFunc             func(goalSlug, timestamp, value, comment, requestid string) error
	CreateDatapointWithDaystampFunc func(goalSlug, timestamp, daystamp, value, comment, requestid string) error
	CreateChargeFunc                func(amount float64, note string, dryrun bool) (*Charge, error)
	CreateGoalFunc                  func(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error)
	CallUncleFunc                   func(goalSlug string) (*Goal, error)
	RatchetGoalFunc                 func(goalSlug string, ratchet int) (*Goal, error)
	UpdateGoalDeadlineFunc          func(goalSlug string, deadline int) (*Goal, error)
	RefreshGoalFunc                 func(goalSlug string) (bool, error)
}

// errFakeNotConfigured is returned by every FakeClient method whose
// corresponding *Func field is nil. Tests that hit it have a missing setup —
// surface that rather than letting an unconfigured path succeed silently.
var errFakeNotConfigured = errors.New("FakeClient method not configured for this test")

func (c *FakeClient) FetchGoals(ctx context.Context) ([]Goal, error) {
	if c.FetchGoalsFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.FetchGoalsFunc()
}

func (c *FakeClient) FetchGoal(ctx context.Context, goalSlug string) (*Goal, error) {
	if c.FetchGoalFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.FetchGoalFunc(goalSlug)
}

func (c *FakeClient) FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error) {
	if c.FetchGoalWithDatapointsFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.FetchGoalWithDatapointsFunc(goalSlug)
}

func (c *FakeClient) FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error) {
	if c.FetchGoalRawJSONFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.FetchGoalRawJSONFunc(goalSlug, includeDatapoints)
}

func (c *FakeClient) GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error) {
	if c.GetLastDatapointValueFunc == nil {
		return 0, errFakeNotConfigured
	}
	return c.GetLastDatapointValueFunc(goalSlug)
}

func (c *FakeClient) CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) error {
	if c.CreateDatapointFunc == nil {
		return errFakeNotConfigured
	}
	return c.CreateDatapointFunc(goalSlug, timestamp, value, comment, requestid)
}

func (c *FakeClient) CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) error {
	if c.CreateDatapointWithDaystampFunc == nil {
		return errFakeNotConfigured
	}
	return c.CreateDatapointWithDaystampFunc(goalSlug, timestamp, daystamp, value, comment, requestid)
}

func (c *FakeClient) CreateCharge(ctx context.Context, amount float64, note string, dryrun bool) (*Charge, error) {
	if c.CreateChargeFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.CreateChargeFunc(amount, note, dryrun)
}

func (c *FakeClient) CreateGoal(ctx context.Context, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
	if c.CreateGoalFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.CreateGoalFunc(slug, title, goalType, gunits, goaldate, goalval, rate)
}

func (c *FakeClient) CallUncle(ctx context.Context, goalSlug string) (*Goal, error) {
	if c.CallUncleFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.CallUncleFunc(goalSlug)
}

func (c *FakeClient) RatchetGoal(ctx context.Context, goalSlug string, ratchet int) (*Goal, error) {
	if c.RatchetGoalFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.RatchetGoalFunc(goalSlug, ratchet)
}

func (c *FakeClient) UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error) {
	if c.UpdateGoalDeadlineFunc == nil {
		return nil, errFakeNotConfigured
	}
	return c.UpdateGoalDeadlineFunc(goalSlug, deadline)
}

func (c *FakeClient) RefreshGoal(ctx context.Context, goalSlug string) (bool, error) {
	if c.RefreshGoalFunc == nil {
		return false, errFakeNotConfigured
	}
	return c.RefreshGoalFunc(goalSlug)
}

// Compile-time check that FakeClient satisfies Client.
var _ Client = (*FakeClient)(nil)
