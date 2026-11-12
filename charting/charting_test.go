package charting

import (
	"math"
	"testing"
)

func TestRenderSize(t *testing.T) {
	for _, kind := range []string{Line, Area, Bar} {
		img, err := Render(Spec{
			Kind:   kind,
			Labels: []string{"a", "b", "c"},
			Series: []Series{{Name: "x", Values: []float64{1, math.NaN(), 3}}},
			Legend: true,
		}, 300, 200)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if b := img.Bounds(); b.Dx() != 300 || b.Dy() != 200 {
			t.Fatalf("%s: got %v, want 300x200", kind, b)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	if _, err := Render(Spec{}, 300, 200); err == nil {
		t.Error("no series should be an error")
	}
	if _, err := Render(Spec{Series: []Series{{Values: []float64{1}}}}, 20, 20); err == nil {
		t.Error("a tiny panel should be an error")
	}
}

func TestAxisFit(t *testing.T) {
	a := newAxis(Spec{Series: []Series{{Values: []float64{3, 97}}}}, false, "")
	a.fit(5)
	if a.min != 0 || a.max != 100 {
		t.Fatalf("range %v..%v, want 0..100", a.min, a.max)
	}
	if a.ticks[0] != 0 || a.ticks[len(a.ticks)-1] != 100 {
		t.Fatalf("ticks %v", a.ticks)
	}

	lo := 10.0
	b := newAxis(Spec{YMin: &lo, Series: []Series{{Values: []float64{12, 18}}}}, false, "")
	b.fit(4)
	if b.min != 10 {
		t.Fatalf("pinned min moved to %v", b.min)
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		in       float64
		decimals int
		want     string
	}{
		{12, 0, "12"},
		{0.5, 1, "0.5"},
		{15000, 0, "15k"},
		{2500000, 0, "2.5M"},
	}
	for _, tt := range tests {
		if got := compact(tt.in, tt.decimals); got != tt.want {
			t.Errorf("compact(%v, %d) = %q, want %q", tt.in, tt.decimals, got, tt.want)
		}
	}
}
