package backend

import (
	"testing"
	"time"
)

func TestDueLabel(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	tests := []struct {
		due     string
		label   string
		overdue bool
	}{
		{"", "", false},
		{"2026-11-12", "today", false},
		{"2026-11-12T09:00:00", "9:00 AM", true},
		{"2026-11-12T15:30:00", "3:30 PM", false},
		{"2026-11-13", "tomorrow", false},
		{"2026-11-11", "yesterday", true},
		{"2026-11-02", "Nov 2", true},
		{"2026-11-16", "Mon", false},
		{"2026-12-25", "Dec 25", false},
	}
	for _, tt := range tests {
		label, overdue := dueLabel(tt.due, now)
		if label != tt.label || overdue != tt.overdue {
			t.Errorf("dueLabel(%q) = %q, %v; want %q, %v", tt.due, label, overdue, tt.label, tt.overdue)
		}
	}
}

func TestParseFilters(t *testing.T) {
	f := parseFilters([]interface{}{"today", map[string]interface{}{"name": "Top", "query": "p1"}})
	if len(f) != 2 || f[0].Name != "today" || f[1].Name != "Top" || f[1].Query != "p1" {
		t.Fatalf("got %+v", f)
	}
	if i, ok := filterIndex("filter:1"); !ok || i != 1 {
		t.Fatal("filterIndex")
	}
	if _, ok := filterIndex("12345"); ok {
		t.Fatal("project ID read as filter")
	}
}
