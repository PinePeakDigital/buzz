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

func TestCreateGoalRoutesByQualifier(t *testing.T) {
	m, a, b := twoAccounts(nil, nil)
	var on string
	made := func(name string) func(string, string, string, string, string, string, string) (*Goal, error) {
		return func(slug, _, _, _, _, _, _ string) (*Goal, error) {
			on = name + "/" + slug
			return &Goal{Slug: slug}, nil
		}
	}
	a.CreateGoalFunc, b.CreateGoalFunc = made("alice"), made("bob")

	// Unqualified goes to the primary; no index fetch is needed to create.
	if _, err := m.CreateGoal(context.Background(), "new", "", "", "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if on != "alice/new" {
		t.Fatalf("unqualified create should hit the primary, got %q", on)
	}

	if _, err := m.CreateGoal(context.Background(), "bob/new", "", "", "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if on != "bob/new" {
		t.Fatalf("qualified create should hit bob with the bare slug, got %q", on)
	}

	if _, err := m.CreateGoal(context.Background(), "carol/new", "", "", "", "", "", ""); err == nil {
		t.Fatal("create on an unknown account should error")
	}
}

func TestUnknownSlugFallsBackToPrimaryAfterRefresh(t *testing.T) {
	alice := []Goal{{ID: "1", Slug: "read"}}
	m, a, _ := twoAccounts(alice, []Goal{{ID: "3", Slug: "walk"}})

	c, slug, err := m.clientFor(context.Background(), "nope")
	if err != nil {
		t.Fatal(err)
	}
	if c != Client(a) || slug != "nope" {
		t.Fatalf("an unknown slug should fall back to the primary with the slug intact, got %q", slug)
	}
}

func TestStaleIndexIsRefreshedForAGoalCreatedMidSession(t *testing.T) {
	bobGoals := []Goal{{ID: "3", Slug: "walk"}}
	m, _, b := twoAccounts([]Goal{{ID: "1", Slug: "read"}}, nil)
	b.FetchGoalsFunc = func() ([]Goal, error) { return bobGoals, nil }

	// Build the index, then create a goal on bob behind its back.
	if _, err := m.FetchGoals(context.Background()); err != nil {
		t.Fatal(err)
	}
	bobGoals = append(bobGoals, Goal{ID: "4", Slug: "swim"})

	c, _, err := m.clientFor(context.Background(), "swim")
	if err != nil {
		t.Fatal(err)
	}
	if c != Client(b) {
		t.Fatal("a goal created after the index was built should still route to its own account")
	}
}

func TestArchivedListingFeedsTheRoutingIndex(t *testing.T) {
	m, _, b := twoAccounts([]Goal{{ID: "1", Slug: "read"}}, nil)
	b.FetchArchivedGoalsFunc = func() ([]Goal, error) { return []Goal{{ID: "9", Slug: "old"}}, nil }
	m.accounts[0].client.(*FakeClient).FetchArchivedGoalsFunc = func() ([]Goal, error) { return nil, nil }

	if _, err := m.FetchArchivedGoals(context.Background()); err != nil {
		t.Fatal(err)
	}
	c, _, err := m.clientFor(context.Background(), "old")
	if err != nil {
		t.Fatal(err)
	}
	if c != Client(b) {
		t.Fatal("an archived goal seen in a listing should route to its own account")
	}
}

