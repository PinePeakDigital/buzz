package main

import "testing"

func TestUrgencyFor(t *testing.T) {
	tests := []struct {
		name    string
		safebuf int
		want    Urgency
		label   string
	}{
		{"negative buffer is overdue", -1, UrgencyOverdue, "red"},
		{"zero buffer is overdue", 0, UrgencyOverdue, "red"},
		{"safebuf 1 is due today", 1, UrgencyDueToday, "orange"},
		{"safebuf 2 is due tomorrow", 2, UrgencyDueTomorrow, "blue"},
		{"safebuf 3 is this week", 3, UrgencyThisWeek, "green"},
		{"safebuf 6 is this week", 6, UrgencyThisWeek, "green"},
		{"safebuf 7 is distant", 7, UrgencyDistant, "gray"},
		{"large buffer is distant", 100, UrgencyDistant, "gray"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UrgencyFor(tt.safebuf)
			if got != tt.want {
				t.Errorf("UrgencyFor(%d) = %v, want %v", tt.safebuf, got, tt.want)
			}
			if got.String() != tt.label {
				t.Errorf("UrgencyFor(%d).String() = %q, want %q", tt.safebuf, got.String(), tt.label)
			}
		})
	}
}

// TestUrgencyStyles spot-checks that the three style accessors return styles
// using the urgency's foreground colour. We can't compare lipgloss styles
// directly, so we verify the colour value rendered into each.
func TestUrgencyStyles(t *testing.T) {
	for _, u := range []Urgency{UrgencyOverdue, UrgencyDueToday, UrgencyDueTomorrow, UrgencyThisWeek, UrgencyDistant} {
		c := u.Color()
		if string(c) == "" {
			t.Errorf("Urgency %v Color() returned empty", u)
		}
		// TextStyle uses the urgency colour as its foreground.
		if got := u.TextStyle().GetForeground(); got != c {
			t.Errorf("Urgency %v TextStyle foreground = %v, want %v", u, got, c)
		}
		// GridCellStyle uses the urgency colour for both border and foreground.
		if got := u.GridCellStyle().GetForeground(); got != c {
			t.Errorf("Urgency %v GridCellStyle foreground = %v, want %v", u, got, c)
		}
		// HighlightedGridCellStyle keeps the urgency foreground but uses a
		// bright-white border for contrast.
		if got := u.HighlightedGridCellStyle().GetForeground(); got != c {
			t.Errorf("Urgency %v HighlightedGridCellStyle foreground = %v, want %v", u, got, c)
		}
	}
}
