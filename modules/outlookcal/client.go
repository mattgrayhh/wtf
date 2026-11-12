package outlookcal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// event is one calendar occurrence, with times in the display time zone.
type event struct {
	Subject  string
	Location string
	Start    time.Time
	End      time.Time
	AllDay   bool
	ShowAs   string // free, tentative, busy, oof, workingElsewhere
	Canceled bool
	Declined bool
	Online   bool
}

type graphTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type graphEvent struct {
	Subject         string    `json:"subject"`
	Start           graphTime `json:"start"`
	End             graphTime `json:"end"`
	IsAllDay        bool      `json:"isAllDay"`
	IsCancelled     bool      `json:"isCancelled"`
	ShowAs          string    `json:"showAs"`
	IsOnlineMeeting bool      `json:"isOnlineMeeting"`
	Location        struct {
		DisplayName string `json:"displayName"`
	} `json:"location"`
	ResponseStatus struct {
		Response string `json:"response"`
	} `json:"responseStatus"`
}

// fetchEvents reads the calendar view between from and to through the
// Composio CLI, which handles the Microsoft sign-in.
func fetchEvents(ctx context.Context, s *Settings, from, to time.Time) ([]event, error) {
	bin, err := composioPath(s.composio)
	if err != nil {
		return nil, err
	}

	args, _ := json.Marshal(map[string]interface{}{
		"start_datetime": from.UTC().Format("2006-01-02T15:04:05Z"),
		"end_datetime":   to.UTC().Format("2006-01-02T15:04:05Z"),
		"timezone":       s.location.String(),
		"top":            100,
	})
	cmdArgs := []string{"execute", "OUTLOOK_GET_CALENDAR_VIEW", "-d", string(args)}
	if s.account != "" {
		cmdArgs = append(cmdArgs, "--account", s.account)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("composio: %s", firstLine(msg))
	}

	var res struct {
		Successful bool   `json:"successful"`
		Error      string `json:"error"`
		Data       struct {
			Value []graphEvent `json:"value"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		return nil, fmt.Errorf("composio: unexpected output: %w", err)
	}
	if !res.Successful {
		if res.Error == "" {
			res.Error = "request failed"
		}
		return nil, fmt.Errorf("outlook: %s", firstLine(res.Error))
	}

	events := make([]event, 0, len(res.Data.Value))
	for _, g := range res.Data.Value {
		start, err1 := parseGraphTime(g.Start, s.location)
		end, err2 := parseGraphTime(g.End, s.location)
		if err1 != nil || err2 != nil {
			continue
		}
		subject := strings.TrimSpace(g.Subject)
		canceled := g.IsCancelled || strings.HasPrefix(strings.ToLower(subject), "canceled:") || strings.HasPrefix(strings.ToLower(subject), "cancelled:")
		events = append(events, event{
			Subject:  subject,
			Location: g.Location.DisplayName,
			Start:    start,
			End:      end,
			AllDay:   g.IsAllDay,
			ShowAs:   g.ShowAs,
			Canceled: canceled,
			Declined: strings.EqualFold(g.ResponseStatus.Response, "declined"),
			Online:   g.IsOnlineMeeting,
		})
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].AllDay != events[j].AllDay {
			return events[i].AllDay
		}
		return events[i].Start.Before(events[j].Start)
	})
	return events, nil
}

// parseGraphTime reads Graph's "2026-11-12T13:00:00.0000000" in its stated zone.
func parseGraphTime(t graphTime, display *time.Location) (time.Time, error) {
	loc := display
	if t.TimeZone != "" {
		if l, err := time.LoadLocation(t.TimeZone); err == nil {
			loc = l
		} else if strings.EqualFold(t.TimeZone, "UTC") {
			loc = time.UTC
		}
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04:05.9999999", t.DateTime, loc)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.In(display), nil
}

func composioPath(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	if p, err := exec.LookPath("composio"); err == nil {
		return p, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".local", "bin", "composio")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("composio CLI not found; install it or set `composio` in this module's config")
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
