package bigclock

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	s := &Settings{hourFormat: "12", dateFormat: "Monday, January 2", color: "#eef0f4", location: time.UTC}
	now := time.Date(2026, 11, 12, 12, 5, 0, 0, time.UTC)
	for _, size := range [][2]int{{448, 352}, {200, 80}} {
		img := render(now, s, size[0], size[1], 2)
		if b := img.Bounds(); b.Dx() != size[0] || b.Dy() != size[1] {
			t.Fatalf("got %v", b)
		}
	}

	if out := os.Getenv("CLOCK_PREVIEW"); out != "" {
		img := render(now, s, 448, 352, 2)
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
}
