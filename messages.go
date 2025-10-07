package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// goalsLoadedMsg is sent when goals are loaded from the API
type goalsLoadedMsg struct {
	goals []Goal
	err   error
}

// refreshTickMsg is sent when it's time to refresh data
type refreshTickMsg struct{}

// datapointSubmittedMsg is sent when a datapoint submission completes
type datapointSubmittedMsg struct {
	err error
}

// goalDetailsLoadedMsg is sent when goal details with datapoints are loaded
type goalDetailsLoadedMsg struct {
	goal *Goal
	err  error
}

// loadGoalsCmd fetches goals from Beeminder API
func loadGoalsCmd(config *Config) tea.Cmd {
	return func() tea.Msg {
		goals, err := FetchGoals(config)
		if err != nil {
			return goalsLoadedMsg{err: err}
		}
		SortGoals(goals)
		return goalsLoadedMsg{goals: goals}
	}
}

// refreshTickCmd creates a command that sends refresh tick messages at intervals
func refreshTickCmd() tea.Cmd {
	return tea.Tick(time.Minute*5, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

// submitDatapointCmd submits a datapoint to Beeminder API
func submitDatapointCmd(config *Config, goalSlug, timestamp, value, comment string) tea.Cmd {
	return func() tea.Msg {
		err := CreateDatapoint(config, goalSlug, timestamp, value, comment)
		return datapointSubmittedMsg{err: err}
	}
}

// loadGoalDetailsCmd fetches detailed goal information including datapoints
func loadGoalDetailsCmd(config *Config, goalSlug string) tea.Cmd {
	return func() tea.Msg {
		goal, err := FetchGoalWithDatapoints(config, goalSlug)
		return goalDetailsLoadedMsg{goal: goal, err: err}
	}
}
