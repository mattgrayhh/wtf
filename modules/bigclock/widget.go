// Package bigclock draws a large clock as an image, so it scales to any
// panel size in terminals with kitty graphics support.
package bigclock

import (
	"image"
	"math"
	"time"

	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/charting"
	"github.com/wtfutil/wtf/view"
)

// Widget is the container for the clock module
type Widget struct {
	*view.ImageWidget

	settings *Settings
	shown    string // the minute on screen, so we only redraw when it changes
}

// NewWidget creates a new instance of the widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	return &Widget{
		ImageWidget: view.NewImageWidget(tviewApp, redrawChan, nil, settings.Common),
		settings:    settings,
	}
}

// Refresh redraws the clock when the minute changes
func (widget *Widget) Refresh() {
	now := time.Now().In(widget.settings.location)
	minute := now.Format("2006-01-02 15:04")
	if minute == widget.shown {
		return
	}
	widget.shown = minute

	s := widget.settings
	widget.SetRenderer(widget.CommonSettings().Title, func(w, h int, scale float64) (image.Image, error) {
		return render(now, s, w, h, scale), nil
	})
}

var (
	dateColor = charting.RGB("#b4b9c4")
	ampmColor = charting.RGB("#7c8290")
)

func render(now time.Time, s *Settings, width, height int, scale float64) image.Image {
	c := charting.NewCanvas(width, height)
	w, h := float64(width), float64(height)
	pad := 8 * scale

	clock, ampm := now.Format("15:04"), ""
	if s.hourFormat != "24" {
		clock, ampm = now.Format("3:04"), now.Format("PM")
	}
	date := now.Format(s.dateFormat)

	timeColor, ok := charting.ParseColor(s.color)
	if !ok {
		timeColor = charting.RGB("#eef0f4")
	}

	// Size the time to fill the width, leaving room for AM/PM and the date.
	avail := w - 2*pad
	big := h * 0.5
	for i := 0; i < 40; i++ {
		total := charting.TextWidth(clock, big, charting.Bold)
		if ampm != "" {
			total += big*0.12 + charting.TextWidth(ampm, big*0.3, charting.Medium)
		}
		if total <= avail {
			break
		}
		big *= 0.94
	}
	small := math.Min(big*0.3, h*0.13)
	dateSize := charting.Fit(date, math.Min(big*0.26, h*0.12), avail, charting.Medium)

	block := big*0.74 + dateSize*1.9
	top := (h-block)/2 + big*0.74

	tw := charting.TextWidth(clock, big, charting.Bold)
	aw := 0.0
	if ampm != "" {
		aw = big*0.12 + charting.TextWidth(ampm, small, charting.Medium)
	}
	x := (w - tw - aw) / 2
	c.Text(clock, x, top, big, charting.Bold, timeColor)
	if ampm != "" {
		c.Text(ampm, x+tw+big*0.12, top, small, charting.Medium, ampmColor)
	}
	c.TextCentered(date, w/2, top+dateSize*1.9, dateSize, charting.Medium, dateColor)

	return c.Img
}
