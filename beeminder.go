package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// Goal represents a Beeminder goal with relevant fields
type Goal struct {
	Slug     string  `json:"slug"`
	Title    string  `json:"title"`
	Losedate int64   `json:"losedate"`
	Pledge   float64 `json:"pledge"`
	Safebuf  int     `json:"safebuf"`
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

// FormatDueDate formats the losedate timestamp into a readable string
func FormatDueDate(losedate int64) string {
	t := time.Unix(losedate, 0)
	now := time.Now()
	
	// Calculate days until due
	duration := t.Sub(now)
	days := int(duration.Hours() / 24)
	
	if days < 0 {
		return "OVERDUE"
	}
	if days == 0 {
		return "TODAY"
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
