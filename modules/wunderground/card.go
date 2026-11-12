package wunderground

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/wtfutil/wtf/charting"
)

var (
	textBright = charting.RGB("#eef0f4")
	textMid    = charting.RGB("#b4b9c4")
	textDim    = charting.RGB("#7c8290")
	divider    = color.NRGBA{0x80, 0x86, 0x94, 0x40}
	trackColor = color.NRGBA{0x80, 0x86, 0x94, 0x38}
	coolColor  = charting.RGB("#4fb3f6")
	warmColor  = charting.RGB("#ff9a3c")
)

// renderCard draws current conditions beside (or above, when the panel is
// narrow) a multi-day outlook.
func renderCard(r *report, width, height int, scale float64, now time.Time) (image.Image, error) {
	if width < 120 || height < 60 {
		return nil, fmt.Errorf("panel too small")
	}
	c := charting.NewCanvas(width, height)
	w, h := float64(width), float64(height)
	pad := 10 * scale

	if w >= 1.8*h {
		split := math.Round(w * 0.42)
		drawCurrent(c, r, pad, pad, split-pad, h-pad)
		c.Rect(split, h*0.12, split+math.Max(1, scale), h*0.88, divider)
		drawOutlook(c, r, split+pad, pad, w-pad, h-pad, now)
	} else {
		split := math.Round(h * 0.5)
		drawCurrent(c, r, pad, pad, w-pad, split-pad/2)
		c.Rect(w*0.06, split, w*0.94, split+math.Max(1, scale), divider)
		drawOutlook(c, r, pad, split+pad/2, w-pad, h-pad, now)
	}
	return c.Img, nil
}

func drawCurrent(c *charting.Canvas, r *report, x0, y0, x1, y1 float64) {
	n := r.Now
	H := y1 - y0
	W := x1 - x0

	// Stats take a fixed band at the bottom; the icon, temperature and
	// description share what's left above it.
	label := H * 0.072
	value := H * 0.09
	rowH := label*1.25 + value*1.3
	statsTop := y1 - 2*rowH
	topH := statsTop - y0 - H*0.05

	icon := math.Min(topH*0.82, W*0.3)
	drawIcon(c, n.Sky, n.Night, x0, y0+(topH-icon)/2, icon)

	tx := x0 + icon + W*0.04
	temp := deg(n.Temp)
	phrase := n.Phrase
	feels := ""
	if !math.IsNaN(n.FeelsLike) && math.Abs(n.FeelsLike-n.Temp) >= 1 {
		feels = "Feels like " + deg(n.FeelsLike)
	}

	// Text block: big*0.78 + phrase*1.3 + feels*1.35, scaled to fit topH.
	big := charting.Fit(temp, topH*0.62, x1-tx, charting.Bold)
	ps := math.Min(big*0.26, topH*0.16)
	fs := ps * 0.82
	block := big*0.78 + ps*1.35
	if feels != "" {
		block += fs * 1.4
	}
	if block > topH {
		k := topH / block
		big, ps, fs = big*k, ps*k, fs*k
		block = topH
	}
	ty := y0 + (topH-block)/2 + big*0.78
	c.Text(temp, tx, ty, big, charting.Bold, textBright)
	if phrase != "" {
		ty += ps * 1.35
		c.Text(phrase, tx, ty, charting.Fit(phrase, ps, x1-tx, charting.Medium), charting.Medium, textMid)
	}
	if feels != "" {
		ty += fs * 1.4
		c.Text(feels, tx, ty, charting.Fit(feels, fs, x1-tx, charting.Regular), charting.Regular, textDim)
	}

	// Stats in a 2x2 grid along the bottom.
	stats := []struct{ label, value string }{
		{"Humidity", pct(n.Humidity)},
		{"Wind", wind(n, r.Imperial)},
		{"Rain today", rain(n.PrecipToday, r.Imperial)},
		{"Pressure", pressure(n.Pressure, r.Imperial)},
	}
	top := statsTop
	colW := W / 2
	for i, s := range stats {
		sx := x0 + float64(i%2)*colW
		sy := top + float64(i/2)*rowH
		c.Text(strings.ToUpper(s.label), sx, sy+label, charting.Fit(strings.ToUpper(s.label), label, colW-4, charting.Medium), charting.Medium, textDim)
		c.Text(s.value, sx, sy+label*1.25+value, charting.Fit(s.value, value, colW-4, charting.Medium), charting.Medium, textBright)
	}
}

