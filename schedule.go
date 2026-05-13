package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"
)

// timeSlot represents a time of day with goals scheduled at that time
type timeSlot struct {
	hour   int      // 0-23
	minute int      // 0-59
	goals  []string // goal slugs at this time
}

// handleScheduleCommand displays a visual representation of goal deadline distribution throughout a 24-hour day
func handleScheduleCommand() {
	// Load config and goals
	_, _, goals, err := loadConfigAndGoals()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}

	if len(goals) == 0 {
		fmt.Println("No goals found.")
		return
	}

	// Extract time-of-day from each goal's deadline
	timeSlots := extractTimeSlots(goals)

	// Generate hourly density data
	hourCounts := make([]int, 24)
	for _, slot := range timeSlots {
		hourCounts[slot.hour] += len(slot.goals)
	}

	// Display hourly density overview
	displayHourlyDensity(hourCounts)

	// Display detailed timeline
	displayTimeline(timeSlots)

	// Check for updates and display message if available
	fmt.Print(getUpdateMessage())
}

// extractTimeSlots extracts time-of-day from all goal deadlines and groups them
func extractTimeSlots(goals []Goal) []timeSlot {
	// Map to group goals by their time of day (hour:minute)
	slotMap := make(map[string]*timeSlot)

	for _, goal := range goals {
		// Convert losedate to local time and extract hour/minute
		t := time.Unix(goal.Losedate, 0).In(time.Local)
		hour := t.Hour()
		minute := t.Minute()

		// Create key for this time slot
		key := fmt.Sprintf("%02d:%02d", hour, minute)

		// Add goal to this time slot
		if slot, exists := slotMap[key]; exists {
			slot.goals = append(slot.goals, goal.Slug)
		} else {
			slotMap[key] = &timeSlot{
				hour:   hour,
				minute: minute,
				goals:  []string{goal.Slug},
			}
		}
	}

	// Convert map to sorted slice
	var slots []timeSlot
	for _, slot := range slotMap {
		slots = append(slots, *slot)
	}

	// Sort by time (hour, then minute)
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].hour != slots[j].hour {
			return slots[i].hour < slots[j].hour
		}
		return slots[i].minute < slots[j].minute
	})

	return slots
}

// displayHourlyDensity displays a compact bar chart showing goals per hour
func displayHourlyDensity(hourCounts []int) {
	const (
		hoursPerDay    = 24
		charsPerHour   = 3 // Each hour uses 3 chars: 2 for display (bar/label) + 1 space
		densityLineLen = hoursPerDay * charsPerHour
	)

	fmt.Println("HOURLY DENSITY")

	// Find max count for scaling
	maxCount := 0
	for _, count := range hourCounts {
		if count > maxCount {
			maxCount = count
		}
	}

	// If no goals, just display empty chart
	if maxCount == 0 {
		fmt.Println("No goals scheduled.")
		return
	}

	// Define bar characters (from empty to full)
	bars := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Build the bar chart line
	var barLine strings.Builder

	for hour := 0; hour < hoursPerDay; hour++ {
		count := hourCounts[hour]
		var bar rune
		if count == 0 {
			bar = bars[0]
		} else {
			// Scale to bar height (1-8)
			barIndex := (count * (len(bars) - 1)) / maxCount
			if barIndex == 0 && count > 0 {
				barIndex = 1 // Ensure at least one bar level for any count
			}
			bar = bars[barIndex]
		}
		// Write the bar twice for 2-char width per hour, plus a space
		barLine.WriteRune(bar)
		barLine.WriteRune(bar)
		barLine.WriteRune(' ')
	}

	fmt.Println(barLine.String())

	// Build hour labels (show all 24 hours with spacing)
	// Use color to de-emphasize hours with no counts (if colors enabled)
	colorProfile := lipgloss.ColorProfile()
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	var labelLine strings.Builder

	for hour := 0; hour < hoursPerDay; hour++ {
		label := fmt.Sprintf("%02d", hour)
		if hourCounts[hour] == 0 && colorProfile != termenv.Ascii {
			// Dim hours with no counts using gray color
			labelLine.WriteString(dimStyle.Render(label) + " ")
		} else {
			labelLine.WriteString(label + " ")
		}
	}
	fmt.Println(labelLine.String())

	// Build axis line with markers for each hour
	axisRunes := make([]rune, densityLineLen)
	for i := range axisRunes {
		axisRunes[i] = ' '
	}

	firstPos, lastPos := -1, -1
	for hour := 0; hour < hoursPerDay; hour++ {
		pos := hour * charsPerHour
		if pos >= len(axisRunes) {
			continue
		}
		// Use ┼ if hour has counts (extends down), ┴ if no counts (only extends up)
		if hourCounts[hour] > 0 {
			axisRunes[pos] = '┼'
		} else {
			axisRunes[pos] = '┴'
		}
		if firstPos == -1 {
			firstPos = pos
		}
		lastPos = pos
	}

	// Draw horizontal line segments between first and last marker
	if firstPos != -1 && lastPos != -1 && firstPos <= lastPos {
		for i := firstPos + 1; i < lastPos; i++ {
			if axisRunes[i] == ' ' {
				axisRunes[i] = '─'
			}
		}
	}

	fmt.Println(string(axisRunes))

	// Build count labels (show counts for all hours with goals)
	countLine := make([]rune, densityLineLen)
	for i := range countLine {
		countLine[i] = ' '
	}

	for hour := 0; hour < hoursPerDay; hour++ {
		count := hourCounts[hour]
		if count > 0 {
			var label string
			if count > 99 {
				label = " ∞" // Use infinity symbol for 99+, with leading space
			} else {
				label = fmt.Sprintf("%-2d", count) // Left-align in 2-char space
			}
			pos := hour * charsPerHour
			// Write the label runes into the count line, guarding against overflow
			for i, r := range label {
				if pos+i >= len(countLine) {
					break
				}
				countLine[pos+i] = r
			}
		}
	}
	fmt.Println(string(countLine))
	fmt.Println()
}

