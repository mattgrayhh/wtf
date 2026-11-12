// Package image shows a picture from a file, a URL or a command's output.
// In terminals with kitty graphics support it's drawn at full resolution.
package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	stdimage "image"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	// Register image decoders
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/view"
)

const (
	fetchTimeout = 20 * time.Second
	maxImageSize = 32 << 20
)

// Widget is the container for the image module
type Widget struct {
	*view.ImageWidget

	settings *Settings
}

// NewWidget creates a new instance of the widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	widget := &Widget{
		ImageWidget: view.NewImageWidget(tviewApp, redrawChan, nil, settings.Common),

		settings: settings,
	}
	widget.SetFit(settings.fit)

	return widget
}

// Refresh loads the image again
func (widget *Widget) Refresh() {
	title := widget.CommonSettings().Title

	img, err := widget.load()
	if err != nil {
		widget.SetMessage(title, err.Error())
		return
	}

	widget.SetImage(title, img)
}

func (widget *Widget) load() (stdimage.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	switch {
	case widget.settings.cmd != "":
		return fromCommand(ctx, widget.settings.cmd, widget.settings.args)
	case widget.settings.url != "":
		return fromURL(ctx, widget.settings.url)
	case widget.settings.path != "":
		return fromFile(widget.settings.path)
	default:
		return nil, errors.New("set path, url or cmd in this module's config")
	}
}

func fromFile(path string) (stdimage.Image, error) {
	path, err := utils.ExpandHomeDir(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return decode(data)
}

func fromURL(ctx context.Context, url string) (stdimage.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	// Some image services (wttr.in among them) return text to curl-like clients
	req.Header.Set("User-Agent", "wtfutil")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize))
	if err != nil {
		return nil, err
	}

	return decode(data)
}

// fromCommand runs cmd. If it prints an image, that's used directly;
// otherwise the output is read as a path or URL.
func fromCommand(ctx context.Context, cmd string, args []string) (stdimage.Image, error) {
	var stderr bytes.Buffer
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stderr = &stderr

	out, err := c.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%s: %s", cmd, msg)
		}
		return nil, fmt.Errorf("%s: %w", cmd, err)
	}

	if img, err := decode(out); err == nil {
		return img, nil
	}

	target := strings.TrimSpace(string(out))
	switch {
	case target == "":
		return nil, errors.New("nothing to show")
	case strings.HasPrefix(target, "http://"), strings.HasPrefix(target, "https://"):
		return fromURL(ctx, target)
	default:
		return fromFile(target)
	}
}

func decode(data []byte) (stdimage.Image, error) {
	img, _, err := stdimage.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("not a supported image: %w", err)
	}
	return img, nil
}
