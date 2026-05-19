package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// httpClientTimeout caps every Beeminder request so a stalled connection
// can't freeze the CLI or block a Bubble Tea Cmd indefinitely. Per-request
// context (this file) layers on top: callers can cancel a request before
// the timeout fires, e.g. when the user quits the TUI.
const httpClientTimeout = 30 * time.Second

// Client is the Beeminder API seam. Every method takes a context.Context as
// its first parameter; callers should pass either the long-lived appModel
// context (TUI) or context.Background() (short-lived CLI commands). The
// context.Context support enables future quit-cancellation wiring without
// further interface changes — that wiring is tracked in a follow-up.
type Client interface {
	FetchGoals(ctx context.Context) ([]Goal, error)
	FetchGoal(ctx context.Context, goalSlug string) (*Goal, error)
	FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error)
	FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error)
	GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error)
	CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) error
	CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) error
	CreateCharge(ctx context.Context, amount float64, note string, dryrun bool) (*Charge, error)
	CreateGoal(ctx context.Context, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error)
	CallUncle(ctx context.Context, goalSlug string) (*Goal, error)
	UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error)
	RefreshGoal(ctx context.Context, goalSlug string) (bool, error)
}

// HTTPClient is the HTTP-backed Client. Construct with NewHTTPClient.
type HTTPClient struct {
	config *Config
	http   *http.Client
}

// MultiClient fans out reads across multiple Beeminder accounts and routes
// goal-specific operations to the first account where the goal exists.
type MultiClient struct {
	clients []Client
}

// NewHTTPClient returns a Client backed by net/http using credentials in config.
// The returned value can be assigned to a Client interface variable; downstream
// code should depend on Client, not *HTTPClient.
func NewHTTPClient(config *Config) Client {
	accounts := config.accountCredentials()
	if len(accounts) <= 1 {
		cfg := config
		if len(accounts) == 1 {
			account := accounts[0]
			cfg = &account
		}
		return &HTTPClient{
			config: cfg,
			http:   &http.Client{Timeout: httpClientTimeout},
		}
	}

	clients := make([]Client, 0, len(accounts))
	for _, account := range accounts {
		accountCopy := account
		clients = append(clients, &HTTPClient{
			config: &accountCopy,
			http:   &http.Client{Timeout: httpClientTimeout},
		})
	}
	return &MultiClient{
		clients: clients,
	}
}

// getBaseURL returns the configured base URL or the default Beeminder URL.
// Also used by non-API code (e.g. building browser URLs in review.go).
func getBaseURL(config *Config) string {
	if config.BaseURL == "" {
		return "https://www.beeminder.com"
	}
	return config.BaseURL
}

func (c *HTTPClient) baseURL() string {
	return getBaseURL(c.config)
}

// doRequest builds a context-aware request, executes it, and emits the
// LogRequest/LogResponse pair. The contentType argument is set as the
// Content-Type header when non-empty (POST/PUT bodies). Per-method callers
// own status-code interpretation and body decoding.
func (c *HTTPClient) doRequest(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	LogRequest(c.config, method, url)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	LogResponse(c.config, resp.StatusCode, url)
	return resp, nil
}

// FetchGoals fetches the user's goals from Beeminder API.
func (c *HTTPClient) FetchGoals(ctx context.Context) ([]Goal, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/goals.json?auth_token=%s",
		c.baseURL(), c.config.Username, c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goals: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var goals []Goal
	if err := json.NewDecoder(resp.Body).Decode(&goals); err != nil {
		return nil, fmt.Errorf("failed to decode goals: %w", err)
	}
	for i := range goals {
		if goals[i].Username == "" {
			goals[i].Username = c.config.Username
		}
	}

	return goals, nil
}

// GetLastDatapointValue fetches the last datapoint value for a goal.
func (c *HTTPClient) GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s&skinny=true",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodGet, apiURL, nil, "")
	if err != nil {
		return 0, fmt.Errorf("failed to fetch goal details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		LastDatapoint *Datapoint `json:"last_datapoint"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode goal details: %w", err)
	}

	if result.LastDatapoint == nil {
		return 0, nil
	}

	return result.LastDatapoint.Value, nil
}

// CreateDatapoint submits a new datapoint to a Beeminder goal.
func (c *HTTPClient) CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) error {
	return c.CreateDatapointWithDaystamp(ctx, goalSlug, timestamp, "", value, comment, requestid)
}

