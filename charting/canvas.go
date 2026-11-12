package charting

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// Font weights for Canvas text
const (
	Regular = iota
	Medium
	Bold
)

// Point is a position in pixels.
type Point struct{ X, Y float64 }

// Canvas is a transparent RGBA image with simple anti-aliased drawing, for
// modules that draw their own cards rather than charts.
type Canvas struct {
	Img *image.RGBA
}

// NewCanvas returns a transparent canvas.
func NewCanvas(width, height int) *Canvas {
	return &Canvas{Img: image.NewRGBA(image.Rect(0, 0, width, height))}
}

func (c *Canvas) raster() *vector.Rasterizer {
	b := c.Img.Bounds()
	r := vector.NewRasterizer(b.Dx(), b.Dy())
	r.DrawOp = draw.Over
	return r
}

func (c *Canvas) paint(r *vector.Rasterizer, col color.NRGBA) {
	r.Draw(c.Img, c.Img.Bounds(), image.NewUniform(col), image.Point{})
}

// Rect fills a rectangle.
func (c *Canvas) Rect(x0, y0, x1, y1 float64, col color.NRGBA) {
	c.RoundRect(x0, y0, x1, y1, 0, col)
}

// RoundRect fills a rectangle with rounded corners of radius rad.
func (c *Canvas) RoundRect(x0, y0, x1, y1, rad float64, col color.NRGBA) {
	if x1 <= x0 || y1 <= y0 {
		return
	}
	rad = math.Min(rad, math.Min(x1-x0, y1-y0)/2)
	r := c.raster()
	if rad <= 0 {
		r.MoveTo(float32(x0), float32(y0))
		r.LineTo(float32(x1), float32(y0))
		r.LineTo(float32(x1), float32(y1))
		r.LineTo(float32(x0), float32(y1))
		r.ClosePath()
		c.paint(r, col)
		return
	}

	var pts []Point
	corner := func(cx, cy, start float64) {
		for k := 0; k <= 6; k++ {
			a := start + float64(k)/6*math.Pi/2
			pts = append(pts, Point{cx + rad*math.Cos(a), cy + rad*math.Sin(a)})
		}
	}
	corner(x1-rad, y0+rad, -math.Pi/2)
	corner(x1-rad, y1-rad, 0)
	corner(x0+rad, y1-rad, math.Pi/2)
	corner(x0+rad, y0+rad, math.Pi)
	addPath(r, pts)
	c.paint(r, col)
}

// Circle fills a disc.
func (c *Canvas) Circle(cx, cy, rad float64, col color.NRGBA) {
	r := c.raster()
	addPath(r, circle(cx, cy, rad))
	c.paint(r, col)
}

// Erase clears a disc to transparent, e.g. to cut a crescent from a circle.
func (c *Canvas) Erase(cx, cy, rad float64) {
	r := c.raster()
	r.DrawOp = draw.Src
	addPath(r, circle(cx, cy, rad))
	r.Draw(c.Img, c.Img.Bounds(), image.Transparent, image.Point{})
}

// Shapes fills the union of several polygons in one colour, so overlapping
// parts (like the puffs of a cloud) don't double up their alpha.
func (c *Canvas) Shapes(col color.NRGBA, polys ...[]Point) {
	r := c.raster()
	for _, p := range polys {
		addPath(r, p)
	}
	c.paint(r, col)
}

// Line strokes a polyline with round joins and caps.
func (c *Canvas) Line(pts []Point, width float64, col color.NRGBA) {
	c.Shapes(col, strokePolys(pts, width)...)
}

// Text draws s with its baseline at y, starting at x.
func (c *Canvas) Text(s string, x, y, size float64, weight int, col color.NRGBA) {
	face := Face(size, weight)
	d := font.Drawer{
		Dst:  c.Img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)},
	}
	d.DrawString(s)
}

// TextCentered draws s centred on cx.
func (c *Canvas) TextCentered(s string, cx, y, size float64, weight int, col color.NRGBA) {
	c.Text(s, cx-TextWidth(s, size, weight)/2, y, size, weight, col)
}

