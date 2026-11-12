// Package charting renders line, area and bar charts as images for
// ImageWidget panels. Backgrounds are transparent so charts sit on the
// terminal's own background colour.
package charting

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// Chart kinds
const (
	Line = "line"
	Area = "area"
	Bar  = "bar"
)

// Palette is used for series without their own colour.
var Palette = []string{"#4e9af1", "#f1a14e", "#5fc27e", "#e86fa8", "#a98bf0", "#4fc6c6", "#e8d36f"}

// Series is one set of values plotted against the shared labels.
type Series struct {
	Name      string
	Values    []float64 // NaN leaves a gap
	Color     string    // hex, e.g. "#4e9af1"; empty picks from Palette
	Kind      string    // overrides Spec.Kind for this series
	Secondary bool      // plot against the right-hand axis
}

// Spec describes a chart.
type Spec struct {
	Labels []string // x-axis labels, one per value
	Series []Series
	Kind   string // default kind for all series: Line, Area or Bar

	TextColor color.Color // axis labels and legend; defaults to light grey
	GridColor color.Color // defaults to a faint grey

	YMin, YMax        *float64 // pin the primary axis
	Y2Min, Y2Max      *float64 // pin the secondary axis
	YFormat, Y2Format string   // fmt verbs for tick labels, e.g. "%.0f°"

	Legend   bool
	Scale    float64 // multiplies line widths and text size, for HiDPI screens
	FontSize float64 // text height in pixels at Scale 1; defaults to 11
}

