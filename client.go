package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	// FetchUserTimezone returns the IANA timezone configured on the user's
	// Beeminder account (e.g. "America/New_York"), or an empty string if the
	// account has none set.
	FetchUserTimezone(ctx context.Context) (string, error)
	// APIRequest performs a raw, authenticated request against the Beeminder
	// API. path is relative to the API root (e.g. "users/me.json"); a leading
	// slash is optional. The configured auth_token is added automatically.
	// params are sent as query parameters for GET/DELETE and as a urlencoded
	// form body otherwise; url.Values preserves repeated keys. A non-2xx status
	// is NOT returned as an error — callers inspect the returned status code and
	// body themselves.
	APIRequest(ctx context.Context, method, path string, params url.Values) (int, []byte, error)
	FetchGoal(ctx context.Context, goalSlug string) (*Goal, error)
	FetchGoalWithDatapoints(ctx context.Context, goalSlug string) (*Goal, error)
	FetchGoalRawJSON(ctx context.Context, goalSlug string, includeDatapoints bool) (json.RawMessage, error)
	GetLastDatapointValue(ctx context.Context, goalSlug string) (float64, error)
	CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) (*Datapoint, error)
	CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) (*Datapoint, error)
	CreateCharge(ctx context.Context, amount float64, note string, dryrun bool) (*Charge, error)
	CreateGoal(ctx context.Context, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error)
	CallUncle(ctx context.Context, goalSlug string) (*Goal, error)
	RatchetGoal(ctx context.Context, goalSlug string, ratchet int) (*Goal, error)
	UpdateGoalDeadline(ctx context.Context, goalSlug string, deadline int) (*Goal, error)
	RefreshGoal(ctx context.Context, goalSlug string) (bool, error)
}

// HTTPClient is the HTTP-backed Client. Construct with NewHTTPClient.
type HTTPClient struct {
	config *Config
	http   *http.Client
}

// NewHTTPClient returns a Client backed by net/http using credentials in config.
// The returned value can be assigned to a Client interface variable; downstream
// code should depend on Client, not *HTTPClient.
func NewHTTPClient(config *Config) *HTTPClient {
	return &HTTPClient{
		config: config,
		http:   &http.Client{Timeout: httpClientTimeout},
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

	return goals, nil
}

// FetchUserTimezone fetches the IANA timezone configured on the user's
// Beeminder account from the user endpoint. Returns an empty string (no error)
// if the account has no timezone set.
func (c *HTTPClient) FetchUserTimezone(ctx context.Context) (string, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s.json?auth_token=%s",
		c.baseURL(), c.config.Username, c.config.AuthToken)

	resp, err := c.doRequest(ctx, http.MethodGet, apiURL, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode user: %w", err)
	}

	return result.Timezone, nil
}

// APIRequest performs a raw, authenticated request against the Beeminder API.
// See the Client interface for the contract. The auth_token is injected into
// the query string for GET/DELETE and into the form body for methods that
// carry one (POST/PUT/PATCH).
func (c *HTTPClient) APIRequest(ctx context.Context, method, path string, params url.Values) (int, []byte, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/%s", c.baseURL(), strings.TrimPrefix(path, "/")))
	if err != nil {
		return 0, nil, fmt.Errorf("invalid API path: %w", err)
	}

	// Start from any query already embedded in path, then layer caller params on
	// top (caller wins per key). Parsing rather than string-concatenating means
	// a path like "...?auth_token=x" can't smuggle in a duplicate auth_token.
	values := u.Query()
	for k, vs := range params {
		values.Del(k)
		for _, v := range vs {
			values.Add(k, v)
		}
	}
	// Set auth_token last so the stored credential always wins over anything in
	// the path or params — honoring the "injected automatically" contract.
	values.Set("auth_token", c.config.AuthToken)

	var reqBody io.Reader
	contentType := ""
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		// Carry everything (including auth_token) in the form body.
		u.RawQuery = ""
		reqBody = strings.NewReader(values.Encode())
		contentType = "application/x-www-form-urlencoded"
	default:
		// GET/DELETE: carry everything in the query string.
		u.RawQuery = values.Encode()
	}

	resp, err := c.doRequest(ctx, method, u.String(), reqBody, contentType)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	return resp.StatusCode, body, nil
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

// CreateDatapoint submits a new datapoint to a Beeminder goal and returns the
// created datapoint (which includes its server-assigned ID).
func (c *HTTPClient) CreateDatapoint(ctx context.Context, goalSlug, timestamp, value, comment, requestid string) (*Datapoint, error) {
	return c.CreateDatapointWithDaystamp(ctx, goalSlug, timestamp, "", value, comment, requestid)
}

// CreateDatapointWithDaystamp submits a new datapoint with optional daystamp and
// returns the created datapoint. If daystamp is provided (format YYYYMMDD), it is
// used instead of timestamp.
func (c *HTTPClient) CreateDatapointWithDaystamp(ctx context.Context, goalSlug, timestamp, daystamp, value, comment, requestid string) (*Datapoint, error) {
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
		return nil, fmt.Errorf("failed to create datapoint: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read create-datapoint response: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var dp Datapoint
	if err := json.Unmarshal(body, &dp); err != nil {
		return nil, fmt.Errorf("failed to decode created datapoint: %w", err)
	}
	return &dp, nil
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

// RatchetGoal removes safety buffer from a goal, leaving at most `ratchet` days
// of buffer between today and the bright red line. Beeminder ignores requests
// that would *add* buffer, so a goal already at or below `ratchet` days is left
// unchanged — this can only ever tighten a goal, never loosen it.
func (c *HTTPClient) RatchetGoal(ctx context.Context, goalSlug string, ratchet int) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/ratchet.json",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug))

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("ratchet", fmt.Sprintf("%d", ratchet))

	resp, err := c.doRequest(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("failed to ratchet goal: %w", err)
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

	return &goal, nil
}

// FetchGoalsWithDatapoints fetches the user's goals and populates the recent
// datapoints for each one. Datapoints are fetched concurrently with a bounded
// worker pool to keep the N+1 round trips fast for users with many goals.
// A per-goal fetch failure is non-fatal: that goal is returned without
// datapoints rather than aborting the whole operation.
func (c *HTTPClient) FetchGoalsWithDatapoints(ctx context.Context) ([]Goal, error) {
	goals, err := c.FetchGoals(ctx)
	if err != nil {
		return nil, err
	}

	const maxWorkers = 5
	goalsChan := make(chan int, maxWorkers)
	var wg sync.WaitGroup

	for w := 0; w < maxWorkers && w < len(goals); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range goalsChan {
				goalWithDatapoints, err := c.FetchGoalWithDatapoints(ctx, goals[i].Slug)
				if err != nil {
					// Leave this goal without datapoints rather than
					// failing the entire review.
					continue
				}
				goals[i].Datapoints = goalWithDatapoints.Datapoints
				// Retain the per-goal fields the bulk list endpoint omits
				// but the review chart needs (road, graph window, cumulative
				// flag, good side).
				goals[i].Roadall = goalWithDatapoints.Roadall
				goals[i].Tmin = goalWithDatapoints.Tmin
				goals[i].Tmax = goalWithDatapoints.Tmax
				goals[i].Kyoom = goalWithDatapoints.Kyoom
				goals[i].Yaw = goalWithDatapoints.Yaw
			}
		}()
	}

	for i := range goals {
		goalsChan <- i
	}
	close(goalsChan)
	wg.Wait()

	return goals, nil
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
