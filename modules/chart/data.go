package chart

import (
	"encoding/csv"
	"errors"
	"math"
	"strconv"
	"strings"
)

// table is parsed chart data: one label per row and one column per series.
type table struct {
	labels []string
	names  []string
	values [][]float64 // values[series][row]
}

// parseTable reads comma-, tab- or space-separated rows.
//
//	value                    a single unlabelled series
//	label,value[,value...]   labelled rows, one series per value column
//
// A first row whose value columns aren't numbers is a header naming the series.
// Blank lines and lines starting with # are skipped. Empty cells or "-" leave gaps.
func parseTable(text string) (*table, error) {
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if t := strings.TrimSpace(l); t != "" && !strings.HasPrefix(t, "#") {
			lines = append(lines, t)
		}
	}
	if len(lines) == 0 {
		return nil, errors.New("no data")
	}

	rows, err := split(lines)
	if err != nil {
		return nil, err
	}

	// Does the first column hold labels? Only if it's not numeric in every row.
	labelled := false
	for _, r := range rows {
		if len(r) > 1 && !isNumber(r[0]) {
			labelled = true
			break
		}
	}
	first := 0
	if labelled {
		first = 1
	}

	t := &table{}

	// Header row: any non-numeric value cell.
	header := false
	for _, cell := range rows[0][first:] {
		if !isNumber(cell) && cell != "" && cell != "-" {
			header = true
			break
		}
	}
	if header {
		t.names = append(t.names, rows[0][first:]...)
		rows = rows[1:]
	}
	if len(rows) == 0 {
		return nil, errors.New("no data rows")
	}

	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r)-first)
	}
	if cols <= 0 {
		return nil, errors.New("no value columns")
	}
	for len(t.names) < cols {
		t.names = append(t.names, "")
	}
	t.names = t.names[:cols]

	t.values = make([][]float64, cols)
	for _, r := range rows {
		if labelled {
			t.labels = append(t.labels, r[0])
		}
		for c := 0; c < cols; c++ {
			v := math.NaN()
			if i := first + c; i < len(r) {
				if f, err := strconv.ParseFloat(strings.TrimSuffix(r[i], "%"), 64); err == nil {
					v = f
				}
			}
			t.values[c] = append(t.values[c], v)
		}
	}

	return t, nil
}

// tail keeps the last n rows.
func (t *table) tail(n int) {
	if n <= 0 {
		return
	}
	if len(t.labels) > n {
		t.labels = t.labels[len(t.labels)-n:]
	}
	for i, v := range t.values {
		if len(v) > n {
			t.values[i] = v[len(v)-n:]
		}
	}
}

func split(lines []string) ([][]string, error) {
	sep := ' '
	switch {
	case strings.Contains(lines[0], "\t"):
		sep = '\t'
	case strings.Contains(lines[0], ","):
		sep = ','
	}

	if sep == ' ' {
		rows := make([][]string, len(lines))
		for i, l := range lines {
			rows[i] = strings.Fields(l)
		}
		return rows, nil
	}

	r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
	r.Comma = sep
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	return r.ReadAll()
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	return err == nil
}