// Render draws the chart at exactly width x height pixels.
func Render(spec Spec, width, height int) (image.Image, error) {
	n := 0
	for _, s := range spec.Series {
		n = max(n, len(s.Values))
	}
	if n == 0 {
		return nil, errors.New("no data to chart")
	}

	scale := spec.Scale
	if scale <= 0 {
		scale = 1
	}
	fontPx := spec.FontSize
	if fontPx <= 0 {
		fontPx = 11
	}
	fontPx *= scale

	if width < int(fontPx*6) || height < int(fontPx*4) {
		return nil, errors.New("panel too small for a chart")
	}

	face, err := loadFace(fontPx)
	if err != nil {
		return nil, err
	}

	c := &canvas{
		img:   image.NewRGBA(image.Rect(0, 0, width, height)),
		face:  face,
		scale: scale,
		text:  orDefault(spec.TextColor, color.NRGBA{0xb0, 0xb0, 0xb0, 0xff}),
		grid:  orDefault(spec.GridColor, color.NRGBA{0x80, 0x80, 0x80, 0x48}),
	}

	primary := newAxis(spec, false, spec.YFormat)
	secondary := newAxis(spec, true, spec.Y2Format)

	// Lay out the plot area around legend and axis labels.
	pad := 4 * scale
	lineH := fontPx * 1.35
	plot := rect{x0: pad, y0: pad + lineH/2, x1: float64(width) - pad, y1: float64(height) - pad - lineH}
	if spec.Legend {
		plot.y0 += lineH
	}

	ticks := int(math.Max(2, math.Min(6, (plot.y1-plot.y0)/(fontPx*2.6))))
	primary.fit(ticks)
	secondary.fit(ticks)
	if primary.used {
		plot.x0 += c.maxWidth(primary.labels()) + pad*1.5
	}
	if secondary.used {
		plot.x1 -= c.maxWidth(secondary.labels()) + pad*1.5
	}
	if plot.x1-plot.x0 < 10 || plot.y1-plot.y0 < 10 {
		return nil, errors.New("panel too small for a chart")
	}

	slot := (plot.x1 - plot.x0) / float64(n)
	xAt := func(i int) float64 { return plot.x0 + slot*(float64(i)+0.5) }

	// Grid and y labels: the primary axis owns the grid when present.
	gridAxis := primary
	if !primary.used {
		gridAxis = secondary
	}
	for _, t := range gridAxis.ticks {
		y := gridAxis.y(t, plot)
		c.fillRect(plot.x0, y-scale/2, plot.x1, y+scale/2, c.grid)
	}
	if primary.used {
		for _, t := range primary.ticks {
			label := primary.format(t)
			c.drawText(label, plot.x0-pad*1.5-c.width(label), primary.y(t, plot)+fontPx*0.35, c.text)
		}
	}
	if secondary.used {
		for _, t := range secondary.ticks {
			c.drawText(secondary.format(t), plot.x1+pad*1.5, secondary.y(t, plot)+fontPx*0.35, c.text)
		}
	}

	// X labels, thinned so they don't collide.
	if len(spec.Labels) > 0 {
		step := 1
		widest := c.maxWidth(spec.Labels) + fontPx
		for float64(step)*slot < widest {
			step++
		}
		for i := 0; i < n && i < len(spec.Labels); i += step {
			label := spec.Labels[i]
			c.drawText(label, xAt(i)-c.width(label)/2, float64(height)-pad-fontPx*0.25, c.text)
		}
	}

	// Series, back to front: area fills, then bars, then lines.
	runs := func(s Series, ax *axis, each func([]point)) {
		var run []point
		for j, v := range s.Values {
			if math.IsNaN(v) {
				if len(run) > 0 {
					each(run)
				}
				run = nil
				continue
			}
			run = append(run, point{xAt(j), ax.y(v, plot)})
		}
		if len(run) > 0 {
			each(run)
		}
	}
	axisFor := func(s Series) *axis {
		if s.Secondary {
			return secondary
		}
		return primary
	}

	for i, s := range spec.Series {
		if kindOf(s, spec) != Area {
			continue
		}
		ax := axisFor(s)
		fill := seriesColor(s.Color, i)
		fill.A = 0x38
		base := ax.y(math.Max(ax.min, math.Min(0, ax.max)), plot)
		runs(s, ax, func(run []point) { c.fillArea(run, base, fill) })
	}

	bars := 0
	for _, s := range spec.Series {
		if kindOf(s, spec) == Bar {
			bars++
		}
	}
	barIndex := 0
	for i, s := range spec.Series {
		if kindOf(s, spec) != Bar {
			continue
		}
		ax := axisFor(s)
		col := seriesColor(s.Color, i)
		col.A = 0xc8

		group := slot * 0.8
		w := group / float64(bars)
		inset := math.Min(scale, w*0.1)
		base := ax.y(math.Max(ax.min, math.Min(0, ax.max)), plot)
		for j, v := range s.Values {
			if math.IsNaN(v) {
				continue
			}
			x := xAt(j) - group/2 + w*float64(barIndex)
			y := ax.y(v, plot)
			c.fillRect(x+inset, math.Min(y, base), x+w-inset, math.Max(y, base), col)
		}
		barIndex++
	}

	for i, s := range spec.Series {
		if kindOf(s, spec) == Bar {
			continue
		}
		col := seriesColor(s.Color, i)
		runs(s, axisFor(s), func(run []point) { c.strokeLine(run, 2*scale, col) })
	}

	if spec.Legend {
		x := plot.x0
		y := pad + fontPx
		for i, s := range spec.Series {
			if s.Name == "" {
				continue
			}
			col := seriesColor(s.Color, i)
			c.fillRect(x, y-fontPx*0.55, x+fontPx*0.9, y-fontPx*0.15, col)
			x += fontPx * 1.3
			c.drawText(s.Name, x, y, c.text)
			x += c.width(s.Name) + fontPx*1.2
		}
	}

	return c.img, nil
}

/* -------------------- Axes -------------------- */

type axis struct {
	used     bool
	min, max float64
	ticks    []float64
	verb     string
	pinLo    bool
	pinHi    bool
	decimals int
}