// displayTimeline displays a vertical timeline listing all goals grouped by deadline time
func displayTimeline(slots []timeSlot) {
	fmt.Println("TIMELINE")
	fmt.Println("────────────────────────────────────────────────")

	// Define styles for timeline elements (disabled if --no-color)
	colorProfile := lipgloss.ColorProfile()
	timeStyle := lipgloss.NewStyle()
	treeStyle := lipgloss.NewStyle()
	if colorProfile != termenv.Ascii {
		timeStyle = timeStyle.Foreground(lipgloss.Color("6")) // Cyan for time labels
		treeStyle = treeStyle.Foreground(lipgloss.Color("8")) // Gray for tree structure (├─ and │)
	}

	for _, slot := range slots {
		timeStr := fmt.Sprintf("%02d:%02d", slot.hour, slot.minute)

		// Build the full line and wrap it to terminal width, indenting wrapped lines
		// Color the time and tree separately using lipgloss
		prefix := timeStyle.Render(timeStr) + " " + treeStyle.Render("├─") + " "
		// Visual width of prefix: "HH:MM ├─ " = 9 characters (ANSI codes have zero width)
		const prefixVisualWidth = 9

		// Determine terminal width; fallback to 80 if unavailable
		width := 80
		fd := uintptr(os.Stdout.Fd())
		if term.IsTerminal(fd) {
			if w, _, err := term.GetSize(fd); err == nil && w > 0 {
				width = w
			}
		}

		// Simple wrapping: break on commas before exceeding width
		available := width
		if prefixVisualWidth < available {
			available -= prefixVisualWidth
		} else {
			available = 10 // minimal safety width
		}

		var line strings.Builder
		line.WriteString(prefix)
		current := 0
		for _, goal := range slot.goals {
			// Determine separator
			sep := ""
			if current > 0 {
				// Add comma+space before this goal if there's already content on this line
				sep = ", "
			}
			chunk := sep + goal
			if current+len(chunk) > available && current > 0 {
				// Always add trailing comma when wrapping (more content follows)
				line.WriteString(",")
				fmt.Println(line.String())
				// start new wrapped line with indent matching prefix width
				// Use vertical line to show continuation
				line.Reset()
				// Time column width (5 chars) + space + vertical continuation (styled)
				line.WriteString("      " + treeStyle.Render("│") + "  ")
				// Add the goal that didn't fit
				line.WriteString(goal)
				current = len(goal)
				continue
			}
			line.WriteString(chunk)
			current += len(chunk)
		}
		fmt.Println(line.String())
	}
}
