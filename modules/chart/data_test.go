package chart

import (
	"math"
	"testing"
)

func TestParseTable(t *testing.T) {
	t.Run("bare numbers", func(t *testing.T) {
		tb, err := parseTable("1\n2\n3\n")
		if err != nil {
			t.Fatal(err)
		}
		if len(tb.labels) != 0 || len(tb.values) != 1 || len(tb.values[0]) != 3 {
			t.Fatalf("unexpected table %+v", tb)
		}
	})

	t.Run("header, labels and two series", func(t *testing.T) {
		tb, err := parseTable("month,Revenue,Costs\nJan,120,90\nFeb,135,\n")
		if err != nil {
			t.Fatal(err)
		}
		if tb.names[0] != "Revenue" || tb.names[1] != "Costs" {
			t.Errorf("names %v", tb.names)
		}
		if tb.labels[1] != "Feb" || tb.values[0][1] != 135 {
			t.Errorf("values %v labels %v", tb.values, tb.labels)
		}
		if !math.IsNaN(tb.values[1][1]) {
			t.Errorf("empty cell should be a gap, got %v", tb.values[1][1])
		}
	})

	t.Run("tabs, comments and percents", func(t *testing.T) {
		tb, err := parseTable("# disk use\n/\t42%\nData\t72%\n")
		if err != nil {
			t.Fatal(err)
		}
		if tb.labels[0] != "/" || tb.values[0][1] != 72 {
			t.Fatalf("unexpected table %+v", tb)
		}
	})

	t.Run("space separated with numeric labels", func(t *testing.T) {
		tb, err := parseTable("2024 5\n2025 7\n")
		if err != nil {
			t.Fatal(err)
		}
		// Every first column is numeric, so both columns are series
		if len(tb.values) != 2 || len(tb.labels) != 0 {
			t.Fatalf("unexpected table %+v", tb)
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := parseTable("\n# nothing\n"); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestTail(t *testing.T) {
	tb, _ := parseTable("a,1\nb,2\nc,3\n")
	tb.tail(2)
	if len(tb.labels) != 2 || tb.labels[0] != "b" || tb.values[0][0] != 2 {
		t.Fatalf("unexpected table %+v", tb)
	}
}