func newAxis(spec Spec, secondary bool, verb string) *axis {
	a := &axis{verb: verb, min: math.Inf(1), max: math.Inf(-1)}
	lo, hi := spec.YMin, spec.YMax
	if secondary {
		lo, hi = spec.Y2Min, spec.Y2Max
	}

	for _, s := range spec.Series {
		if s.Secondary != secondary {
			continue
		}
		a.used = true
		if kindOf(s, spec) != Line {
			// Bars and areas are measured from zero
			a.min, a.max = math.Min(a.min, 0), math.Max(a.max, 0)
		}
		for _, v := range s.Values {
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				a.min, a.max = math.Min(a.min, v), math.Max(a.max, v)
			}
		}
	}
	if math.IsInf(a.min, 0) {
		a.min, a.max = 0, 1
	}
	if lo != nil {
		a.min, a.pinLo = *lo, true
	}
	if hi != nil {
		a.max, a.pinHi = *hi, true
	}
	if a.max <= a.min {
		a.max = a.min + 1
	}
	return a
}

// fit picks round tick values and widens the range to land on them.
func (a *axis) fit(count int) {
	step := niceStep((a.max - a.min) / float64(count))
	if !a.pinLo {
		a.min = math.Floor(a.min/step) * step
	}
	if !a.pinHi {
		a.max = math.Ceil(a.max/step) * step
	}
	a.decimals = max(0, -int(math.Floor(math.Log10(step))))
	a.ticks = nil
	for t := math.Ceil(a.min/step) * step; t <= a.max+step*1e-9; t += step {
		a.ticks = append(a.ticks, t)
	}
}

func (a *axis) y(v float64, r rect) float64 {
	return r.y1 - (v-a.min)/(a.max-a.min)*(r.y1-r.y0)
}

func (a *axis) labels() []string {
	out := make([]string, len(a.ticks))
	for i, t := range a.ticks {
		out[i] = a.format(t)
	}
	return out
}

func (a *axis) format(v float64) string {
	if a.verb != "" {
		return fmt.Sprintf(a.verb, v)
	}
	return compact(v, a.decimals)
}

func niceStep(raw float64) float64 {
	if raw <= 0 || math.IsNaN(raw) {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	switch f := raw / mag; {
	case f <= 1:
		return mag
	case f <= 2:
		return 2 * mag
	case f <= 5:
		return 5 * mag
	default:
		return 10 * mag
	}
}

// compact formats numbers like 1.2k and 3.4M for narrow axes.
func compact(f float64, decimals int) string {
	a := math.Abs(f)
	switch {
	case a >= 1e9:
		return trim(f/1e9) + "B"
	case a >= 1e6:
		return trim(f/1e6) + "M"
	case a >= 1e4:
		return trim(f/1e3) + "k"
	default:
		return strconv.FormatFloat(f, 'f', min(decimals, 3), 64)
	}
}

func trim(f float64) string {
	return strings.TrimSuffix(strconv.FormatFloat(f, 'f', 1, 64), ".0")
}

func kindOf(s Series, spec Spec) string {
	if s.Kind != "" {
		return s.Kind
	}
	if spec.Kind != "" {
		return spec.Kind
	}
	return Line
}

/* -------------------- Drawing -------------------- */

type point struct{ x, y float64 }

type rect struct{ x0, y0, x1, y1 float64 }

type canvas struct {
	img   *image.RGBA
	face  font.Face
	scale float64
	text  color.NRGBA
	grid  color.NRGBA
}

func (c *canvas) fill(r *vector.Rasterizer, col color.NRGBA) {
	r.Draw(c.img, c.img.Bounds(), image.NewUniform(col), image.Point{})
}

func (c *canvas) newRasterizer() *vector.Rasterizer {
	b := c.img.Bounds()
	r := vector.NewRasterizer(b.Dx(), b.Dy())
	r.DrawOp = draw.Over
	return r
}

func (c *canvas) fillRect(x0, y0, x1, y1 float64, col color.NRGBA) {
	if x1 <= x0 || y1 <= y0 {
		return
	}
	r := c.newRasterizer()
	r.MoveTo(float32(x0), float32(y0))
	r.LineTo(float32(x1), float32(y0))
	r.LineTo(float32(x1), float32(y1))
	r.LineTo(float32(x0), float32(y1))
	r.ClosePath()
	c.fill(r, col)
}

func (c *canvas) fillArea(pts []point, baseY float64, col color.NRGBA) {
	r := c.newRasterizer()
	r.MoveTo(float32(pts[0].x), float32(baseY))
	for _, p := range pts {
		r.LineTo(float32(p.x), float32(p.y))
	}
	r.LineTo(float32(pts[len(pts)-1].x), float32(baseY))
	r.ClosePath()
	c.fill(r, col)
}

// strokeLine draws a polyline with round joins. Each segment is a quad and
// each vertex a disc; all are wound the same way so overlaps don't cancel.
func (c *canvas) strokeLine(pts []point, width float64, col color.NRGBA) {
	r := c.newRasterizer()
	half := width / 2

	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		dx, dy := b.x-a.x, b.y-a.y
		l := math.Hypot(dx, dy)
		if l == 0 {
			continue
		}
		nx, ny := -dy/l*half, dx/l*half
		quad := []point{{a.x + nx, a.y + ny}, {a.x - nx, a.y - ny}, {b.x - nx, b.y - ny}, {b.x + nx, b.y + ny}}
		addPolygon(r, quad)
	}
	for _, p := range pts {
		const sides = 12
		disc := make([]point, sides)
		for k := range disc {
			t := float64(k) / sides * 2 * math.Pi
			disc[k] = point{p.x + half*math.Cos(t), p.y + half*math.Sin(t)}
		}
		addPolygon(r, disc)
	}
	c.fill(r, col)
}

