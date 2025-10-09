package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Goal represents a Beeminder goal with relevant fields
type Goal struct {
	Slug       string      `json:"slug"`
	Title      string      `json:"title"`
	Losedate   int64       `json:"losedate"`
	Pledge     float64     `json:"pledge"`
	Safebuf    int         `json:"safebuf"`
	Limsum     string      `json:"limsum"`
	Datapoints []Datapoint `json:"datapoints,omitempty"`
}

// Datapoint represents a Beeminder datapoint
type Datapoint struct {
	ID        string  `json:"id"`
	Timestamp int64   `json:"timestamp"`
	Daystamp  string  `json:"daystamp"`
	Value     float64 `json:"value"`
	Comment   string  `json:"comment"`
}

// FetchGoals fetches the user's goals from Beeminder API
func FetchGoals(config *Config) ([]Goal, error) {
	url := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals.json?auth_token=%s",
		config.Username, config.AuthToken)

	resp, err := http.Get(url)
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

// SortGoals sorts goals by: 1. Due ascending, 2. Stakes descending, 3. Name ascending
func SortGoals(goals []Goal) {
	sort.Slice(goals, func(i, j int) bool {
		// 1. Due ascending (losedate)
		if goals[i].Losedate != goals[j].Losedate {
			return goals[i].Losedate < goals[j].Losedate
		}
		// 2. Stakes descending (pledge)
		if goals[i].Pledge != goals[j].Pledge {
			return goals[i].Pledge > goals[j].Pledge
		}
		// 3. Name alphabetical ascending (slug)
		return goals[i].Slug < goals[j].Slug
	})
}

// GetBufferColor returns the color name based on safebuf value
// 0 days buffer (safebuf < 1) = red
// 1 day buffer (safebuf < 2) = orange
// 2 days buffer (safebuf < 3) = blue
// 3-6 days (safebuf < 7) = green
// 7+ days = gray
func GetBufferColor(safebuf int) string {
	if safebuf < 1 {
		return "red"
	}
	if safebuf < 2 {
		return "orange"
	}
	if safebuf < 3 {
		return "blue"
	}
	if safebuf < 7 {
		return "green"
	}
	return "gray"
}

// ParseLimsumValue extracts the delta value from limsum string
// e.g., "+2 within 1 day" -> "2", "+1 in 3 hours" -> "1", "0 today" -> "0"
// Time formats are preserved: "+00:05 within 1 day" -> "00:05", "+1:30 in 2 hours" -> "1:30"
func ParseLimsumValue(limsum string) string {
	if limsum == "" {
		return "0"
	}
	var value string
	// Split on " within "
	parts := strings.Split(limsum, " within ")
	if len(parts) == 2 {
		value = parts[0]
	} else {
		// Split on " in "
		parts = strings.Split(limsum, " in ")
		if len(parts) == 2 {
			value = parts[0]
		} else {
			// Handle "0 today" or similar cases - extract just the number/value at the start
			fields := strings.Fields(limsum)
			if len(fields) > 0 {
				value = fields[0]
			} else {
				// If format doesn't match, return "0" as fallback
				return "0"
			}
		}
	}
	// Strip leading plus sign
	cleaned := strings.TrimPrefix(value, "+")
	// Return "0" if the cleaned value is empty
	if cleaned == "" {
		return "0"
	}
	return cleaned
}

// FormatDueDate formats the losedate timestamp into a readable string
func FormatDueDate(losedate int64) string {
	return FormatDueDateAt(losedate, time.Now())
}

// FormatDueDateAt formats the losedate timestamp relative to a given time
func FormatDueDateAt(losedate int64, now time.Time) string {
	t := time.Unix(losedate, 0)

	// Calculate duration until due
	duration := t.Sub(now)
	totalHours := duration.Hours()

	if totalHours < 0 {
		return "OVERDUE"
	}

	// If less than 1 day, show in hours or minutes
	if totalHours < 24 {
		if totalHours >= 1 {
			// Show in hours (rounded down)
			hours := int(totalHours)
			return fmt.Sprintf("%dh", hours)
		} else {
			// Show in minutes (rounded down)
			minutes := int(duration.Minutes())
			if minutes < 1 {
				return "0m"
			}
			return fmt.Sprintf("%dm", minutes)
		}
	}

	// Show in days with "d" suffix
	days := int(totalHours / 24)
	return fmt.Sprintf("%dd", days)
}

// GetLastDatapointValue fetches the last datapoint value for a goal
func GetLastDatapointValue(config *Config, goalSlug string) (float64, error) {
	url := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals/%s.json?auth_token=%s&skinny=true",
		config.Username, goalSlug, config.AuthToken)

	resp, err := http.Get(url)
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
		return 0, nil // No previous datapoints
	}

	return result.LastDatapoint.Value, nil
}

// CreateDatapoint submits a new datapoint to a Beeminder goal
func CreateDatapoint(config *Config, goalSlug, timestamp, value, comment string) error {
	url := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals/%s/datapoints.json",
		config.Username, goalSlug)

	data := fmt.Sprintf("auth_token=%s&timestamp=%s&value=%s&comment=%s",
		config.AuthToken, timestamp, value, comment)

	resp, err := http.Post(url, "application/x-www-form-urlencoded",
		strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create datapoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// FetchGoalWithDatapoints fetches goal details including recent datapoints
func FetchGoalWithDatapoints(config *Config, goalSlug string) (*Goal, error) {
	url := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals/%s.json?auth_token=%s&datapoints=true",
		config.Username, goalSlug, config.AuthToken)

	resp, err := http.Get(url)
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

// CreateGoal creates a new goal for the user
// Requires slug, title, goal_type, gunits, and exactly 2 of 3: goaldate, goalval, rate
func CreateGoal(config *Config, slug, title, goalType, gunits, goaldate, goalval, rate string) (*Goal, error) {
	apiURL := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals.json",
		config.Username)

	data := url.Values{}
	data.Set("auth_token", config.AuthToken)
	data.Set("slug", slug)
	data.Set("title", title)
	data.Set("goal_type", goalType)
	data.Set("gunits", gunits)
	data.Set("goaldate", goaldate)
	data.Set("goalval", goalval)
	data.Set("rate", rate)

	resp, err := http.Post(apiURL, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
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
