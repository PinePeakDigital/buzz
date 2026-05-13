package main

import "github.com/charmbracelet/lipgloss"

// Urgency expresses how close a goal is to derailing, derived from its
// `safebuf` (days of buffer). Both the threshold logic and the presentation
// choices (foreground colour, cell style) live here, so adding or shifting an
// urgency level — e.g. inserting a "yellow at 1.5 days" tier — is a one-file
// edit instead of a cross-file hunt that previously touched beeminder.go,
// styles.go, grid.go, review.go, and main.go in lockstep.

// Urgency is a closed enum of severity levels for goal deadlines.
type Urgency int

const (
	// UrgencyOverdue: safebuf < 1 — due today or already derailed (red).
	UrgencyOverdue Urgency = iota
	// UrgencyDueToday: safebuf < 2 — due tomorrow at the latest (orange).
	UrgencyDueToday
	// UrgencyDueTomorrow: safebuf < 3 — due within 2 days (blue).
	UrgencyDueTomorrow
	// UrgencyThisWeek: safebuf < 7 — due within a week (green).
	UrgencyThisWeek
	// UrgencyDistant: safebuf >= 7 — plenty of buffer (gray).
	UrgencyDistant
)

// String returns the legacy colour-name label ("red", "orange", "blue",
// "green", "gray") that the previous GetBufferColor function returned. This
// exists solely so the modal's "Buffer Color: <name>" line keeps rendering
// the same string; new code should match on the typed Urgency value instead.
func (u Urgency) String() string {
	switch u {
	case UrgencyOverdue:
		return "red"
	case UrgencyDueToday:
		return "orange"
	case UrgencyDueTomorrow:
		return "blue"
	case UrgencyThisWeek:
		return "green"
	default:
		return "gray"
	}
}

// UrgencyFor returns the urgency level for a given safebuf value.
func UrgencyFor(safebuf int) Urgency {
	switch {
	case safebuf < 1:
		return UrgencyOverdue
	case safebuf < 2:
		return UrgencyDueToday
	case safebuf < 3:
		return UrgencyDueTomorrow
	case safebuf < 7:
		return UrgencyThisWeek
	default:
		return UrgencyDistant
	}
}

// Color returns the lipgloss colour code used for this urgency level. The
// codes are ANSI palette indices: 1=red, 208=orange, 4=blue, 2=green, 8=gray.
func (u Urgency) Color() lipgloss.Color {
	switch u {
	case UrgencyOverdue:
		return lipgloss.Color("1")
	case UrgencyDueToday:
		return lipgloss.Color("208")
	case UrgencyDueTomorrow:
		return lipgloss.Color("4")
	case UrgencyThisWeek:
		return lipgloss.Color("2")
	default:
		return lipgloss.Color("8")
	}
}

// TextStyle returns a lipgloss style that only sets the foreground colour for
// this urgency. Used for inline-coloured text (list-view rows, modal details).
func (u Urgency) TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(u.Color())
}

// GridCellStyle returns the bordered cell style used in the TUI grid for an
// unselected goal. Border and text share the urgency colour.
func (u Urgency) GridCellStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(u.Color()).
		Foreground(u.Color()).
		Padding(PaddingVertical, PaddingHorizontal).
		MarginRight(GridMarginRight).
		MarginBottom(GridMarginBottom)
}

// HighlightedGridCellStyle returns the cell style for the currently-selected
// goal in the TUI grid: thick bright-white border for contrast, but the text
// retains the urgency colour so the user can still read the severity at a
// glance.
func (u Urgency) HighlightedGridCellStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("15")). // Bright white border for contrast
		Foreground(u.Color()).
		Padding(PaddingVertical, PaddingHorizontal).
		MarginRight(GridMarginRight).
		MarginBottom(GridMarginBottom)
}
