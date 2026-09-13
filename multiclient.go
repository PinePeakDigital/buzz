package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
)

// newClient builds the Client for a config: a multiClient over every configured
// account, or over just the one named by the global --account filter.
//
// One account takes no more round trips than the plain HTTPClient ever did —
// clientFor short-circuits when there is nothing to disambiguate — and going
// through multiClient uniformly means the "username/slug" qualifier is
// understood on every path. An HTTPClient handed a qualified slug would escape
// the "/" into the slug itself and ask Beeminder for a goal named "alice%2Fread".
func newClient(config *Config) Client {
	configs := config.accountConfigs()

	// Config.checkAccounts has already rejected an unconfigured --account at
	// every entry point, so a non-matching filter here can only be a
	// programming error; fall through to the unfiltered client rather than
	// silently acting as an account the user didn't name.
	if accountFilter != "" {
		for _, c := range configs {
			if c.Username == accountFilter {
				configs = []*Config{c}
				break
			}
		}
	}

	// An unauthenticated config still builds a client; callers gate on
	// Config.checkAccounts first, and the API rejects the empty credentials.
	if len(configs) == 0 {
		configs = []*Config{config}
	}

	accounts := make([]accountClient, len(configs))
	for i, c := range configs {
		accounts[i] = accountClient{username: c.Username, client: NewHTTPClient(c)}
	}
	return &multiClient{accounts: accounts}
}

// accountClient pairs a Client with the username it authenticates as, so
// errors and routing can name the account rather than an index.
type accountClient struct {
	username string
	client   Client
}

// multiClient spreads the Client interface across the configured Beeminder
// accounts. It is the only Client construction path — one account included,
// where it is a thin pass-through that still understands the "username/slug"
// qualifier.
// Calls fall into three groups:
//
//   - listing the user's goals — fans out to every account and merges, dropping
//     repeats of a goal shared by more than one account;
//   - naming a goal by slug — routed to the account that owns that slug;
//   - account-scoped odds and ends (timezone, charges, raw API) — the primary
//     account, since there is no goal to route on.
//
// ponytail: accounts are queried sequentially. One, two or three accounts is
// the realistic case; fan out concurrently if someone with many accounts finds
// startup slow.
type multiClient struct {
	accounts []accountClient

	mu    sync.Mutex
	owner map[string][]int // goal slug -> indexes into accounts; built lazily by index()
}

var _ Client = (*multiClient)(nil)

// primary is the account used by calls that name no goal.
func (m *multiClient) primary() Client { return m.accounts[0].client }

// fetchAll runs fetch against every account, in account order. One account
// failing fails the whole call: a silently dropped account is a silently
// missing goal, and a goal you can't see is a goal you derail.
func (m *multiClient) fetchAll(fetch func(Client) ([]Goal, error)) ([][]Goal, error) {
	per := make([][]Goal, len(m.accounts))
	for i, a := range m.accounts {
		goals, err := fetch(a.client)
		if err != nil {
			return nil, fmt.Errorf("account %s: %w", a.username, err)
		}
		per[i] = goals
	}
	return per, nil
}

