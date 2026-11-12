package outlookcal

import (
	"strings"
	"testing"
	"time"
)

func TestRenderAgenda(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 11, 12, 12, 10, 0, 0, loc)
	at := func(h, m int) time.Time { return time.Date(2026, 11, 12, h, m, 0, 0, loc) }
	s := &Settings{days: 2, hourFormat: "12", location: loc}

	events := []event{
		{Subject: "Huddle", Start: at(9, 0), End: at(9, 15)},
		{Subject: "Webinar", Start: at(12, 0), End: at(13, 0)},
		{Subject: "Canceled: Sync", Start: at(13, 30), End: at(14, 0), Canceled: true},
		{Subject: "1:1", Start: at(14, 0), End: at(14, 30)},
		{Subject: "Standup", Start: at(9, 0).AddDate(0, 0, 1), End: at(9, 15).AddDate(0, 0, 1)},
	}
	out := renderAgenda(events, now, s)

	for _, want := range []string{"Today · Thu Nov 12", "[gray]   9:00 AM  Huddle", "▶ 12:00 PM  Webinar", "until 1:00 PM", "›  2:00 PM  1:1", "in 1h 50m", "Tomorrow · Fri Nov 13", "Standup"} {
		if !strings.Contains(out, want) {
			t.Errorf("agenda missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Canceled") {
		t.Error("canceled event shown")
	}

	empty := renderAgenda(nil, now, s)
	if strings.Count(empty, "Nothing scheduled") != 2 {
		t.Errorf("expected two empty days:\n%s", empty)
	}
}

func TestParseGraphTime(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	got, err := parseGraphTime(graphTime{DateTime: "2026-11-12T13:00:00.0000000", TimeZone: "UTC"}, ny)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hour() != 8 || got.Location() != ny {
		t.Fatalf("got %v", got)
	}
}
