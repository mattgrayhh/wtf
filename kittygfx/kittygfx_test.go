package kittygfx

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseReply(t *testing.T) {
	supported, w, h := parseReply([]byte("\x1b_Gi=31;OK\x1b\\\x1b[6;16;8t\x1b[?62;22c"))
	if !supported || w != 8 || h != 16 {
		t.Fatalf("got supported=%v cell=%dx%d, want true 8x16", supported, w, h)
	}

	supported, w, h = parseReply([]byte("\x1b[?62;22c"))
	if supported || w != 0 || h != 0 {
		t.Fatalf("plain terminal: got supported=%v cell=%dx%d", supported, w, h)
	}

	if supported, _, _ := parseReply([]byte("\x1b_Gi=31;ENOTSUPPORTED\x1b\\")); supported {
		t.Fatal("error reply treated as supported")
	}
}

func TestHasDeviceAttributes(t *testing.T) {
	if hasDeviceAttributes([]byte("\x1b_Gi=31;OK\x1b\\")) {
		t.Fatal("found DA1 in a graphics reply")
	}
	if !hasDeviceAttributes([]byte("\x1b_Gi=31;OK\x1b\\\x1b[?62;22c")) {
		t.Fatal("missed DA1")
	}
}

func TestIDColor(t *testing.T) {
	r, g, b := IDColor(0x0a0b0c).RGB()
	if r != 0x0a || g != 0x0b || b != 0x0c {
		t.Fatalf("got %x %x %x", r, g, b)
	}
}

func TestUploadChunksAndPlaces(t *testing.T) {
	pending.Reset()
	defer pending.Reset()

	img := testImage(200, 200) // large enough to need several chunks
	if err := Upload(7, img, 10, 5); err != nil {
		t.Fatal(err)
	}
	out := pending.String()

	if !strings.HasPrefix(out, "\x1b_Ga=t,q=2,f=100,i=7,m=1;") {
		t.Fatalf("first chunk header wrong: %q", out[:40])
	}
	if !strings.Contains(out, "\x1b_Gm=0;") {
		t.Fatal("missing final chunk")
	}
	if !strings.HasSuffix(out, "\x1b_Ga=p,U=1,q=2,i=7,p=1,c=10,r=5\x1b\\") {
		t.Fatal("missing virtual placement")
	}
	for _, chunk := range strings.Split(out, "\x1b\\") {
		if i := strings.IndexByte(chunk, ';'); i >= 0 && len(chunk)-i-1 > chunkSize {
			t.Fatalf("chunk payload is %d bytes, max %d", len(chunk)-i-1, chunkSize)
		}
	}
}

func TestPaint(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(10, 4)

	Paint(screen, 5, 1, 1, 3, 2, tcell.ColorDefault)

	str, style, _ := screen.Get(2, 2)
	want := string([]rune{Placeholder, diacritics[1], diacritics[1]})
	if str != want {
		t.Fatalf("cell (2,2) = %q, want %q", str, want)
	}
	fg, _, _ := style.Decompose()
	if fg != IDColor(5) {
		t.Fatalf("cell colour %v, want image ID 5", fg)
	}
	if str, _, _ := screen.Get(0, 0); str == want {
		t.Fatal("painted outside the block")
	}
}