// merge flattens per-account results, stamping each goal with its account and
// keeping only the first copy of a goal that several accounts can see. Identity
// is Beeminder's goal id; a goal with no id falls back to account+slug, which
// dedupes nothing — deliberately, since collapsing two goals that merely share
// a slug would hide one of them.
func merge(accounts []accountClient, per [][]Goal) []Goal {
	var merged []Goal
	seen := make(map[string]bool)
	bySlug := make(map[string][]int)
	for i, goals := range per {
		for _, g := range goals {
			key := g.ID
			if key == "" {
				key = accounts[i].username + "/" + g.Slug
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			g.Account = accounts[i].username
			merged = append(merged, g)
			bySlug[g.Slug] = append(bySlug[g.Slug], len(merged)-1)
		}
	}
	// A slug two accounts both use can't identify a goal on its own, so mark
	// every copy of it for the display and routing paths.
	for _, idxs := range bySlug {
		if len(idxs) < 2 {
			continue
		}
		for _, i := range idxs {
			merged[i].ambiguous = true
		}
	}
	return merged
}

func (m *multiClient) FetchGoals(ctx context.Context) ([]Goal, error) {
	per, err := m.fetchAll(func(c Client) ([]Goal, error) { return c.FetchGoals(ctx) })
	if err != nil {
		return nil, err
	}
	m.record(per, false)
	return merge(m.accounts, per), nil
}

func (m *multiClient) FetchArchivedGoals(ctx context.Context) ([]Goal, error) {
	per, err := m.fetchAll(func(c Client) ([]Goal, error) { return c.FetchArchivedGoals(ctx) })
	if err != nil {
		return nil, err
	}
	// Archived goals stay reachable by slug (buzz view/data on an archived
	// goal), so they belong in the routing index too — added to it, not
	// replacing it, since a listing of archived goals says nothing about
	// where the active ones live. A later active-goal fetch rebuilds the map
	// and drops these again, which is what keeps the index self-healing; they
	// last as long as the command that listed them, and "username/slug" is
	// the durable way to name an archived goal on a secondary account.
	m.record(per, true)
	return merge(m.accounts, per), nil
}

// record rebuilds (or, with add, extends) the slug -> accounts index from a
// goal listing. Each slug maps to the accounts that have a goal by that name;
// an account is listed once per slug however many listings mention it.
func (m *multiClient) record(per [][]Goal, add bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !add || m.owner == nil {
		m.owner = make(map[string][]int)
	}
	for i, goals := range per {
		for _, g := range goals {
			if !slices.Contains(m.owner[g.Slug], i) {
				m.owner[g.Slug] = append(m.owner[g.Slug], i)
			}
		}
	}
}

// index returns the slug -> accounts map, fetching goals to build it if no
// listing has happened yet this run. refresh forces a re-fetch, which is how a
// slug that appeared after the index was built (a goal created mid-session)
// still finds its account instead of being misrouted.
//
// ponytail: only *active* goals are fetched to build the index — archived ones
// land in it when something lists them, but are otherwise routed to the primary
// account. Fetching both lists here would double every routing round trip to
// serve a rare case; "username/slug" names the account explicitly meanwhile.
func (m *multiClient) index(ctx context.Context, refresh bool) (map[string][]int, error) {
	m.mu.Lock()
	owner := m.owner
	m.mu.Unlock()
	if owner != nil && !refresh {
		return owner, nil
	}
	if _, err := m.FetchGoals(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.owner, nil
}

// splitAccount pulls an explicit "username/slug" qualifier off a slug. Beeminder
// slugs cannot contain "/", so a slash is unambiguously an account prefix.
func splitAccount(slug string) (account, bare string) {
	if i := strings.Index(slug, "/"); i >= 0 {
		return slug[:i], slug[i+1:]
	}
	return "", slug
}

// clientFor resolves a goal slug to the account that owns it, returning that
// account's client, its username, and the slug stripped of any "username/"
// qualifier. A bare slug that exists on more than one account is an error rather
// than a guess — picking one would write a datapoint to the wrong person's goal.
func (m *multiClient) clientFor(ctx context.Context, slug string) (Client, string, string, error) {
	if account, bare := splitAccount(slug); account != "" {
		c, err := m.clientByUsername(account)
		if err != nil {
			return nil, "", "", err
		}
		return c, account, bare, nil
	}

	// With one account there is nothing to disambiguate, so skip the index
	// entirely. This is what keeps the ordinary single-account setup (and
	// --account) paying no more round trips than a bare HTTPClient would.
	if len(m.accounts) == 1 {
		return m.accounts[0].client, m.accounts[0].username, slug, nil
	}

	m.mu.Lock()
	cached := m.owner != nil
	m.mu.Unlock()

	owner, err := m.index(ctx, false)
	if err != nil {
		return nil, "", "", err
	}
	// A *cached* index can simply be out of date — the goal may have been
	// created since it was built. Rebuild once before concluding the slug is
	// unknown, so a fresh goal isn't silently sent to the wrong account. An
	// index just built above is already current; re-fetching it would only
	// double the round trips.
	if cached && len(owner[slug]) == 0 {
		if owner, err = m.index(ctx, true); err != nil {
			return nil, "", "", err
		}
	}
	switch owners := owner[slug]; len(owners) {
	case 0:
		// Genuinely unknown: hand it to the primary account so the API's own
		// "goal not found" is what the user sees.
		return m.primary(), m.accounts[0].username, slug, nil
	case 1:
		return m.accounts[owners[0]].client, m.accounts[owners[0]].username, slug, nil
	default:
		var names []string
		for _, i := range owners {
			names = append(names, m.accounts[i].username+"/"+slug)
		}
		return nil, "", "", fmt.Errorf("goal %q exists on several accounts; name one: %s", slug, strings.Join(names, ", "))
	}
}

// clientByUsername resolves an explicit account qualifier to its client.
func (m *multiClient) clientByUsername(username string) (Client, error) {
	for _, a := range m.accounts {
		if a.username == username {
			return a.client, nil
		}
	}
	return nil, fmt.Errorf("no such account: %s (configured: %s)", username, strings.Join(m.usernames(), ", "))
}

func (m *multiClient) usernames() []string {
	names := make([]string, len(m.accounts))
	for i, a := range m.accounts {
		names[i] = a.username
	}
	return names
}

// Account-scoped calls that name no goal: the primary account answers.

// FetchUserTimezone returns the *primary* account's timezone.
//
// ponytail: `buzz schedule` applies this one timezone to every goal, so a
// secondary account set to a different Beeminder timezone has its deadlines
// rendered in the primary's. Give the callers that care a per-goal lookup
// (keyed on Goal.Account) if anyone actually runs accounts across timezones.
func (m *multiClient) FetchUserTimezone(ctx context.Context) (string, error) {
	return m.primary().FetchUserTimezone(ctx)
}

func (m *multiClient) APIRequest(ctx context.Context, method, path string, params url.Values) (int, []byte, error) {
	return m.primary().APIRequest(ctx, method, path, params)
}

func (m *multiClient) CreateCharge(ctx context.Context, amount float64, note string, dryrun bool) (*Charge, error) {
	return m.primary().CreateCharge(ctx, amount, note, dryrun)
}

// CreateGoal makes the goal on the account named by a "username/slug" prefix,
// or on the primary account. There is no index lookup — the goal does not exist
// yet, so there is nothing to route on.
func (m *multiClient) CreateGoal(ctx context.Context, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
	account, bare := splitAccount(slug)
	client := m.primary()
	if account != "" {
		c, err := m.clientByUsername(account)
		if err != nil {
			return nil, err
		}
		client = c
	}
	return client.CreateGoal(ctx, bare, title, goalType, gunits, goaldate, goalval, rate)
}

// stamped records which account a routed call actually reached. A single-goal
// fetch returns the raw goal, which carries no provenance of its own — without
// this, `buzz view bob/read` would render the primary account's URL.
func stamped(g *Goal, err error, account string) (*Goal, error) {
	if err != nil || g == nil {
		return g, err
	}
	g.Account = account
	return g, nil
}

// Goal-scoped calls: routed to the owning account.

func (m *multiClient) FetchGoal(ctx context.Context, goalSlug string) (*Goal, error) {
	c, account, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	g, err := c.FetchGoal(ctx, slug)
	return stamped(g, err, account)
}

func (m *multiClient) FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error) {
	c, account, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	g, err := c.FetchGoalWithDatapoints(ctx, slug)
	return stamped(g, err, account)
}

func (m *multiClient) FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error) {
	c, _, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.FetchGoalRawJSON(ctx, slug, includeDatapoints)
}

