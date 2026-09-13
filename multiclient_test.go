package main

import (
	"context"
	"strings"
	"testing"
)

// twoAccounts builds a multiClient over two fakes whose goal listings are
// supplied by the caller.
func twoAccounts(alice, bob []Goal) (*multiClient, *FakeClient, *FakeClient) {
	a := &FakeClient{FetchGoalsFunc: func() ([]Goal, error) { return alice, nil }}
	b := &FakeClient{FetchGoalsFunc: func() ([]Goal, error) { return bob, nil }}
	return &multiClient{accounts: []accountClient{{"alice", a}, {"bob", b}}}, a, b
}

func TestNewClientSingleAccountStaysHTTPClient(t *testing.T) {
	if _, ok := newClient(&Config{Username: "alice", AuthToken: "t"}).(*HTTPClient); !ok {
		t.Fatal("single-account config should build a plain *HTTPClient")
	}
	multi := &Config{Username: "alice", AuthToken: "t", Accounts: []Account{{Username: "bob", AuthToken: "u"}}}
	if _, ok := newClient(multi).(*multiClient); !ok {
		t.Fatal("two-account config should build a *multiClient")
	}
}

func TestFetchGoalsMergesAndDedupes(t *testing.T) {
	m, _, _ := twoAccounts(
		[]Goal{{ID: "1", Slug: "read"}, {ID: "2", Slug: "shared"}},
		[]Goal{{ID: "2", Slug: "shared"}, {ID: "3", Slug: "walk"}},
	)
	goals, err := m.FetchGoals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 3 {
		t.Fatalf("want 3 goals after deduping the shared one, got %d: %+v", len(goals), goals)
	}
	if goals[1].Account != "alice" || goals[2].Account != "bob" {
		t.Fatalf("goals should be stamped with the account they came from: %+v", goals)
	}
}

func TestFetchGoalsFailsLoudlyWhenAnAccountFails(t *testing.T) {
	m, _, b := twoAccounts([]Goal{{ID: "1", Slug: "read"}}, nil)
	b.FetchGoalsFunc = nil // unconfigured fake => error
	_, err := m.FetchGoals(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bob") {
		t.Fatalf("want an error naming the failing account, got %v", err)
	}
}

func TestWritesRouteToTheOwningAccount(t *testing.T) {
	m, a, b := twoAccounts([]Goal{{ID: "1", Slug: "read"}}, []Goal{{ID: "3", Slug: "walk"}})
	var got string
	b.CreateDatapointWithDaystampFunc = func(slug, _, _, _, _, _ string) (*Datapoint, error) {
		got = slug
		return &Datapoint{}, nil
	}
	a.CreateDatapointWithDaystampFunc = func(string, string, string, string, string, string) (*Datapoint, error) {
		t.Fatal("datapoint went to the wrong account")
		return nil, nil
	}
	if _, err := m.CreateDatapoint(context.Background(), "walk", "", "1", "", ""); err != nil {
		t.Fatal(err)
	}
	if got != "walk" {
		t.Fatalf("want slug walk, got %q", got)
	}
}

func TestAmbiguousSlugIsAnErrorAndQualifyingResolvesIt(t *testing.T) {
	m, _, b := twoAccounts([]Goal{{ID: "1", Slug: "read"}}, []Goal{{ID: "2", Slug: "read"}})
	_, _, err := m.clientFor(context.Background(), "read")
	if err == nil || !strings.Contains(err.Error(), "bob/read") {
		t.Fatalf("want an ambiguity error listing qualified slugs, got %v", err)
	}
	c, slug, err := m.clientFor(context.Background(), "bob/read")
	if err != nil || slug != "read" || c != Client(b) {
		t.Fatalf("qualified slug should resolve to bob with the bare slug: %v %q", err, slug)
	}
	if _, _, err := m.clientFor(context.Background(), "carol/read"); err == nil {
		t.Fatal("unknown account should error")
	}
}

func TestConfigAccountAddReplaceRemove(t *testing.T) {
	c := &Config{Username: "alice", AuthToken: "old", BaseURL: "https://example.test"}

	c.setAccount("alice", "new")
	if c.AuthToken != "new" || len(c.Accounts) != 0 {
		t.Fatalf("re-auth as the same user should refresh the token in place: %+v", c)
	}

	c.setAccount("bob", "b")
	if len(c.Accounts) != 1 || c.Accounts[0].Username != "bob" {
		t.Fatalf("a new username should be added alongside: %+v", c)
	}

	if cfgs := c.accountConfigs(); len(cfgs) != 2 || cfgs[1].BaseURL != "https://example.test" {
		t.Fatalf("every account config should carry the shared base URL: %+v", cfgs)
	}

	if !c.removeAccount("alice") || c.Username != "bob" || len(c.Accounts) != 0 {
		t.Fatalf("removing the primary should promote the next account: %+v", c)
	}
	if c.removeAccount("carol") {
		t.Fatal("removing an unknown account should report false")
	}
}
