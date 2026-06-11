package main

import "testing"

func TestGridLayout(t *testing.T) {
	tests := []struct {
		name                             string
		width, height, goalCount         int
		wantCols, wantTotalRows, wantVis int
	}{
		{"typical 80x24", 80, 24, 10, 4, 3, 5},         // 4 cols, ceil(10/4)=3, (24-4)/4=5
		{"single column", 20, 8, 3, 1, 3, 1},           // 1 col, 3 rows, (8-4)/4=1
		{"narrow clamps cols to 1", 5, 24, 3, 1, 3, 5}, // width<cell → 1 col
		{"no goals → zero total rows", 80, 24, 0, 4, 0, 5},
		{"short height clamps visible to 1", 80, 4, 5, 4, 2, 1}, // (4-4)/4=0 → max(1,..)=1
		{"height below chrome stays 1", 80, 3, 5, 4, 2, 1},      // (3-4)/4=0 → 1
		{"zero size stays sane", 0, 0, 0, 1, 0, 1},
		{"exact fill", 40, 12, 6, 2, 3, 2}, // 2 cols, ceil(6/2)=3, (12-4)/4=2
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gridLayout(tt.width, tt.height, tt.goalCount)
			if g.cols != tt.wantCols || g.totalRows != tt.wantTotalRows || g.visibleRows != tt.wantVis {
				t.Errorf("gridLayout(%d,%d,%d) = {cols:%d totalRows:%d visibleRows:%d}, want {cols:%d totalRows:%d visibleRows:%d}",
					tt.width, tt.height, tt.goalCount,
					g.cols, g.totalRows, g.visibleRows,
					tt.wantCols, tt.wantTotalRows, tt.wantVis)
			}
		})
	}
}