// TextRight draws s ending at x.
func (c *Canvas) TextRight(s string, x, y, size float64, weight int, col color.NRGBA) {
	c.Text(s, x-TextWidth(s, size, weight), y, size, weight, col)
}

// TextWidth measures s in pixels.
func TextWidth(s string, size float64, weight int) float64 {
	return float64(font.MeasureString(Face(size, weight), s)) / 64
}

// Fit shrinks size until s fits in width.
func Fit(s string, size, width float64, weight int) float64 {
	for size > 6 && TextWidth(s, size, weight) > width {
		size *= 0.92
	}
	return size
}

/* -------------------- Geometry -------------------- */

func circle(cx, cy, rad float64) []Point {
	sides := int(math.Max(16, math.Min(96, rad)))
	pts := make([]Point, sides)
	for k := range pts {
		a := float64(k) / float64(sides) * 2 * math.Pi
		pts[k] = Point{cx + rad*math.Cos(a), cy + rad*math.Sin(a)}
	}
	return pts
}

// Circle returns a disc outline, for use with Shapes.
func CirclePath(cx, cy, rad float64) []Point { return circle(cx, cy, rad) }

func strokePolys(pts []Point, width float64) [][]Point {
	half := width / 2
	var polys [][]Point
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		dx, dy := b.X-a.X, b.Y-a.Y
		l := math.Hypot(dx, dy)
		if l == 0 {
			continue
		}
		nx, ny := -dy/l*half, dx/l*half
		polys = append(polys, []Point{{a.X + nx, a.Y + ny}, {a.X - nx, a.Y - ny}, {b.X - nx, b.Y - ny}, {b.X + nx, b.Y + ny}})
	}
	for _, p := range pts {
		polys = append(polys, circle(p.X, p.Y, half))
	}
	return polys
}

// addPath adds a closed polygon wound clockwise, so shapes in one rasterizer
// union instead of cancelling.
func addPath(r *vector.Rasterizer, pts []Point) {
	if len(pts) < 3 {
		return
	}
	area := 0.0
	for i := range pts {
		j := (i + 1) % len(pts)
		area += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	if area < 0 {
		rev := make([]Point, len(pts))
		for i := range pts {
			rev[i] = pts[len(pts)-1-i]
		}
		pts = rev
	}
	r.MoveTo(float32(pts[0].X), float32(pts[0].Y))
	for _, p := range pts[1:] {
		r.LineTo(float32(p.X), float32(p.Y))
	}
	r.ClosePath()
}

/* -------------------- Fonts -------------------- */

type faceKey struct {
	size   float64
	weight int
}

var (
	fontsOnce sync.Once
	fontsErr  error
	fonts     [3]*opentype.Font
	faceMu    sync.Mutex
	faceCache = map[faceKey]font.Face{}
)

// Face returns a Go font face at size pixels.
func Face(size float64, weight int) font.Face {
	fontsOnce.Do(func() {
		for i, ttf := range [][]byte{goregular.TTF, gomedium.TTF, gobold.TTF} {
			if fonts[i], fontsErr = opentype.Parse(ttf); fontsErr != nil {
				return
			}
		}
	})
	if weight < Regular || weight > Bold {
		weight = Regular
	}
	size = math.Round(size*4) / 4 // keep the cache small

	faceMu.Lock()
	defer faceMu.Unlock()
	key := faceKey{size, weight}
	if f, ok := faceCache[key]; ok {
		return f
	}
	f, err := opentype.NewFace(fonts[weight], &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		f, _ = opentype.NewFace(fonts[Regular], &opentype.FaceOptions{Size: 12, DPI: 72})
	}
	faceCache[key] = f
	return f
}

// RGB is a convenience for opaque colours from a hex string; it panics on
// bad input, so use it only with constants.
func RGB(hex string) color.NRGBA {
	c, ok := ParseColor(hex)
	if !ok {
		panic("charting: bad colour " + hex)
	}
	return c
}
