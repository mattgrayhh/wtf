// Package weatherchart charts the hourly temperature and chance of rain from
// Open-Meteo, which needs no API key.
package weatherchart

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/charting"
	"github.com/wtfutil/wtf/view"
)

const fetchTimeout = 20 * time.Second

// Widget is the container for the weather chart module
type Widget struct {
	*view.ImageWidget

	settings *Settings
	place    *place
}

// NewWidget creates a new instance of the widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	return &Widget{
		ImageWidget: view.NewImageWidget(tviewApp, redrawChan, nil, settings.Common),

		settings: settings,
	}
}

// Refresh fetches the forecast and redraws the chart
func (widget *Widget) Refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	title := widget.CommonSettings().Title

	p, err := widget.location(ctx)
	if err != nil {
		widget.SetMessage(title, err.Error())
		return
	}

	imperial := strings.EqualFold(widget.settings.units, "imperial")
	f, err := fetchForecast(ctx, p.Latitude, p.Longitude, imperial, widget.settings.hours)
	if err != nil {
		widget.SetMessage(title, err.Error())
		return
	}

	spec := widget.spec(f)
	widget.SetRenderer(widget.title(p, f), func(w, h int, scale float64) (image.Image, error) {
		spec.Scale = scale
		return charting.Render(spec, w, h)
	})
}

func (widget *Widget) location(ctx context.Context) (place, error) {
	s := widget.settings
	if s.latitude != 0 || s.longitude != 0 {
		name := s.city
		if name == "" {
			name = fmt.Sprintf("%.2f, %.2f", s.latitude, s.longitude)
		}
		return place{Name: name, Latitude: s.latitude, Longitude: s.longitude}, nil
	}
	if s.city == "" {
		return place{}, errors.New("set city, or latitude and longitude, in this module's config")
	}
	if widget.place == nil {
		p, err := geocode(ctx, s.city)
		if err != nil {
			return place{}, err
		}
		widget.place = &p
	}
	return *widget.place, nil
}

// title reads like "London  8° Clear  ↑11° ↓4°".
func (widget *Widget) title(p place, f *forecast) string {
	parts := []string{p.Name, fmt.Sprintf("%.0f°", f.Current.Temperature)}
	if c := conditions(f.Current.WeatherCode); c != "" {
		parts = append(parts, c)
	}
	if len(f.Daily.Max) > 0 && len(f.Daily.Min) > 0 {
		parts = append(parts, fmt.Sprintf("↑%.0f° ↓%.0f°", f.Daily.Max[0], f.Daily.Min[0]))
	}
	return strings.Join(parts, "  ")
}

func (widget *Widget) spec(f *forecast) charting.Spec {
	// Start at the current hour; times are local to the forecast location.
	start := 0
	now := f.Current.Time
	if len(now) >= 13 {
		now = now[:13] // "2026-11-12T15"
	}
	for i, t := range f.Hourly.Time {
		if strings.HasPrefix(t, now) {
			start = i
			break
		}
	}
	end := min(start+widget.settings.hours, len(f.Hourly.Time))

	var labels []string
	var temps, rain []float64
	for i := start; i < end; i++ {
		labels = append(labels, hourLabel(f.Hourly.Time[i]))
		temps = append(temps, valueAt(f.Hourly.Temperature, i))
		rain = append(rain, valueAt(f.Hourly.Precipitation, i))
	}

	zero, hundred := 0.0, 100.0
	spec := charting.Spec{
		Labels:   labels,
		Legend:   true,
		YFormat:  "%.0f°",
		Y2Format: "%.0f%%",
		Y2Min:    &zero,
		Y2Max:    &hundred,
		FontSize: widget.settings.fontSize,
		Series: []charting.Series{
			{Name: "Rain chance", Values: rain, Kind: charting.Bar, Secondary: true, Color: "#4e9af1"},
			{Name: "Temperature", Values: temps, Kind: charting.Line, Color: "#f1a14e"},
		},
	}

	if c, ok := charting.ParseColor(widget.settings.textColor); ok {
		spec.TextColor = c
	} else {
		spec.TextColor = color.NRGBA{0xb0, 0xb0, 0xb0, 0xff}
	}

	return spec
}

// hourLabel turns "2026-11-12T15:00" into "15", or the weekday at midnight.
func hourLabel(t string) string {
	parsed, err := time.Parse("2006-01-02T15:04", t)
	if err != nil {
		return t
	}
	if parsed.Hour() == 0 {
		return parsed.Format("Mon")
	}
	return parsed.Format("15")
}

func valueAt(values []*float64, i int) float64 {
	if i < len(values) && values[i] != nil {
		return *values[i]
	}
	return math.NaN()
}
