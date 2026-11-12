package view

import "testing"

func TestContainCells(t *testing.T) {
	tests := []struct {
		name               string
		imgW, imgH         int
		maxCols, maxRows   int
		wantCols, wantRows int
	}{
		// 8x16 cells: a 40x20 panel is 320x320 px
		{"square image in square panel", 100, 100, 40, 20, 40, 20},
		{"wide image is limited by width", 400, 100, 40, 20, 40, 5},
		{"tall image is limited by height", 100, 400, 40, 20, 10, 20},
		{"never zero", 1000, 1, 40, 20, 40, 1},
		{"bad size fills panel", 0, 0, 40, 20, 40, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows := containCells(tt.imgW, tt.imgH, tt.maxCols, tt.maxRows, 8, 16)
			if cols != tt.wantCols || rows != tt.wantRows {
				t.Errorf("got %dx%d, want %dx%d", cols, rows, tt.wantCols, tt.wantRows)
			}
		})
	}
}
