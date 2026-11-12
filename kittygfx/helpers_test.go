package kittygfx

import (
	"image"
	"image/color"
	"math/rand"
)

// testImage returns noise, which doesn't compress, so PNGs stay large.
func testImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(1))
	for i := 0; i < w*h; i++ {
		img.Set(i%w, i/w, color.NRGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 0xff})
	}
	return img
}
