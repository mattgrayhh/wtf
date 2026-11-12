// Package kittygfx draws images in terminals that support the kitty graphics
// protocol (kitty, Ghostty, WezTerm, Rex and others).
//
// Images are shown with Unicode placeholders: the image is uploaded once and
// given a virtual placement, and the widget then fills its cells with the
// placeholder character U+10EEEE. The foreground colour of each cell carries
// the image ID and two combining diacritics carry the row and column. Because
// the placeholders are ordinary text to tcell, redraws, overlapping modals and
// layout changes need no special handling.
//
// See https://sw.kovidgoyal.net/kitty/graphics-protocol/
package kittygfx

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Placeholder is the Unicode character kitty replaces with image pixels.
const Placeholder rune = 0x10EEEE

// MaxCells is the largest row or column index a placeholder can encode.
var MaxCells = len(diacritics)

const (
	chunkSize       = 4096
	defaultCellW    = 8
	defaultCellH    = 16
	queryTimeout    = 500 * time.Millisecond
	envOverride     = "WTF_KITTY_GRAPHICS"
	firstImageID    = 1
	maxImageID      = 1<<24 - 1 // IDs are carried in a 24-bit colour
	defaultPlacemnt = 1
)

var (
	mu      sync.Mutex
	enabled bool
	cellW          = defaultCellW
	cellH          = defaultCellH
	nextID  uint32 = firstImageID
	live           = map[uint32]bool{}
	pending bytes.Buffer
)

// Init detects kitty graphics support. Call it once, before tview starts.
//
// mode comes from the wtf.kittyGraphics config setting: "auto" (the default)
// asks the terminal, "true" assumes support (useful inside multiplexers that
// pass graphics through but don't answer the query) and "false" turns images
// off. The WTF_KITTY_GRAPHICS environment variable overrides the setting.
func Init(mode string) {
	if env := os.Getenv(envOverride); env != "" {
		mode = env
	}

	var supported bool
	w, h := 0, 0
	switch strings.ToLower(mode) {
	case "0", "false", "off", "no":
		// Don't touch the terminal at all.
	case "1", "true", "on", "yes":
		_, w, h = queryTerminal(queryTimeout)
		supported = true
	default:
		supported, w, h = queryTerminal(queryTimeout)
	}

	mu.Lock()
	defer mu.Unlock()
	enabled = supported
	if w > 0 && h > 0 {
		cellW, cellH = w, h
	}
}

// Enabled reports whether images are drawn with the kitty graphics protocol.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// CellSize returns the size of one terminal cell in pixels.
func CellSize() (w, h int) {
	mu.Lock()
	defer mu.Unlock()
	return cellW, cellH
}

// RenderScale is how many image pixels to draw per reported cell pixel.
// Some terminals (Rex among them) report cell sizes in points on HiDPI
// screens, so small cells get 2x supersampling to keep text and lines sharp.
func RenderScale() float64 {
	_, h := CellSize()
	if h <= 20 {
		return 2
	}
	return 1
}

// NewID reserves an image ID.
func NewID() uint32 {
	mu.Lock()
	defer mu.Unlock()
	id := nextID
	nextID++
	if nextID > maxImageID {
		nextID = firstImageID
	}
	return id
}

// Upload sends an image to the terminal and places it virtually across a
// grid of cols x rows cells. Replaces any image already stored under id.
// The data is queued and written on the next Flush.
func Upload(id uint32, img image.Image, cols, rows int) error {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return err
	}
	payload := base64.StdEncoding.EncodeToString(encoded.Bytes())

	mu.Lock()
	defer mu.Unlock()

	for first := true; first || len(payload) > 0; first = false {
		n := min(chunkSize, len(payload))
		chunk := payload[:n]
		payload = payload[n:]
		more := 0
		if len(payload) > 0 {
			more = 1
		}

		if first {
			fmt.Fprintf(&pending, "\x1b_Ga=t,q=2,f=100,i=%d,m=%d;%s\x1b\\", id, more, chunk)
		} else {
			fmt.Fprintf(&pending, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}

	live[id] = true
	placeLocked(id, cols, rows)
	return nil
}

// Place changes the grid an uploaded image is stretched across.
func Place(id uint32, cols, rows int) {
	mu.Lock()
	defer mu.Unlock()
	placeLocked(id, cols, rows)
}

func placeLocked(id uint32, cols, rows int) {
	fmt.Fprintf(&pending, "\x1b_Ga=p,U=1,q=2,i=%d,p=%d,c=%d,r=%d\x1b\\", id, defaultPlacemnt, cols, rows)
}

// Delete frees an image and its placements in the terminal.
func Delete(id uint32) {
	mu.Lock()
	defer mu.Unlock()
	if !live[id] {
		return
	}
	delete(live, id)
	fmt.Fprintf(&pending, "\x1b_Ga=d,d=I,q=2,i=%d\x1b\\", id)
}

// Flush writes queued graphics commands. Call it from tview's after-draw
// handler, which runs on the draw goroutine just before tcell writes the
// frame, so our escape sequences never interleave with tcell's output.
func Flush(screen tcell.Screen) {
	mu.Lock()
	defer mu.Unlock()
	if pending.Len() == 0 {
		return
	}

	var w io.Writer = os.Stdout
	if tty, ok := screen.Tty(); ok && tty != nil {
		w = tty
	}
	_, _ = w.Write(pending.Bytes())
	pending.Reset()
}

// Cleanup frees every image still in the terminal. Call it on exit.
func Cleanup() {
	mu.Lock()
	ids := make([]uint32, 0, len(live))
	for id := range live {
		ids = append(ids, id)
	}
	empty := len(ids) == 0 && pending.Len() == 0
	mu.Unlock()

	if empty {
		return
	}

	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer func() { _ = tty.Close() }()

	for _, id := range ids {
		Delete(id)
	}
	mu.Lock()
	_, _ = tty.Write(pending.Bytes())
	pending.Reset()
	mu.Unlock()
}

// Paint fills a cols x rows block of cells at (x, y) with placeholders for
// image id. Background is the cell background used around transparent pixels.
func Paint(screen tcell.Screen, id uint32, x, y, cols, rows int, background tcell.Color) {
	style := tcell.StyleDefault.
		Foreground(IDColor(id)).
		Background(background)

	cols = min(cols, MaxCells)
	rows = min(rows, MaxCells)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			screen.SetContent(x+c, y+r, Placeholder, []rune{diacritics[r], diacritics[c]}, style)
		}
	}
}

// IDColor encodes an image ID as the 24-bit foreground colour kitty reads it from.
func IDColor(id uint32) tcell.Color {
	return tcell.NewRGBColor(int32(id>>16&0xff), int32(id>>8&0xff), int32(id&0xff))
}

var (
	graphicsOK = regexp.MustCompile(`\x1b_Gi=31;OK\x1b\\`)
	cellSizeRe = regexp.MustCompile(`\x1b\[6;(\d+);(\d+)t`)
)

// parseReply reads the terminal's answers to the detection query.
func parseReply(reply []byte) (supported bool, w, h int) {
	supported = graphicsOK.Match(reply)
	if m := cellSizeRe.FindSubmatch(reply); m != nil {
		h, _ = strconv.Atoi(string(m[1]))
		w, _ = strconv.Atoi(string(m[2]))
	}
	return supported, w, h
}
