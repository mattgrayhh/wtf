// Package chart draws line, area and bar charts from CSV-style data produced
// by a command, a file or a URL.
package chart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/charting"
	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/view"
)

const (
	fetchTimeout = 30 * time.Second
	maxDataSize  = 8 << 20
)

// Widget is the container for the chart module
type Widget struct {
	*view.ImageWidget

	settings *Settings
}

// NewWidget creates a new instance of the widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	return &Widget{
		ImageWidget: view.NewImageWidget(tviewApp, redrawChan, nil, settings.Common),

		settings: settings,
	}
}

// Refresh reloads the data and redraws the chart
func (widget *Widget) Refresh() {
	title := widget.CommonSettings().Title

	text, err := widget.load()
	if err != nil {
		widget.SetMessage(title, err.Error())
		return
	}

	data, err := parseTable(text)
	if err != nil {
		widget.SetMessage(title, err.Error())
		return
	}
	data.tail(widget.settings.maxPoints)

	spec := widget.spec(data)
	widget.SetRenderer(title, func(w, h int, scale float64) (image.Image, error) {
		spec.Scale = scale
		return charting.Render(spec, w, h)
	})
}

func (widget *Widget) spec(data *table) charting.Spec {
	s := widget.settings

	spec := charting.Spec{
		Labels:   data.labels,
		Kind:     s.kind,
		YMin:     s.yMin,
		YMax:     s.yMax,
		YFormat:  s.yFormat,
		FontSize: s.fontSize,
	}

	for i, values := range data.values {
		series := charting.Series{Name: data.names[i], Values: values}
		if i < len(s.colors) {
			series.Color = s.colors[i]
		}
		series.Secondary = series.Name != "" && slices.Contains(s.secondary, series.Name)
		spec.Series = append(spec.Series, series)
	}

	switch strings.ToLower(s.legend) {
	case "true", "yes", "on":
		spec.Legend = true
	case "auto", "":
		spec.Legend = len(spec.Series) > 1 && data.names[0] != ""
	}

	if c, ok := charting.ParseColor(s.textColor); ok {
		spec.TextColor = c
	} else {
		spec.TextColor = color.NRGBA{0xb0, 0xb0, 0xb0, 0xff}
	}

	return spec
}

func (widget *Widget) load() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	s := widget.settings
	switch {
	case s.cmd != "":
		var stderr bytes.Buffer
		c := exec.CommandContext(ctx, s.cmd, s.args...)
		c.Stderr = &stderr
		out, err := c.Output()
		if err != nil {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				return "", fmt.Errorf("%s: %s", s.cmd, msg)
			}
			return "", fmt.Errorf("%s: %w", s.cmd, err)
		}
		return string(out), nil

	case s.url != "":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, http.NoBody)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("%s: %s", s.url, resp.Status)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxDataSize))
		return string(body), err

	case s.path != "":
		path, err := utils.ExpandHomeDir(s.path)
		if err != nil {
			return "", err
		}
		body, err := os.ReadFile(path)
		return string(body), err

	default:
		return "", errors.New("set cmd, path or url in this module's config")
	}
}
