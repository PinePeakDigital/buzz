package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// httpClientTimeout caps every Beeminder request so a stalled connection can't
// freeze the CLI or block a Bubble Tea Cmd indefinitely. A real cancellation
// story (per-request context, user-quit propagation) is tracked in issue #253;
// this timeout is the cheap stopgap until then.
const httpClientTimeout = 30 * time.Second

// Client is the Beeminder API seam. Production code depends on this interface;
// HTTPClient is the only adapter today, and tests use it via NewHTTPClient
// pointed at httptest. Future work (see issue #244 follow-on) can introduce a
// fake adapter so command/handler tests don't need an HTTP server.
type Client interface {
	FetchGoals() ([]Goal, error)
	FetchGoal(goalSlug string) (*Goal, error)
	FetchGoalWithDatapoints(goalSlug string) (*Goal, error)
	FetchGoalRawJSON(goalSlug string, includeDatapoints bool) (json.RawMessage, error)
	GetLastDatapointValue(goalSlug string) (float64, error)
	CreateDatapoint(goalSlug, timestamp, value, comment, requestid string) error
	CreateDatapointWithDaystamp(goalSlug, timestamp, daystamp, value, comment, requestid string) error
	CreateCharge(amount float64, note string, dryrun bool) (*Charge, error)
	CreateGoal(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error)
	CallUncle(goalSlug string) (*Goal, error)
	UpdateGoalDeadline(goalSlug string, deadline int) (*Goal, error)
	RefreshGoal(goalSlug string) (bool, error)
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

// FetchGoals fetches the user's goals from Beeminder API.
func (c *HTTPClient) FetchGoals() ([]Goal, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/goals.json?auth_token=%s",
		c.baseURL(), c.config.Username, c.config.AuthToken)

	LogRequest(c.config, "GET", url)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goals: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, url)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var goals []Goal
	if err := json.NewDecoder(resp.Body).Decode(&goals); err != nil {
		return nil, fmt.Errorf("failed to decode goals: %w", err)
	}

	return goals, nil
}

// GetLastDatapointValue fetches the last datapoint value for a goal.
func (c *HTTPClient) GetLastDatapointValue(goalSlug string) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s&skinny=true",
		c.baseURL(), c.config.Username, goalSlug, c.config.AuthToken)

	LogRequest(c.config, "GET", url)
	resp, err := c.http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch goal details: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, url)

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
func (c *HTTPClient) CreateDatapoint(goalSlug, timestamp, value, comment, requestid string) error {
	return c.CreateDatapointWithDaystamp(goalSlug, timestamp, "", value, comment, requestid)
}

// CreateDatapointWithDaystamp submits a new datapoint with optional daystamp.
// If daystamp is provided (format YYYYMMDD), it is used instead of timestamp.
func (c *HTTPClient) CreateDatapointWithDaystamp(goalSlug, timestamp, daystamp, value, comment, requestid string) error {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/datapoints.json",
		c.baseURL(), c.config.Username, goalSlug)

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

	LogRequest(c.config, "POST", apiURL)
	resp, err := c.http.Post(apiURL, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create datapoint: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateCharge creates a new charge for the authenticated user and returns it.
func (c *HTTPClient) CreateCharge(amount float64, note string, dryrun bool) (*Charge, error) {
	apiURL := fmt.Sprintf("%s/api/v1/charges.json", c.baseURL())

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("user_id", c.config.Username)
	data.Set("amount", fmt.Sprintf("%.2f", amount))
	data.Set("note", note)
	if dryrun {
		data.Set("dryrun", "true")
	}

	LogRequest(c.config, "POST", apiURL)
	resp, err := c.http.Post(apiURL, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create charge: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

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
func (c *HTTPClient) CallUncle(goalSlug string) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/uncleme.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	LogRequest(c.config, "POST", apiURL)
	resp, err := c.http.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		return nil, fmt.Errorf("failed to call uncle: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

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
func (c *HTTPClient) FetchGoal(goalSlug string) (*Goal, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	LogRequest(c.config, "GET", apiURL)
	resp, err := c.http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goal: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

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
func (c *HTTPClient) FetchGoalWithDatapoints(goalSlug string) (*Goal, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s&datapoints=true",
		c.baseURL(), c.config.Username, goalSlug, c.config.AuthToken)

	LogRequest(c.config, "GET", url)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goal details: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, url)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var goal Goal
	if err := json.NewDecoder(resp.Body).Decode(&goal); err != nil {
		return nil, fmt.Errorf("failed to decode goal details: %w", err)
	}

	return &goal, nil
}

// FetchGoalRawJSON fetches a goal and returns the raw JSON response.
// This preserves all fields from the API, not just the ones defined in the Goal struct.
func (c *HTTPClient) FetchGoalRawJSON(goalSlug string, includeDatapoints bool) (json.RawMessage, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json?auth_token=%s",
		c.baseURL(), c.config.Username, url.PathEscape(goalSlug), c.config.AuthToken)

	if includeDatapoints {
		apiURL += "&datapoints=true"
	}

	LogRequest(c.config, "GET", apiURL)
	resp, err := c.http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch goal: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

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
func (c *HTTPClient) CreateGoal(slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
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

	LogRequest(c.config, "POST", apiURL)
	resp, err := c.http.Post(apiURL, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create goal: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

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
func (c *HTTPClient) UpdateGoalDeadline(goalSlug string, deadline int) (*Goal, error) {
	escapedSlug := url.PathEscape(goalSlug)
	apiURL := fmt.Sprintf("%s/api/v1/users/%s/goals/%s.json",
		c.baseURL(), c.config.Username, escapedSlug)

	data := url.Values{}
	data.Set("auth_token", c.config.AuthToken)
	data.Set("deadline", fmt.Sprintf("%d", deadline))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	LogRequest(c.config, "PUT", apiURL)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update goal deadline: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, apiURL)

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
func (c *HTTPClient) RefreshGoal(goalSlug string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/goals/%s/refresh_graph.json?auth_token=%s",
		c.baseURL(), c.config.Username, goalSlug, c.config.AuthToken)

	LogRequest(c.config, "GET", url)
	resp, err := c.http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to refresh goal: %w", err)
	}
	defer resp.Body.Close()
	LogResponse(c.config, resp.StatusCode, url)

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result bool
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode refresh result: %w", err)
	}

	return result, nil
}
