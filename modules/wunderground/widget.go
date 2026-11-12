// Package wunderground shows current conditions from a Weather Underground
// personal weather station beside a multi-day outlook, drawn as an image.
package wunderground

import (
	"context"
	"image"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/view"
)

const fetchTimeout = 30 * time.Second

// Widget is the container for the weather card module
type Widget struct {
	*view.ImageWidget

	settings *Settings
	location *time.Location
}

// NewWidget creates a new instance of the widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	loc := time.Local
	if settings.timezone != "" {
		if l, err := time.LoadLocation(settings.timezone); err == nil {
			loc = l
		}
	}

	return &Widget{
		ImageWidget: view.NewImageWidget(tviewApp, redrawChan, nil, settings.Common),

		settings: settings,
		location: loc,
	}
}

// Refresh fetches the latest observation and forecast
func (widget *Widget) Refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	var (
		r   *report
		err error
	)
	if widget.settings.apiKey != "" && widget.settings.stationID != "" {
		r, err = fetchWU(ctx, widget.settings, widget.location)
	} else {
		r, err = fetchOpenMeteo(ctx, widget.settings, widget.location)
	}
	if err != nil {
		widget.SetMessage(widget.title(nil), err.Error())
		return
	}

	widget.SetRenderer(widget.title(r), func(w, h int, scale float64) (image.Image, error) {
		return renderCard(r, w, h, scale, time.Now().In(widget.location))
	})
}

// title reads like "St. Simons Island · KGASAINT62 · 3:42 PM".
func (widget *Widget) title(r *report) string {
	parts := []string{}
	if p := widget.settings.place; p != "" {
		parts = append(parts, p)
	} else if t := widget.CommonSettings().Title; t != "" {
		parts = append(parts, t)
	}
	if r != nil {
		parts = append(parts, r.Now.Source)
		if !r.Now.Observed.IsZero() {
			parts = append(parts, r.Now.Observed.Format("3:04 PM"))
		}
	}
	return strings.Join(parts, " · ")
}