func drawOutlook(c *charting.Canvas, r *report, x0, y0, x1, y1 float64, now time.Time) {
	days := r.Days
	if len(days) == 0 {
		return
	}
	H := y1 - y0
	colW := (x1 - x0) / float64(len(days))

	// The week's range, for the temperature bars.
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, d := range days {
		for _, v := range []float64{d.Low, d.High} {
			if !math.IsNaN(v) {
				lo, hi = math.Min(lo, v), math.Max(hi, v)
			}
		}
	}
	if math.IsInf(lo, 0) || hi-lo < 1 {
		lo, hi = lo-1, lo+1
	}

	today := now.Format("2006-01-02")
	labelSize := math.Min(H*0.1, colW*0.2)
	iconSize := math.Min(H*0.34, colW*0.72)
	tempSize := math.Min(H*0.12, colW*0.22)
	smallSize := math.Min(H*0.085, colW*0.17)

	for i, d := range days {
		cx := x0 + colW*(float64(i)+0.5)
		y := y0

		name := d.Date.Format("Mon")
		nameColor := textMid
		if d.Date.Format("2006-01-02") == today {
			name, nameColor = "Today", textBright
		}
		y += labelSize
		c.TextCentered(name, cx, y, labelSize, charting.Medium, nameColor)

		y += labelSize * 0.45
		drawIcon(c, d.Sky, false, cx-iconSize/2, y, iconSize)
		y += iconSize + tempSize*0.95

		highs := deg(d.High)
		lows := deg(d.Low)
		gap := tempSize * 0.35
		hw := charting.TextWidth(highs, tempSize, charting.Bold)
		lw := charting.TextWidth(lows, tempSize*0.85, charting.Regular)
		start := cx - (hw+gap+lw)/2
		c.Text(highs, start, y, tempSize, charting.Bold, textBright)
		c.Text(lows, start+hw+gap, y, tempSize*0.85, charting.Regular, textDim)

		// Range bar: where this day's low..high sits within the week.
		y += tempSize * 0.55
		barW := colW * 0.62
		bx := cx - barW/2
		bh := math.Max(3, tempSize*0.22)
		c.RoundRect(bx, y, bx+barW, y+bh, bh/2, trackColor)
		if !math.IsNaN(d.Low) && !math.IsNaN(d.High) {
			f0 := (d.Low - lo) / (hi - lo)
			f1 := (d.High - lo) / (hi - lo)
			gradientBar(c, bx+barW*f0, bx+barW*math.Max(f1, f0+0.04), y, bh, f0, math.Max(f1, f0+0.04))
		}
		y += bh

		if !math.IsNaN(d.PrecipChance) && d.PrecipChance >= 10 {
			y += smallSize * 1.5
			label := fmt.Sprintf("%.0f%%", d.PrecipChance)
			tw := charting.TextWidth(label, smallSize, charting.Medium)
			dr := smallSize * 0.32
			total := dr*2.4 + tw
			dx := cx - total/2
			drop(c, dx+dr, y-smallSize*0.36, dr)
			c.Text(label, dx+dr*2.4, y, smallSize, charting.Medium, rainColor)
		}
	}
}

// gradientBar fills a rounded bar from x0 to x1 whose colour runs from cool
// to warm across the week's temperature range (f0..f1 of it).
func gradientBar(c *charting.Canvas, x0, x1, y, h, f0, f1 float64) {
	r := h / 2
	c.Circle(x0+r, y+r, r, blend(coolColor, warmColor, f0))
	c.Circle(x1-r, y+r, r, blend(coolColor, warmColor, f1))
	step := math.Max(1, h/4)
	for x := x0 + r; x < x1-r; x += step {
		t := f0 + (f1-f0)*(x-x0)/(x1-x0)
		c.Rect(x, y, math.Min(x+step+0.5, x1-r), y+h, blend(coolColor, warmColor, t))
	}
}

// drop draws a raindrop centred at (cx, cy).
func drop(c *charting.Canvas, cx, cy, r float64) {
	pts := []charting.Point{{X: cx, Y: cy - r*1.9}}
	for k := 0; k <= 12; k++ {
		a := math.Pi*0.15 - float64(k)/12*math.Pi*1.3
		pts = append(pts, charting.Point{X: cx + r*math.Cos(a), Y: cy - r*math.Sin(a)})
	}
	c.Shapes(rainColor, pts)
}

// blend runs from a through a bright yellow-green midpoint to b, which reads
// better as a temperature scale than a straight mix (that goes grey).
func blend(a, b color.NRGBA, t float64) color.NRGBA {
	t = math.Max(0, math.Min(1, t))
	mid := color.NRGBA{0xe6, 0xd2, 0x5a, 0xff}
	mix := func(p, q color.NRGBA, u float64) color.NRGBA {
		m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*u) }
		return color.NRGBA{m(p.R, q.R), m(p.G, q.G), m(p.B, q.B), 0xff}
	}
	if t < 0.5 {
		return mix(a, mid, t*2)
	}
	return mix(mid, b, (t-0.5)*2)
}

func deg(v float64) string {
	if math.IsNaN(v) {
		return "–"
	}
	return fmt.Sprintf("%.0f°", v)
}

func pct(v float64) string {
	if math.IsNaN(v) {
		return "–"
	}
	return fmt.Sprintf("%.0f%%", v)
}

func wind(n current, imperial bool) string {
	if math.IsNaN(n.WindSpeed) {
		return "–"
	}
	unit := "km/h"
	if imperial {
		unit = "mph"
	}
	if n.WindSpeed < 0.5 {
		return "Calm"
	}
	s := fmt.Sprintf("%s %.0f %s", compass(n.WindDir), n.WindSpeed, unit)
	if !math.IsNaN(n.WindGust) && n.WindGust >= n.WindSpeed+3 {
		s += fmt.Sprintf(", gusts %.0f", n.WindGust)
	}
	return strings.TrimSpace(s)
}

func rain(v float64, imperial bool) string {
	if math.IsNaN(v) {
		return "–"
	}
	if imperial {
		return fmt.Sprintf("%.2f in", v)
	}
	return fmt.Sprintf("%.1f mm", v)
}

func pressure(v float64, imperial bool) string {
	if math.IsNaN(v) {
		return "–"
	}
	if imperial {
		return fmt.Sprintf("%.2f in", v)
	}
	return fmt.Sprintf("%.0f hPa", v)
}
