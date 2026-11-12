package view

import (
	"image"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/cfg"
	"github.com/wtfutil/wtf/kittygfx"
	"github.com/wtfutil/wtf/wtf"
)

// RenderFunc draws an image at exactly width x height pixels. Charts use it
// so they're rendered crisply at the panel's real size instead of scaled.
// scale is how many image pixels make one screen point; multiply line widths
// and font sizes by it.
type RenderFunc func(width, height int, scale float64) (image.Image, error)

// Fit modes for static images
const (
	FitContain = "contain" // keep aspect ratio, fit inside the panel
	FitFill    = "fill"    // stretch to fill the panel
)

// ImageWidget is a module panel that shows a picture instead of text. In
// terminals that support the kitty graphics protocol the picture is drawn at
// full resolution; elsewhere it falls back to coloured block characters.
type ImageWidget struct {
	*Base
	*KeyboardWidget

	View *tview.TextView

	mu       sync.Mutex
	id       uint32
	img      image.Image // static image, scaled by the terminal
	render   RenderFunc  // dynamic image, re-rendered when the panel resizes
	fit      string
	version  int // bumped whenever img or render changes
	sent     placement
	fallback *tview.Image
}

// placement records what was last uploaded, so we only resend on change.
type placement struct {
	version    int
	cols, rows int
	pxW, pxH   int
}

// NewImageWidget creates an image panel.
func NewImageWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, commonSettings *cfg.Common) *ImageWidget {
	widget := &ImageWidget{
		Base:           NewBase(tviewApp, redrawChan, pages, commonSettings),
		KeyboardWidget: NewKeyboardWidget(commonSettings),

		id:  kittygfx.NewID(),
		fit: FitContain,
	}

	widget.View = widget.createView(widget.bordered)
	widget.View.SetInputCapture(widget.InputCapture)
	widget.View.SetDrawFunc(widget.drawImage)

	widget.SetView(widget.View)
	widget.helpTextFunc = widget.HelpText

	return widget
}

/* -------------------- Exported Functions -------------------- */

// TextView returns the tview.TextView the image is drawn into
func (widget *ImageWidget) TextView() *tview.TextView {
	return widget.View
}

// SetFit chooses how static images are sized: FitContain or FitFill.
func (widget *ImageWidget) SetFit(fit string) {
	widget.mu.Lock()
	defer widget.mu.Unlock()
	widget.fit = strings.ToLower(fit)
	widget.version++
}

// SetImage shows a static image, replacing any message or chart.
func (widget *ImageWidget) SetImage(title string, img image.Image) {
	widget.mu.Lock()
	widget.img = img
	widget.render = nil
	widget.version++
	widget.mu.Unlock()

	widget.update(title, "")
}

// SetRenderer shows an image that's redrawn at the panel's pixel size, such
// as a chart. Call it again whenever the underlying data changes.
func (widget *ImageWidget) SetRenderer(title string, render RenderFunc) {
	widget.mu.Lock()
	widget.img = nil
	widget.render = render
	widget.version++
	widget.mu.Unlock()

	widget.update(title, "")
}

// SetMessage shows text instead of an image, for errors and loading states.
func (widget *ImageWidget) SetMessage(title, message string) {
	widget.mu.Lock()
	widget.img = nil
	widget.render = nil
	widget.version++
	widget.mu.Unlock()

	widget.update(title, message)
}

// Stop frees the image in the terminal along with the widget.
func (widget *ImageWidget) Stop() {
	kittygfx.Delete(widget.id)
	widget.Base.Stop()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *ImageWidget) update(title, message string) {
	widget.View.SetTitle(widget.ContextualTitle(title))
	widget.View.SetText(message)

	select {
	case widget.RedrawChan <- true:
	default:
	}
}

func (widget *ImageWidget) createView(bordered bool) *tview.TextView {
	view := tview.NewTextView()

	view.SetBackgroundColor(wtf.ColorFor(widget.commonSettings.Colors.Background))
	view.SetBorder(bordered)
	view.SetBorderColor(wtf.ColorFor(widget.BorderColor()))
	view.SetDynamicColors(true)
	view.SetTextColor(wtf.ColorFor(widget.commonSettings.Colors.Text))
	view.SetTitleColor(wtf.ColorFor(widget.commonSettings.Colors.Title))
	view.SetWrap(true)

	return view
}

