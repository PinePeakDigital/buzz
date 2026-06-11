package main

// Goal-grid geometry. The Browse grid lays goals out in fixed-size cells; these
// constants and gridLayout are the single source of truth for its shape.
// Rendering (grid.go), scroll math (utils.go), and navigation/hit-testing
// (handlers.go) all read from here, so the cell-size and chrome dimensions live
// in one place instead of being copy-pasted with a "must match" comment.
const (
	// gridCellWidth is the minimum terminal columns one cell occupies
	// (~16 content + 2 border + 2 horizontal padding).
	gridCellWidth = 20
	// gridCellHeight is the terminal rows one cell occupies (3 content + 1
	// spacing).
	gridCellHeight = 4
	// gridChromeRows is the rows reserved for the header and footer, excluded
	// from the scrollable viewport when counting how many cell-rows are visible.
	gridChromeRows = 4
	// gridHeaderRows is the header height (title + blank line), used to offset a
	// mouse-click Y coordinate into a grid row.
	gridHeaderRows = 2
)

// gridGeometry is the Browse grid's layout for a given terminal size and goal
// count. Construct it with gridLayout.
type gridGeometry struct {
	cols        int // columns that fit the width (>= 1)
	totalRows   int // cell-rows needed to show every goal
	visibleRows int // cell-rows visible at once given the height (>= 1)
}

// gridLayout computes the grid geometry for the given terminal width/height and
// number of goals.
func gridLayout(width, height, goalCount int) gridGeometry {
	cols := calculateColumns(width)
	return gridGeometry{
		cols:        cols,
		totalRows:   (goalCount + cols - 1) / cols,
		visibleRows: max(1, (height-gridChromeRows)/gridCellHeight),
	}
}

// calculateColumns returns how many cells fit across the given terminal width,
// always at least 1.
func calculateColumns(width int) int {
	return max(1, width/gridCellWidth)
}
