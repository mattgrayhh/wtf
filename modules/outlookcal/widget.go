// Package outlookcal shows upcoming Outlook calendar events, read through the
// Composio CLI.
package outlookcal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/view"
)

const fetchTimeout = 60 * time.Second

// Widget is the container for the Outlook calendar module
type Widget struct {
	view.TextWidget

	settings *Settings

	mu        sync.Mutex
	events    []event
	err       error
	fetchedAt time.Time
	fetching  bool
}

// NewWidget creates a new instance of the widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	widget := &Widget{
		TextWidget: view.NewTextWidget(tviewApp, redrawChan, nil, settings.Common),

		settings: settings,
	}
	widget.View.SetWrap(false)
	return widget
}

// Refresh redraws the agenda, fetching from Outlook when the data is stale
func (widget *Widget) Refresh() {
	widget.mu.Lock()
	stale := time.Since(widget.fetchedAt) >= widget.settings.fetchInterval && !widget.fetching
	if stale {
		widget.fetching = true
	}
	widget.mu.Unlock()

	if stale {
		widget.fetch()
	}

	widget.Redraw(widget.content)
}

func (widget *Widget) fetch() {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	now := time.Now().In(widget.settings.location)
	from := startOfDay(now)
	to := from.AddDate(0, 0, widget.settings.days)
	events, err := fetchEvents(ctx, widget.settings, from, to)

	widget.mu.Lock()
	defer widget.mu.Unlock()
	widget.fetching = false
	widget.fetchedAt = time.Now()
	if err != nil {
		widget.err = err
		return
	}
	widget.err = nil
	widget.events = events
}

func (widget *Widget) content() (string, string, bool) {
	widget.mu.Lock()
	defer widget.mu.Unlock()

	title := widget.CommonSettings().Title
	if widget.err != nil && widget.events == nil {
		return title, "[red]" + tview.Escape(widget.err.Error()), true
	}
	if widget.fetchedAt.IsZero() {
		return title, "[gray]Loading…", false
	}

	now := time.Now().In(widget.settings.location)
	return title, renderAgenda(widget.events, now, widget.settings), false
}

// renderAgenda lists events by day. Past events are dimmed, the current one
// is green and the next one says how soon it starts.
func renderAgenda(events []event, now time.Time, s *Settings) string {
	var b strings.Builder
	today := startOfDay(now)
	nextMarked := false

	for d := 0; d < s.days; d++ {
		day := today.AddDate(0, 0, d)
		dayEnd := day.AddDate(0, 0, 1)

		var todays []event
		for _, e := range events {
			if e.Canceled && !s.showCanceled || e.Declined && !s.showDeclined {
				continue
			}
			if e.Start.Before(dayEnd) && e.End.After(day) || e.Start.Equal(day) {
				todays = append(todays, e)
			}
		}

		if d > 0 {
			b.WriteString("\n")
		}
		b.WriteString(dayHeading(day, d))
		b.WriteString("\n")

		if len(todays) == 0 {
			b.WriteString("  [gray]Nothing scheduled[-]\n")
			continue
		}

		for _, e := range todays {
			b.WriteString(eventLine(e, now, s, &nextMarked))
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func dayHeading(day time.Time, offset int) string {
	label := day.Format("Monday, Jan 2")
	switch offset {
	case 0:
		label = "Today · " + day.Format("Mon Jan 2")
	case 1:
		label = "Tomorrow · " + day.Format("Mon Jan 2")
	}
	return fmt.Sprintf("[::b]%s[::-]", label)
}

func eventLine(e event, now time.Time, s *Settings, nextMarked *bool) string {
	when := "All day"
	if !e.AllDay {
		when = clock(e.Start, s.hourFormat)
	}

	subject := tview.Escape(e.Subject)
	if e.Canceled {
		subject = "[::s]" + subject + "[::-]"
	}

	color, marker, suffix := "white", " ", ""
	switch {
	case e.AllDay:
		color = "#a98bf0"
	case !e.End.After(now):
		color = "gray"
	case !e.Start.After(now):
		color, marker = "#5fc27e", "▶"
		suffix = fmt.Sprintf(" [gray]until %s[-]", clock(e.End, s.hourFormat))
	case !*nextMarked:
		*nextMarked = true
		marker = "›"
		if until := e.Start.Sub(now); until < 3*time.Hour {
			suffix = fmt.Sprintf(" [#f1a14e]in %s[-]", humanize(until))
		}
	}
	if e.ShowAs == "free" || e.ShowAs == "tentative" {
		if color == "white" {
			color = "#9a9a9a"
		}
		subject = "[::i]" + subject + "[::-]"
	}

	return fmt.Sprintf("[%s]%s %8s  %s[-]%s", color, marker, when, subject, suffix)
}

func clock(t time.Time, hourFormat string) string {
	if hourFormat == "24" {
		return t.Format("15:04")
	}
	return t.Format("3:04 PM")
}

func humanize(d time.Duration) string {
	d = d.Round(time.Minute)
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
