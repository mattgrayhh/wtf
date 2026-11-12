package charting

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"testing"
)

// TestPreview writes a sample chart for eyeballing changes:
//
//	CHART_PREVIEW=/tmp/chart.png go test ./charting -run TestPreview
func TestPreview(t *testing.T) {
	out := os.Getenv("CHART_PREVIEW")
	if out == "" {
		t.Skip()
	}
	var labels []string
	var temp, rain []float64
	for i := 0; i < 24; i++ {
		labels = append(labels, fmt.Sprintf("%02d", i))
		temp = append(temp, 9+5*math.Sin(float64(i-8)/24*2*math.Pi))
		rain = append(rain, math.Max(0, 60*math.Sin(float64(i)/5)))
	}
	lo, hi := 0.0, 100.0
	img, err := Render(Spec{Labels: labels, Kind: Area, Legend: true, Scale: 2, YFormat: "%.0f°", Y2Format: "%.0f%%", Y2Min: &lo, Y2Max: &hi,
		Series: []Series{{Name: "Temp", Values: temp, Color: "#f1a14e"}, {Name: "Rain %", Values: rain, Kind: Bar, Secondary: true, Color: "#4e9af1"}}}, 50*16, 16*32)
	if err != nil {
		t.Fatal(err)
	}
	bg := image.NewRGBA(img.Bounds())
	draw.Draw(bg, bg.Bounds(), &image.Uniform{color.RGBA{0x1e, 0x1e, 0x22, 0xff}}, image.Point{}, draw.Src)
	draw.Draw(bg, bg.Bounds(), img, image.Point{}, draw.Over)
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, bg); err != nil {
		t.Fatal(err)
	}
}