// addPolygon adds a closed path, reversing it if needed so every shape has
// the same (positive) winding.
func addPolygon(r *vector.Rasterizer, pts []point) {
	area := 0.0
	for i := range pts {
		j := (i + 1) % len(pts)
		area += pts[i].x*pts[j].y - pts[j].x*pts[i].y
	}
	if area < 0 {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	r.MoveTo(float32(pts[0].x), float32(pts[0].y))
	for _, p := range pts[1:] {
		r.LineTo(float32(p.x), float32(p.y))
	}
	r.ClosePath()
}

func (c *canvas) drawText(s string, x, y float64, col color.NRGBA) {
	d := font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(col),
		Face: c.face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)},
	}
	d.DrawString(s)
}

func (c *canvas) width(s string) float64 {
	return float64(font.MeasureString(c.face, s)) / 64
}

func (c *canvas) maxWidth(ss []string) float64 {
	w := 0.0
	for _, s := range ss {
		w = math.Max(w, c.width(s))
	}
	return w
}

/* -------------------- Fonts and colours -------------------- */

var (
	fontOnce sync.Once
	fontData *opentype.Font
	fontErr  error
	facesMu  sync.Mutex
	faces    = map[float64]font.Face{}
)

func loadFace(px float64) (font.Face, error) {
	fontOnce.Do(func() { fontData, fontErr = opentype.Parse(goregular.TTF) })
	if fontErr != nil {
		return nil, fontErr
	}

	facesMu.Lock()
	defer facesMu.Unlock()
	if f, ok := faces[px]; ok {
		return f, nil
	}
	f, err := opentype.NewFace(fontData, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	faces[px] = f
	return f, nil
}

// ParseColor reads "#rrggbb", "rrggbb" or "#rgb".
func ParseColor(hex string) (color.NRGBA, bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return color.NRGBA{}, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.NRGBA{}, false
	}
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}, true
}

func seriesColor(hex string, i int) color.NRGBA {
	if c, ok := ParseColor(hex); ok {
		return c
	}
	c, _ := ParseColor(Palette[i%len(Palette)])
	return c
}

func orDefault(c color.Color, fallback color.NRGBA) color.NRGBA {
	if c == nil {
		return fallback
	}
	return color.NRGBAModel.Convert(c).(color.NRGBA)
}
