package wunderground

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func sampleReport() *report {
	now := time.Date(2026, 11, 12, 15, 40, 0, 0, time.Local)
	r := &report{Imperial: true, Now: current{
		Temp: 68.4, FeelsLike: 70.1, Humidity: 62, WindSpeed: 9, WindGust: 16, WindDir: 135,
		PrecipToday: 0.12, Pressure: 30.02, Phrase: "Partly Cloudy", Sky: skyPartly, Observed: now,
	}}
	skies := []sky{skyPartly, skyRain, skyStorm, skyClear, skyCloudy, skyFog, skySnow}
	for i := 0; i < 5; i++ {
		r.Days = append(r.Days, day{
			Date: now.AddDate(0, 0, i), High: 74 - float64(i*2) + float64(i%2)*5, Low: 58 + float64(i),
			PrecipChance: float64([]int{10, 70, 90, 0, 30}[i]), Sky: skies[i],
		})
	}
	r.Days[0].High = math.NaN()
	return r
}

// TestCardPreview writes sample cards for eyeballing:
//
//	CARD_PREVIEW=/tmp/card go test ./modules/wunderground -run TestCardPreview
func TestCardPreview(t *testing.T) {
	out := os.Getenv("CARD_PREVIEW")
	if out == "" {
		t.Skip()
	}
	for name, size := range map[string][2]int{"wide": {1440, 384}, "narrow": {640, 640}} {
		img, err := renderCard(sampleReport(), size[0], size[1], 2, sampleReport().Now.Observed)
		if err != nil {
			t.Fatal(err)
		}
		bg := image.NewRGBA(img.Bounds())
		draw.Draw(bg, bg.Bounds(), &image.Uniform{color.RGBA{0x1e, 0x1e, 0x22, 0xff}}, image.Point{}, draw.Src)
		draw.Draw(bg, bg.Bounds(), img, image.Point{}, draw.Over)
		f, err := os.Create(strings.TrimSuffix(out, ".png") + "-" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, bg); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
}

func TestRenderCardSizes(t *testing.T) {
	for _, size := range [][2]int{{1440, 384}, {640, 640}, {300, 120}} {
		img, err := renderCard(sampleReport(), size[0], size[1], 1, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != size[0] || b.Dy() != size[1] {
			t.Fatalf("got %v want %v", b, size)
		}
	}
	if _, err := renderCard(sampleReport(), 50, 20, 1, time.Now()); err == nil {
		t.Fatal("expected an error for a tiny panel")
	}
}

func TestHelpers(t *testing.T) {
	if compass(135) != "SE" || compass(359) != "N" || compass(math.NaN()) != "" {
		t.Error("compass")
	}
	if wuSky(32) != skyClear || wuSky(12) != skyRain || wuSky(4) != skyStorm || wuSky(30) != skyPartly {
		t.Error("wuSky")
	}
	if feelsLike(80, 85, math.NaN()) != 85 || feelsLike(40, math.NaN(), 33) != 33 || feelsLike(60, 58, 61) != 60 {
		t.Error("feelsLike")
	}
}
