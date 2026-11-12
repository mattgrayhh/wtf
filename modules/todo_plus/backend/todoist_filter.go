package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// filterPrefix marks a project ID that is really a Todoist filter query.
const filterPrefix = "filter:"

const todoistAPI = "https://api.todoist.com/api/v1"

// todoistFilter is a saved query, shown as its own project.
type todoistFilter struct {
	Name  string
	Query string
}

// parseFilters reads the `filters` setting, whose items are either a query
// string or a map with `name` and `query`.
func parseFilters(raw []interface{}) []todoistFilter {
	var filters []todoistFilter
	for _, item := range raw {
		switch v := item.(type) {
		case string:
			filters = append(filters, todoistFilter{Name: v, Query: v})
		case map[string]interface{}:
			query := fmt.Sprint(v["query"])
			name, _ := v["name"].(string)
			if name == "" {
				name = query
			}
			filters = append(filters, todoistFilter{Name: name, Query: query})
		}
	}
	return filters
}

type todoistTask struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	Priority   int    `json:"priority"` // 4 is p1, the highest
	ChildOrder int    `json:"child_order"`
	Checked    bool   `json:"checked"`
	Due        *struct {
		Date string `json:"date"` // "2026-11-12" or "2026-11-12T15:00:00"
	} `json:"due"`
}

// filterTasks runs a Todoist filter query and returns the matching tasks,
// most important first: priority, then due date, then Todoist's own order.
func (todo *Todoist) filterTasks(query string, limit int) ([]Task, error) {
	var all []todoistTask
	cursor := ""
	for {
		params := url.Values{"query": {query}, "limit": {"200"}}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		req, err := http.NewRequest(http.MethodGet, todoistAPI+"/tasks/filter?"+params.Encode(), http.NoBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+todo.apiKey)

		resp, err := todo.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		var page struct {
			Results    []todoistTask `json:"results"`
			NextCursor string        `json:"next_cursor"`
		}
		err = json.NewDecoder(resp.Body).Decode(&page)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("todoist filter %q: %s", query, resp.Status)
		}
		if err != nil {
			return nil, err
		}

		all = append(all, page.Results...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		da, db := dueDate(a), dueDate(b)
		if da != db {
			// Undated tasks go last
			return db == "" || (da != "" && da < db)
		}
		return a.ChildOrder < b.ChildOrder
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	now := time.Now()
	tasks := make([]Task, 0, len(all))
	for _, t := range all {
		label, overdue := dueLabel(dueDate(t), now)
		tasks = append(tasks, Task{
			ID:        t.ID,
			Completed: t.Checked,
			Name:      t.Content,
			Priority:  5 - t.Priority,
			Due:       label,
			Overdue:   overdue,
		})
	}
	return tasks, nil
}

func dueDate(t todoistTask) string {
	if t.Due == nil {
		return ""
	}
	return t.Due.Date
}

// dueLabel turns a Todoist due date into "today", "3:00 PM", "tomorrow",
// "Fri" or "Nov 20", and reports whether it has passed.
func dueLabel(due string, now time.Time) (string, bool) {
	if due == "" {
		return "", false
	}

	hasTime := strings.Contains(due, "T")
	var when time.Time
	var err error
	if hasTime {
		when, err = time.ParseInLocation("2006-01-02T15:04:05", strings.TrimSuffix(due, "Z"), now.Location())
	} else {
		when, err = time.ParseInLocation("2006-01-02", due, now.Location())
	}
	if err != nil {
		return due, false
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day := time.Date(when.Year(), when.Month(), when.Day(), 0, 0, 0, 0, now.Location())
	days := int(day.Sub(today).Hours() / 24)

	overdue := days < 0 || (hasTime && when.Before(now))

	switch {
	case days == 0 && hasTime:
		return when.Format("3:04 PM"), overdue
	case days == 0:
		return "today", false
	case days == 1:
		return "tomorrow", false
	case days == -1:
		return "yesterday", true
	case days > 1 && days < 7:
		return when.Format("Mon"), false
	default:
		return when.Format("Jan 2"), overdue
	}
}

func filterIndex(id string) (int, bool) {
	if !strings.HasPrefix(id, filterPrefix) {
		return 0, false
	}
	i, err := strconv.Atoi(strings.TrimPrefix(id, filterPrefix))
	return i, err == nil
}