func (m *multiClient) GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error) {
	c, _, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return 0, err
	}
	return c.GetLastDatapointValue(ctx, slug)
}

func (m *multiClient) CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) (*Datapoint, error) {
	return m.CreateDatapointWithDaystamp(ctx, goalSlug, timestamp, "", value, comment, requestid)
}

func (m *multiClient) CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) (*Datapoint, error) {
	c, _, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.CreateDatapointWithDaystamp(ctx, slug, timestamp, daystamp, value, comment, requestid)
}

func (m *multiClient) CallUncle(ctx context.Context, goalSlug string) (*Goal, error) {
	c, account, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	g, err := c.CallUncle(ctx, slug)
	return stamped(g, err, account)
}

func (m *multiClient) RatchetGoal(ctx context.Context, goalSlug string, ratchet int) (*Goal, error) {
	c, account, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	g, err := c.RatchetGoal(ctx, slug, ratchet)
	return stamped(g, err, account)
}

func (m *multiClient) UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error) {
	c, account, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	g, err := c.UpdateGoalDeadline(ctx, slug, deadline)
	return stamped(g, err, account)
}

func (m *multiClient) RefreshGoal(ctx context.Context, goalSlug string) (bool, error) {
	c, _, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return false, err
	}
	return c.RefreshGoal(ctx, slug)
}