// drawImage runs after the box has drawn its background and border, and
// before the TextView draws its (empty) text. It returns the inner rectangle.
func (widget *ImageWidget) drawImage(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	if widget.bordered {
		x, y, width, height = x+1, y+1, width-2, height-2
	}
	if width <= 0 || height <= 0 {
		return x, y, max(width, 0), max(height, 0)
	}

	widget.mu.Lock()
	defer widget.mu.Unlock()

	if widget.img == nil && widget.render == nil {
		return x, y, width, height
	}

	cellW, cellH := kittygfx.CellSize()
	scale := kittygfx.RenderScale()
	width = min(width, kittygfx.MaxCells)
	height = min(height, kittygfx.MaxCells)

	// Work out the image and the block of cells it covers.
	img := widget.img
	cols, rows := width, height
	pxW, pxH := int(float64(width*cellW)*scale), int(float64(height*cellH)*scale)

	if widget.render != nil {
		if widget.sent.version != widget.version || widget.sent.pxW != pxW || widget.sent.pxH != pxH || !kittygfx.Enabled() {
			rendered, err := widget.render(pxW, pxH, scale)
			if err != nil {
				widget.drawError(screen, x, y, width, height, err.Error())
				return x, y, width, height
			}
			img = rendered
		}
	} else if widget.fit != FitFill {
		cols, rows = containCells(img.Bounds().Dx(), img.Bounds().Dy(), width, height, cellW, cellH)
	}

	// Centre the block inside the panel.
	ox := x + (width-cols)/2
	oy := y + (height-rows)/2

	if !kittygfx.Enabled() {
		widget.drawFallback(screen, img, ox, oy, cols, rows)
		return x, y, width, height
	}

	switch {
	case widget.sent.version != widget.version || (widget.render != nil && (widget.sent.pxW != pxW || widget.sent.pxH != pxH)):
		if err := kittygfx.Upload(widget.id, img, cols, rows); err != nil {
			widget.drawError(screen, x, y, width, height, err.Error())
			return x, y, width, height
		}
	case widget.sent.cols != cols || widget.sent.rows != rows:
		kittygfx.Place(widget.id, cols, rows)
	}
	widget.sent = placement{version: widget.version, cols: cols, rows: rows, pxW: pxW, pxH: pxH}

	bg := wtf.ColorFor(widget.commonSettings.Colors.Background)
	kittygfx.Paint(screen, widget.id, ox, oy, cols, rows, bg)

	return x, y, width, height
}

func (widget *ImageWidget) drawFallback(screen tcell.Screen, img image.Image, x, y, cols, rows int) {
	if img == nil {
		return
	}
	if widget.fallback == nil {
		widget.fallback = tview.NewImage()
		widget.fallback.SetBackgroundColor(wtf.ColorFor(widget.commonSettings.Colors.Background))
	}
	widget.fallback.SetImage(img)
	widget.fallback.SetRect(x, y, cols, rows)
	widget.fallback.Draw(screen)
}

func (widget *ImageWidget) drawError(screen tcell.Screen, x, y, width, _ int, message string) {
	tview.Print(screen, "[red]"+tview.Escape(message), x, y, width, tview.AlignLeft, tcell.ColorRed)
}

// containCells returns the largest cols x rows block, within maxCols x
// maxRows, that keeps an imgW x imgH image's aspect ratio.
func containCells(imgW, imgH, maxCols, maxRows, cellW, cellH int) (int, int) {
	if imgW <= 0 || imgH <= 0 {
		return maxCols, maxRows
	}

	panelW := float64(maxCols * cellW)
	panelH := float64(maxRows * cellH)
	scale := min(panelW/float64(imgW), panelH/float64(imgH))

	cols := int(float64(imgW)*scale/float64(cellW) + 0.5)
	rows := int(float64(imgH)*scale/float64(cellH) + 0.5)

	return min(max(cols, 1), maxCols), min(max(rows, 1), maxRows)
}
