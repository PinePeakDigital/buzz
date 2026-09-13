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

// newClient builds the Client for a config: a plain HTTPClient when a single
// account is configured, a multiClient fanning across all of them otherwise.
// The single-account path is byte-for-byte the behaviour buzz has always had.
func newClient(config *Config) Client {
	configs := config.accountConfigs()
	if len(configs) <= 1 {
		// Zero accounts means an unauthenticated config; callers gate on
		// Config.hasCredentials before getting here, so build the same client
		// the single-account path always did and let the API reject it.
		if len(configs) == 0 {
			return NewHTTPClient(config)
		}
		return NewHTTPClient(configs[0])
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

// multiClient spreads the Client interface across several Beeminder accounts.
// Calls fall into three groups:
//
//   - listing the user's goals — fans out to every account and merges, dropping
//     repeats of a goal shared by more than one account;
//   - naming a goal by slug — routed to the account that owns that slug;
//   - account-scoped odds and ends (timezone, charges, raw API) — the primary
//     account, since there is no goal to route on.
//
// ponytail: accounts are queried sequentially. Two or three accounts is the
// realistic case; fan out concurrently if someone with many accounts finds
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
	// where the active ones live.
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

// clientFor resolves a goal slug to the account that owns it, returning the
// slug stripped of any "username/" qualifier. A bare slug that exists on more
// than one account is an error rather than a guess — picking one would write a
// datapoint to the wrong person's goal.
func (m *multiClient) clientFor(ctx context.Context, slug string) (Client, string, error) {
	if account, bare := splitAccount(slug); account != "" {
		c, err := m.clientByUsername(account)
		if err != nil {
			return nil, "", err
		}
		return c, bare, nil
	}

	m.mu.Lock()
	cached := m.owner != nil
	m.mu.Unlock()

	owner, err := m.index(ctx, false)
	if err != nil {
		return nil, "", err
	}
	// A *cached* index can simply be out of date — the goal may have been
	// created since it was built. Rebuild once before concluding the slug is
	// unknown, so a fresh goal isn't silently sent to the wrong account. An
	// index just built above is already current; re-fetching it would only
	// double the round trips.
	if cached && len(owner[slug]) == 0 {
		if owner, err = m.index(ctx, true); err != nil {
			return nil, "", err
		}
	}
	switch owners := owner[slug]; len(owners) {
	case 0:
		// Genuinely unknown: hand it to the primary account so the API's own
		// "goal not found" is what the user sees.
		return m.primary(), slug, nil
	case 1:
		return m.accounts[owners[0]].client, slug, nil
	default:
		var names []string
		for _, i := range owners {
			names = append(names, m.accounts[i].username+"/"+slug)
		}
		return nil, "", fmt.Errorf("goal %q exists on several accounts; name one: %s", slug, strings.Join(names, ", "))
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

// Goal-scoped calls: routed to the owning account.

func (m *multiClient) FetchGoal(ctx context.Context, goalSlug string) (*Goal, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.FetchGoal(ctx, slug)
}

func (m *multiClient) FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.FetchGoalWithDatapoints(ctx, slug)
}

func (m *multiClient) FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.FetchGoalRawJSON(ctx, slug, includeDatapoints)
}

func (m *multiClient) GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return 0, err
	}
	return c.GetLastDatapointValue(ctx, slug)
}

func (m *multiClient) CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) (*Datapoint, error) {
	return m.CreateDatapointWithDaystamp(ctx, goalSlug, timestamp, "", value, comment, requestid)
}

func (m *multiClient) CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) (*Datapoint, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.CreateDatapointWithDaystamp(ctx, slug, timestamp, daystamp, value, comment, requestid)
}

func (m *multiClient) CallUncle(ctx context.Context, goalSlug string) (*Goal, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.CallUncle(ctx, slug)
}

func (m *multiClient) RatchetGoal(ctx context.Context, goalSlug string, ratchet int) (*Goal, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.RatchetGoal(ctx, slug, ratchet)
}

func (m *multiClient) UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return nil, err
	}
	return c.UpdateGoalDeadline(ctx, slug, deadline)
}

func (m *multiClient) RefreshGoal(ctx context.Context, goalSlug string) (bool, error) {
	c, slug, err := m.clientFor(ctx, goalSlug)
	if err != nil {
		return false, err
	}
	return c.RefreshGoal(ctx, slug)
}