// CreateDatapointWithDaystamp submits a new datapoint with optional daystamp.
// If daystamp is provided (format YYYYMMDD), it is used instead of timestamp.
func (c *HTTPClient) CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) error {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/datapoints.json",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug))

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("value", value)
	data.Set("comment", comment)

	if daystamp != "" {
		data.Set("daystamp", daystamp)
	} else {
		data.Set("timestamp", timestamp)
	}

	if requestid != "" {
		data.Set("requestid", requestid)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("failed to create datapoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateCharge creates a new charge for the authenticated user and returns it.
func (c *HTTPClient) CreateCharge(ctx context.Context, amount float64, note string, dryrun bool) (*Charge, error) {
	apiURL := fmt.Sprintf("%s/api/v1/charges.json", c.baseURL())

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("user_id", c.config.Username)
	data.Set("amount", fmt.Sprintf("%.2f", amount))
	data.Set("note", note)
	if dryrun {
		data.Set("dryrun", "true")
	}

	resp, err := c.doRequest(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("failed to create charge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var ch Charge
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		return nil, fmt.Errorf("failed to decode charge: %w", err)
	}
	return &ch, nil
}

// CallUncle instantly derails a goal that is in the red (safebuf <= 0).
// It charges the pledge amount and inserts the post-derail respite into the graph.
func (c *HTTPClient) CallUncle(ctx context.Context, goalSlug string) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/uncleme.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodPost, apiURL, strings.NewReader(""), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("failed to call uncle: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var goal Goal
	if err := json.Unmarshal(body, &goal); err != nil {
		return nil, fmt.Errorf("failed to decode goal: %w", err)
	}
	return &goal, nil
}

// FetchGoal fetches a single goal by slug.
func (c *HTTPClient) FetchGoal(ctx context.Context, goalSlug string) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("goal not found: %s", goalSlug)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var goal Goal
	if err := json.NewDecoder(resp.Body).Decode(&goal); err != nil {
		return nil, fmt.Errorf("failed to decode goal: %w", err)
	}
	if goal.Username == "" {
		goal.Username = c.config.Username
	}

	return &goal, nil
}

// FetchGoalWithDatapoints fetches goal details including recent datapoints.
func (c *HTTPClient) FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s&datapoints=true",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goal details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var goal Goal
	if err := json.NewDecoder(resp.Body).Decode(&goal); err != nil {
		return nil, fmt.Errorf("failed to decode goal details: %w", err)
	}
	if goal.Username == "" {
		goal.Username = c.config.Username
	}

	return &goal, nil
}

// FetchGoalRawJSON fetches a goal and returns the raw JSON response.
// This preserves all fields from the API, not just the ones defined in the Goal struct.
func (c *HTTPClient) FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	if includeDatapoints {
		apiURL += "&datapoints=true"
	}

	resp, err := c.doRequest(ctx, http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("goal not found: %s", goalSlug)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return json.RawMessage(body), nil
}

// CreateGoal creates a new goal for the user.
// Requires slug, title, goal_type, gunits, and exactly 2 of 3: goaldate, goalval, rate.
func (c *HTTPClient) CreateGoal(ctx context.Context, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals.json",
		c.baseURL(), c.config.Username)

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("slug", slug)
	data.Set("title", title)
	data.Set("goal_type", goalType)
	data.Set("gunits", gunits)
	data.Set("goaldate", goaldate)
	data.Set("goalval", goalval)
	data.Set("rate", rate)

	resp, err := c.doRequest(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("failed to create goal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var goal Goal
	if err := json.NewDecoder(resp.Body).Decode(&goal); err != nil {
		return nil, fmt.Errorf("failed to decode created goal: %w", err)
	}
	if goal.Username == "" {
		goal.Username = c.config.Username
	}

	return &goal, nil
}

// UpdateGoalDeadline updates the deadline (seconds from midnight) for a goal.
// The deadline parameter is undocumented in the official API but is supported:
// https://forum.beeminder.com/t/api-deadline/10666
func (c *HTTPClient) UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error) {
	escapedSlug := url.PathEscape(goalSlug)
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json",
		c.baseURL(), c.config.Username, escapedSlug)

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("deadline", fmt.Sprintf("%d", deadline))

	resp, err := c.doRequest(ctx, http.MethodPut, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("failed to update goal deadline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var goal Goal
	if err := json.NewDecoder(resp.Body).Decode(&goal); err != nil {
		return nil, fmt.Errorf("failed to decode updated goal: %w", err)
	}
	if goal.Username == "" {
		goal.Username = c.config.Username
	}

	return &goal, nil
}

// RefreshGoal forces a fetch of autodata and graph refresh for a goal.
// Returns true if the goal was queued for refresh, false if not.
func (c *HTTPClient) RefreshGoal(ctx context.Context, goalSlug string) (bool, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/refresh_graph.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodGet, apiURL, nil, "")
	if err != nil {
		return false, fmt.Errorf("failed to refresh goal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result bool
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode refresh result: %w", err)
	}

	return result, nil
}

func isGoalNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "goal not found") || strings.Contains(errStr, "status 404")
}

func (c *MultiClient) FetchGoals(ctx context.Context) ([]Goal, error) {
	var allGoals []Goal
	seen := make(map[string]struct{})
	for _, client := range c.clients {
		goals, err := client.FetchGoals(ctx)
		if err != nil {
			return nil, err
		}
		for _, goal := range goals {
			if _, exists := seen[goal.Slug]; exists {
				continue
			}
			seen[goal.Slug] = struct{}{}
			allGoals = append(allGoals, goal)
		}
	}
	return allGoals, nil
}

func (c *MultiClient) FetchGoal(ctx context.Context, goalSlug string) (*Goal, error) {
	var lastErr error
	for _, client := range c.clients {
		goal, err := client.FetchGoal(ctx, goalSlug)
		if err == nil {
			return goal, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return nil, lastErr
}

func (c *MultiClient) FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error) {
	var lastErr error
	for _, client := range c.clients {
		goal, err := client.FetchGoalWithDatapoints(ctx, goalSlug)
		if err == nil {
			return goal, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return nil, lastErr
}

func (c *MultiClient) FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error) {
	var lastErr error
	for _, client := range c.clients {
		raw, err := client.FetchGoalRawJSON(ctx, goalSlug, includeDatapoints)
		if err == nil {
			return raw, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return nil, lastErr
}

func (c *MultiClient) GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error) {
	var lastErr error
	for _, client := range c.clients {
		value, err := client.GetLastDatapointValue(ctx, goalSlug)
		if err == nil {
			return value, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return 0, lastErr
}

func (c *MultiClient) CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) error {
	var lastErr error
	for _, client := range c.clients {
		err := client.CreateDatapoint(ctx, goalSlug, timestamp, value, comment, requestid)
		if err == nil {
			return nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return lastErr
}

func (c *MultiClient) CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) error {
	var lastErr error
	for _, client := range c.clients {
		err := client.CreateDatapointWithDaystamp(ctx, goalSlug, timestamp, daystamp, value, comment, requestid)
		if err == nil {
			return nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return lastErr
}

func (c *MultiClient) CreateCharge(ctx context.Context, amount float64, note string, dryrun bool) (*Charge, error) {
	if len(c.clients) == 0 {
		return nil, fmt.Errorf("no configured clients")
	}
	return c.clients[0].CreateCharge(ctx, amount, note, dryrun)
}

func (c *MultiClient) CreateGoal(ctx context.Context, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
	if len(c.clients) == 0 {
		return nil, fmt.Errorf("no configured clients")
	}
	return c.clients[0].CreateGoal(ctx, slug, title, goalType, gunits, goaldate, goalval, rate)
}

func (c *MultiClient) CallUncle(ctx context.Context, goalSlug string) (*Goal, error) {
	var lastErr error
	for _, client := range c.clients {
		goal, err := client.CallUncle(ctx, goalSlug)
		if err == nil {
			return goal, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return nil, lastErr
}

func (c *MultiClient) UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error) {
	var lastErr error
	for _, client := range c.clients {
		goal, err := client.UpdateGoalDeadline(ctx, goalSlug, deadline)
		if err == nil {
			return goal, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return nil, lastErr
}

func (c *MultiClient) RefreshGoal(ctx context.Context, goalSlug string) (bool, error) {
	var lastErr error
	for _, client := range c.clients {
		queued, err := client.RefreshGoal(ctx, goalSlug)
		if err == nil {
			return queued, nil
		}
		lastErr = err
		if isGoalNotFoundError(err) {
			continue
		}
	}
	return false, lastErr
}