func TestAccountConfigsSkipsBlankPrimaryAndDuplicates(t *testing.T) {
	// A hand-written config that only lists "accounts" must not produce a
	// blank-credentialled phantom primary.
	onlyList := &Config{Accounts: []Account{{Username: "bob", AuthToken: "b"}}}
	cfgs := onlyList.accountConfigs()
	if len(cfgs) != 1 || cfgs[0].Username != "bob" {
		t.Fatalf("a config with no primary should yield just its listed accounts, got %+v", cfgs)
	}
	if _, ok := newClient(onlyList).(*HTTPClient); !ok {
		t.Fatal("one usable account should still build a plain *HTTPClient")
	}

	// A username listed twice must collapse, or every one of its goals looks
	// ambiguous with no qualifier able to resolve it.
	dup := &Config{Username: "alice", AuthToken: "a", Accounts: []Account{{Username: "alice", AuthToken: "a"}}}
	if cfgs := dup.accountConfigs(); len(cfgs) != 1 {
		t.Fatalf("a duplicated username should collapse to one account, got %+v", cfgs)
	}

	empty := &Config{}
	if empty.hasCredentials() || len(empty.accountConfigs()) != 0 {
		t.Fatal("an empty config should report no credentials")
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

	// Re-authenticating as an existing SECONDARY account refreshes its token
	// rather than adding a duplicate entry.
	c.setAccount("dave", "d")
	c.setAccount("dave", "d2")
	if len(c.Accounts) != 1 || c.Accounts[0].AuthToken != "d2" {
		t.Fatalf("re-auth as a secondary should refresh in place: %+v", c)
	}

	// Removing the last account leaves a valid but credential-less config.
	c.removeAccount("dave")
	if !c.removeAccount("bob") || c.Username != "" || c.AuthToken != "" {
		t.Fatalf("removing the last account should clear the credentials: %+v", c)
	}
	if c.hasCredentials() {
		t.Fatal("a config with no accounts left should report no credentials")
	}
}

// TestGoalURLUsesTheOwningAccount pins the bug that multi-account introduced:
// a goal from a secondary account must link to that account's page, not the
// primary's — which would 404, or silently open a same-slug goal.
func TestGoalURLUsesTheOwningAccount(t *testing.T) {
	config := &Config{Username: "alice", AuthToken: "a", BaseURL: "https://example.test"}

	if got := goalURL(config, "bob", "read"); got != "https://example.test/bob/read" {
		t.Errorf("got %q, want bob's page", got)
	}
	// Single-account goals carry no Account, and fall back to the primary.
	if got := goalURL(config, "", "read"); got != "https://example.test/alice/read" {
		t.Errorf("got %q, want the primary's page", got)
	}
}

func TestAccountFilterNarrowsToOneAccount(t *testing.T) {
	config := &Config{Username: "alice", AuthToken: "a", Accounts: []Account{{Username: "bob", AuthToken: "b"}}}

	// Unfiltered, several accounts fan out.
	if _, ok := newClient(config).(*multiClient); !ok {
		t.Fatal("no filter should build a multiClient")
	}

	// accountFilter is a package-level global; a leak would corrupt every other
	// test in the package.
	t.Cleanup(func() { accountFilter = "" })

	accountFilter = "bob"
	c, ok := newClient(config).(*HTTPClient)
	if !ok {
		t.Fatal("--account should narrow to a single-account client")
	}
	if c.config.Username != "bob" || c.config.AuthToken != "b" {
		t.Fatalf("got %+v, want bob's credentials", c.config)
	}
	if err := config.checkAccounts(); err != nil {
		t.Fatalf("a configured account should pass: %v", err)
	}

	accountFilter = "carol"
	err := config.checkAccounts()
	if err == nil || !strings.Contains(err.Error(), "alice, bob") {
		t.Fatalf("an unconfigured --account should be rejected and list the real ones, got %v", err)
	}
}

func TestAmbiguousSlugsAreQualifiedForDisplayAndRouting(t *testing.T) {
	m, _, _ := twoAccounts(
		[]Goal{{ID: "1", Slug: "read"}, {ID: "2", Slug: "solo"}},
		[]Goal{{ID: "3", Slug: "read"}},
	)
	goals, err := m.FetchGoals(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"1": "alice/read", "2": "solo", "3": "bob/read"}
	for _, g := range goals {
		if got := g.DisplaySlug(); got != want[g.ID] {
			t.Errorf("goal %s: display %q, want %q", g.ID, got, want[g.ID])
		}
		// Routing always qualifies, so a same-slug goal reaches its own account.
		if g.routeSlug() != g.Account+"/"+g.Slug {
			t.Errorf("goal %s: routeSlug %q should be account-qualified", g.ID, g.routeSlug())
		}
	}
}

func TestFullyOverlappingAccountsDoNotDoubleGoals(t *testing.T) {
	shared := []Goal{{ID: "1", Slug: "read"}, {ID: "2", Slug: "walk"}}
	m, _, _ := twoAccounts(shared, shared)

	goals, err := m.FetchGoals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != len(shared) {
		t.Fatalf("two accounts seeing the identical goal set should merge to %d, got %d", len(shared), len(goals))
	}
	for _, g := range goals {
		if g.ambiguous {
			t.Errorf("goal %s is the same goal, not two goals sharing a slug", g.ID)
		}
	}
}

// A single-account setup stamps no Account, so nothing is qualified anywhere.
func TestSingleAccountGoalsStayBare(t *testing.T) {
	g := Goal{Slug: "read"}
	if g.DisplaySlug() != "read" || g.routeSlug() != "read" {
		t.Errorf("single-account goals should stay bare, got %q / %q", g.DisplaySlug(), g.routeSlug())
	}
}

// TestModalKeepsAccountAcrossDetailFetch pins the field-preserving replace in
// tui.go: a detail fetch returns the owning account's raw goal, which carries
// no Account (only a multi-account listing stamps one). Replacing modalGoal
// wholesale would revert the modal's URL to the primary account's page.
func TestModalKeepsAccountAcrossDetailFetch(t *testing.T) {
	testModel := model{
		appModel: appModel{
			goals:     []Goal{{Slug: "read", Account: "bob", ambiguous: true}},
			modalGoal: &Goal{Slug: "read", Account: "bob", ambiguous: true},
			mode:      modeGoalDetail,
		},
	}

	// The detail fetch's goal has the same slug but no account provenance.
	detailed := &Goal{Slug: "read", Title: "Read more"}
	result, _ := testModel.updateApp(goalDetailsLoadedMsg{goal: detailed})
	got := result.(model).appModel.modalGoal

	if got.Title != "Read more" {
		t.Errorf("the detail fetch's fields should land, got title %q", got.Title)
	}
	if got.Account != "bob" || !got.ambiguous {
		t.Errorf("account provenance should survive the replace, got %+v", got)
	}
	if got.DisplaySlug() != "bob/read" {
		t.Errorf("DisplaySlug = %q, want bob/read", got.DisplaySlug())
	}
}
