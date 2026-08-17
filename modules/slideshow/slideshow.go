// Package slideshow implements the Photo Slideshow module: a rotating display
// of images from a local directory. No network access.
package slideshow

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	// Image codecs registered for image.Decode.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

var extensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
}

// Slideshow renders images from a directory, rotating on a cadence.
type Slideshow struct {
	dir      string
	interval time.Duration
	mode     render.FitMode

	files []string

	cacheIdx int
	cacheImg image.Image
}

// New constructs a photo slideshow module.
func New() engine.Module { return &Slideshow{} }

func init() { modules.Register("slideshow", New) }

// Name identifies the module.
func (s *Slideshow) Name() string { return "slideshow" }

// Configure applies module options and rescans the image directory.
func (s *Slideshow) Configure(opts map[string]any) error {
	s.dir = opt.Str(opts, "dir", "")
	if s.dir == "" {
		return fmt.Errorf("slideshow: option \"dir\" is required")
	}
	s.interval = opt.Duration(opts, "interval", 15)
	if s.interval <= 0 {
		s.interval = time.Second
	}
	switch opt.Str(opts, "fit", "contain") {
	case "cover":
		s.mode = render.FitCover
	default:
		s.mode = render.FitContain
	}

	files, err := scan(s.dir)
	if err != nil {
		return fmt.Errorf("slideshow: %w", err)
	}
	s.files = files
	s.cacheIdx = -1
	s.cacheImg = nil
	return nil
}

func (s *Slideshow) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "slideshow",
		Description: "Rotating photo display from a local directory",
		Fields: []engine.Field{
			{Key: "dir", Label: "Image Directory", Kind: engine.FieldFile, Required: true, Placeholder: "/path/to/photos", Hint: "Path to directory containing images"},
			{Key: "interval", Label: "Interval (s)", Kind: engine.FieldDuration, Default: "15", Hint: "Seconds between image transitions"},
			{Key: "fit", Label: "Fit Mode", Kind: engine.FieldSelect, Default: "contain", Options: []string{"contain", "cover"}},
		},
	}
}

// Start has no background work; the image index is derived from the clock.
func (s *Slideshow) Start(_ *engine.Bus, _ *engine.Logger) error { return nil }

// Stop is a no-op.
func (s *Slideshow) Stop() error { return nil }

// Draw renders the current image scaled to fit bounds.
func (s *Slideshow) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	if len(s.files) == 0 {
		return drawEmpty(cv, bounds)
	}
	idx := 0
	if s.interval > 0 && len(s.files) > 1 {
		idx = int(now.Unix()/int64(s.interval.Seconds())) % len(s.files)
	}
	if idx < 0 {
		idx = -idx % len(s.files)
	}
	img := s.imageAt(idx)
	if img == nil {
		return drawEmpty(cv, bounds)
	}
	cv.DrawImageFit(bounds, img, s.mode)
	return nil
}

// imageAt returns the decoded image at index idx, skipping corrupt files and
// caching the most recent decode.
func (s *Slideshow) imageAt(idx int) image.Image {
	if s.cacheIdx == idx && s.cacheImg != nil {
		return s.cacheImg
	}
	for i := 0; i < len(s.files); i++ {
		j := (idx + i) % len(s.files)
		img, err := decodeFile(s.files[j])
		if err != nil {
			continue
		}
		s.cacheIdx = j
		s.cacheImg = img
		return img
	}
	return nil
}

// scan lists supported image files in dir in sorted order.
func scan(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if extensions[strings.ToLower(filepath.Ext(e.Name()))] {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func decodeFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func drawEmpty(cv *render.Canvas, bounds image.Rectangle) error {
	f := fonts.Face(18, fonts.Regular)
	msg := "no images"
	w, _ := cv.TextSize(f, msg)
	ascent := cv.Ascent(f)
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-w)/2, bounds.Min.Y+(bounds.Dy()-ascent)/2), msg, f, color.RGBA{R: 154, G: 167, B: 184, A: 255})
	return nil
}
