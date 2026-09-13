package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// twoAccounts builds a multiClient over two fakes whose goal listings are
// supplied by the caller.
func twoAccounts(alice, bob []Goal) (*multiClient, *FakeClient, *FakeClient) {
	a := &FakeClient{FetchGoalsFunc: func() ([]Goal, error) { return alice, nil }}
	b := &FakeClient{FetchGoalsFunc: func() ([]Goal, error) { return bob, nil }}
	return &multiClient{accounts: []accountClient{{"alice", a}, {"bob", b}}}, a, b
}

func TestNewClientHoldsEveryConfiguredAccount(t *testing.T) {
	one, ok := newClient(&Config{Username: "alice", AuthToken: "t"}).(*multiClient)
	if !ok {
		t.Fatal("newClient should always build a *multiClient")
	}
	if len(one.accounts) != 1 || one.accounts[0].username != "alice" {
		t.Fatalf("got %v, want alice alone", one.usernames())
	}

	multi := &Config{Username: "alice", AuthToken: "t", Accounts: []Account{{Username: "bob", AuthToken: "u"}}}
	two := newClient(multi).(*multiClient)
	if len(two.accounts) != 2 {
		t.Fatalf("got %v, want both accounts", two.usernames())
	}
}

// A lone account must cost no more round trips than the plain HTTPClient did:
// there is nothing to disambiguate, so routing must not fetch a goal listing.
func TestSingleAccountRoutesWithoutAGoalListing(t *testing.T) {
	unreachable := &FakeClient{} // every method errors if called
	m := &multiClient{accounts: []accountClient{{"alice", unreachable}}}

	for _, in := range []string{"read", "alice/read"} {
		c, slug, err := m.clientFor(context.Background(), in)
		if err != nil || slug != "read" || c != Client(unreachable) {
			t.Errorf("clientFor(%q) = %q, %v; want the bare slug with no fetch", in, slug, err)
		}
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
	if c := newClient(onlyList).(*multiClient); len(c.accounts) != 1 || c.accounts[0].username != "bob" {
		t.Fatalf("got %v, want bob alone", c.usernames())
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

	// Unfiltered, every account is present.
	if c := newClient(config).(*multiClient); len(c.accounts) != 2 {
		t.Fatalf("no filter should hold both accounts, got %v", c.usernames())
	}

	// accountFilter is a package-level global; a leak would corrupt every other
	// test in the package.
	t.Cleanup(func() { accountFilter = "" })

	accountFilter = "bob"
	// Still a multiClient, but holding only bob — so a scoped command can be
	// handed a qualified "bob/read" slug and have it understood.
	m, ok := newClient(config).(*multiClient)
	if !ok {
		t.Fatal("--account should narrow to a single-account client")
	}
	if len(m.accounts) != 1 || m.accounts[0].username != "bob" {
		t.Fatalf("got %+v, want bob alone", m.usernames())
	}

	// One account needs no goal listing to route, and accepts its own
	// qualifier as well as a bare slug.
	unreachable := &FakeClient{} // every method errors if called
	m.accounts[0].client = unreachable
	for _, in := range []string{"read", "bob/read"} {
		c, slug, err := m.clientFor(context.Background(), in)
		if err != nil || slug != "read" || c != Client(unreachable) {
			t.Errorf("clientFor(%q) = %q, %v; want bare slug with no index fetch", in, slug, err)
		}
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

// A goal carrying no account (built outside a listing) qualifies nothing.
func TestUnstampedGoalsStayBare(t *testing.T) {
	g := Goal{Slug: "read"}
	if g.DisplaySlug() != "read" || g.routeSlug() != "read" {
		t.Errorf("an unstamped goal should stay bare, got %q / %q", g.DisplaySlug(), g.routeSlug())
	}
}

// With one account a goal is still stamped, but nothing is ambiguous: the user
// sees the bare slug, while routing carries the (harmless) qualifier.
func TestOneAccountDisplaysBareButRoutesQualified(t *testing.T) {
	m := &multiClient{accounts: []accountClient{{"alice", &FakeClient{
		FetchGoalsFunc: func() ([]Goal, error) { return []Goal{{ID: "1", Slug: "read"}}, nil },
	}}}}

	goals, err := m.FetchGoals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if goals[0].DisplaySlug() != "read" {
		t.Errorf("one account has nothing to disambiguate, got %q", goals[0].DisplaySlug())
	}
	if goals[0].routeSlug() != "alice/read" {
		t.Errorf("routing should carry the account, got %q", goals[0].routeSlug())
	}
	// ...and that qualifier round-trips back to the bare slug.
	if _, slug, err := m.clientFor(context.Background(), goals[0].routeSlug()); err != nil || slug != "read" {
		t.Errorf("routeSlug should resolve back to the bare slug, got %q, %v", slug, err)
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
	result, _ := testModel.updateApp(goalDetailsLoadedMsg{slug: "bob/read", goal: detailed})
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

// TestModalIgnoresAnotherAccountsDetailResponse pins the race that two accounts
// sharing a slug makes possible: open alice/read, close it, open bob/read, and
// alice's still-in-flight response arrives with the bare slug "read". Matching
// on that bare slug would file alice's datapoints under bob's name.
func TestModalIgnoresAnotherAccountsDetailResponse(t *testing.T) {
	testModel := model{
		appModel: appModel{
			goals:     []Goal{{Slug: "read", Account: "bob", ambiguous: true}},
			modalGoal: &Goal{Slug: "read", Account: "bob", ambiguous: true, Title: "Bob reads"},
			mode:      modeGoalDetail,
		},
	}

	stale := &Goal{Slug: "read", Title: "Alice reads"}
	result, _ := testModel.updateApp(goalDetailsLoadedMsg{slug: "alice/read", goal: stale})
	got := result.(model).appModel.modalGoal

	if got.Title != "Bob reads" {
		t.Errorf("another account's response must not land in this modal, got title %q", got.Title)
	}
}

// TestCursorFindsTheRightSameSlugGoal pins the cursor/modal sync: opening the
// second of two same-named goals must move the cursor to *that* one. Matching
// on the bare slug stops at the first, after which left/right navigation in the
// modal walks through the wrong account's neighbours.
func TestCursorFindsTheRightSameSlugGoal(t *testing.T) {
	m := &appModel{goals: []Goal{
		{Slug: "read", Account: "alice", ambiguous: true},
		{Slug: "walk", Account: "alice"},
		{Slug: "read", Account: "bob", ambiguous: true},
	}}

	if got := m.indexOfGoal(&m.goals[2]); got != 2 {
		t.Errorf("bob/read is at index 2, got %d", got)
	}
	if got := m.indexOfGoal(&m.goals[0]); got != 0 {
		t.Errorf("alice/read is at index 0, got %d", got)
	}
	if got := m.indexOfGoal(&Goal{Slug: "gone", Account: "alice"}); got != -1 {
		t.Errorf("a goal not in the list should report -1, got %d", got)
	}
}

// TestAccountLabelFollowsTheFilter: with --account set, the grid header names
// the account actually being shown, not every configured one.
func TestAccountLabelFollowsTheFilter(t *testing.T) {
	config := &Config{Username: "alice", AuthToken: "a", Accounts: []Account{{Username: "bob", AuthToken: "b"}}}
	if got := accountLabel(config); got != "alice, bob" {
		t.Errorf("unfiltered header should name every account, got %q", got)
	}

	t.Cleanup(func() { accountFilter = "" })
	accountFilter = "bob"
	if got := accountLabel(config); got != "bob" {
		t.Errorf("--account bob shows only bob's goals, so the header should say bob, got %q", got)
	}
}

// TestArchivingSlugsMatchWhatTheTimelineStores: the schedule timeline stores
// DisplaySlug and looks the archive set up by it, so both sides must agree —
// a bare key here silently drops the colour from exactly the ambiguous goals.
func TestArchivingSlugsMatchWhatTheTimelineStores(t *testing.T) {
	soon := time.Now().Add(24 * time.Hour).Unix()
	goals := []Goal{
		{Slug: "read", Account: "alice", ambiguous: true, Archivedate: soon},
		{Slug: "walk", Account: "alice"},
	}

	set := archivingSlugs(goals, time.Now())
	if !set[goals[0].DisplaySlug()] {
		t.Errorf("the timeline looks up %q; got set %v", goals[0].DisplaySlug(), set)
	}
	if set[goals[1].DisplaySlug()] {
		t.Error("a goal not scheduled for archive should not be in the set")
	}
}
