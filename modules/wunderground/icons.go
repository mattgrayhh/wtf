package wunderground

import (
	"image/color"
	"math"

	"github.com/wtfutil/wtf/charting"
)

var (
	sunColor   = charting.RGB("#f5c242")
	moonColor  = charting.RGB("#d9dce6")
	cloudLight = charting.RGB("#d4d9e2")
	cloudDark  = charting.RGB("#9aa3b2")
	rainColor  = charting.RGB("#4e9af1")
	snowColor  = charting.RGB("#e8eef7")
	boltColor  = charting.RGB("#f5c242")
	fogColor   = charting.RGB("#a7afbd")
)

// drawIcon draws a weather icon filling a size x size box at (x, y).
func drawIcon(c *charting.Canvas, sk sky, night bool, x, y, size float64) {
	u := size / 100 // icons are designed on a 100x100 grid

	switch sk {
	case skyClear:
		if night {
			moon(c, x+50*u, y+50*u, 30*u)
		} else {
			sun(c, x+50*u, y+50*u, 22*u, true)
		}
	case skyPartly:
		if night {
			moon(c, x+62*u, y+36*u, 22*u)
		} else {
			sun(c, x+62*u, y+36*u, 16*u, true)
		}
		cloud(c, x+4*u, y+34*u, 72*u, cloudLight)
	case skyCloudy:
		cloud(c, x+26*u, y+20*u, 64*u, cloudDark)
		cloud(c, x+6*u, y+34*u, 76*u, cloudLight)
	case skyRain:
		cloud(c, x+10*u, y+18*u, 80*u, cloudDark)
		for i, dx := range []float64{30, 50, 70} {
			top := y + (64+float64(i%2)*6)*u
			c.Line([]charting.Point{{X: x + dx*u, Y: top}, {X: x + (dx-6)*u, Y: top + 18*u}}, 5*u, rainColor)
		}
	case skyStorm:
		cloud(c, x+10*u, y+14*u, 80*u, cloudDark)
		c.Shapes(boltColor, []charting.Point{
			{X: x + 54*u, Y: y + 54*u}, {X: x + 38*u, Y: y + 78*u}, {X: x + 50*u, Y: y + 78*u},
			{X: x + 42*u, Y: y + 98*u}, {X: x + 66*u, Y: y + 70*u}, {X: x + 54*u, Y: y + 70*u}, {X: x + 62*u, Y: y + 54*u},
		})
	case skySnow:
		cloud(c, x+10*u, y+18*u, 80*u, cloudLight)
		for i, dx := range []float64{30, 50, 70} {
			c.Circle(x+dx*u, y+(72+float64(i%2)*10)*u, 5*u, snowColor)
		}
	case skyFog:
		for i := 0; i < 4; i++ {
			yy := y + (30+float64(i)*14)*u
			x0, x1 := x+(14+float64(i%2)*8)*u, x+(86-float64((i+1)%2)*8)*u
			c.Line([]charting.Point{{X: x0, Y: yy}, {X: x1, Y: yy}}, 6*u, fogColor)
		}
	case skyWind:
		for i, w := range []float64{60, 76, 48} {
			yy := y + (34+float64(i)*16)*u
			c.Line([]charting.Point{{X: x + 14*u, Y: yy}, {X: x + (14+w)*u, Y: yy}}, 6*u, fogColor)
		}
	}
}

func sun(c *charting.Canvas, cx, cy, r float64, rays bool) {
	if rays {
		var polys [][]charting.Point
		for k := 0; k < 8; k++ {
			a := float64(k) * math.Pi / 4
			in, out := r*1.35, r*1.8
			polys = append(polys, strokeSeg(cx+in*math.Cos(a), cy+in*math.Sin(a), cx+out*math.Cos(a), cy+out*math.Sin(a), r*0.22)...)
		}
		c.Shapes(sunColor, polys...)
	}
	c.Circle(cx, cy, r, sunColor)
}

func moon(c *charting.Canvas, cx, cy, r float64) {
	c.Circle(cx, cy, r, moonColor)
	c.Erase(cx+r*0.55, cy-r*0.35, r*0.85)
}

// cloud draws a cloud whose bounding box starts at (x, y) and is w wide.
func cloud(c *charting.Canvas, x, y, w float64, col color.NRGBA) {
	u := w / 100
	base := y + 52*u
	c.Shapes(col,
		charting.CirclePath(x+30*u, base-10*u, 20*u),
		charting.CirclePath(x+56*u, base-22*u, 26*u),
		charting.CirclePath(x+78*u, base-6*u, 16*u),
		roundedBar(x+12*u, base-14*u, x+92*u, base+10*u, 12*u),
	)
}

func roundedBar(x0, y0, x1, y1, r float64) []charting.Point {
	var pts []charting.Point
	corner := func(cx, cy, start float64) {
		for k := 0; k <= 6; k++ {
			a := start + float64(k)/6*math.Pi/2
			pts = append(pts, charting.Point{X: cx + r*math.Cos(a), Y: cy + r*math.Sin(a)})
		}
	}
	corner(x1-r, y0+r, -math.Pi/2)
	corner(x1-r, y1-r, 0)
	corner(x0+r, y1-r, math.Pi/2)
	corner(x0+r, y0+r, math.Pi)
	return pts
}

func strokeSeg(x0, y0, x1, y1, w float64) [][]charting.Point {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	nx, ny := -dy/l*w/2, dx/l*w/2
	return [][]charting.Point{
		{{X: x0 + nx, Y: y0 + ny}, {X: x0 - nx, Y: y0 - ny}, {X: x1 - nx, Y: y1 - ny}, {X: x1 + nx, Y: y1 + ny}},
		charting.CirclePath(x0, y0, w/2),
		charting.CirclePath(x1, y1, w/2),
	}
}
