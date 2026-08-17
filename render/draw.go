package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
)

// Ascent returns the cap-height offset from a baseline to the top of the
// glyph box, used to convert top-left anchors into baseline coordinates.
func (c *Canvas) Ascent(f font.Face) int {
	m := f.Metrics()
	return m.Ascent.Ceil()
}

// WrapText breaks text into lines no wider than maxWidth pixels. Existing
// newlines are preserved; words longer than maxWidth force their own line.
func WrapText(f font.Face, text string, maxWidth int) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			cand := cur + " " + w
			if font.MeasureString(f, cand).Ceil() <= maxWidth {
				cur = cand
			} else {
				lines = append(lines, cur)
				cur = w
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

// DrawTextBlock draws word-wrapped, top-aligned text clipped to bounds. Each
// line is drawn at lineHeight spacing (defaults to the face line height).
func (c *Canvas) DrawTextBlock(bounds image.Rectangle, text string, f font.Face, lineHeight int, col color.RGBA) {
	if lineHeight <= 0 {
		_, _, lineHeight = c.FaceMetrics(f)
	}
	ascent := c.Ascent(f)
	y := bounds.Min.Y
	for _, line := range WrapText(f, text, bounds.Dx()) {
		if y+lineHeight > bounds.Max.Y {
			break
		}
		c.DrawText(image.Pt(bounds.Min.X, y+ascent), line, f, col)
		y += lineHeight
	}
}

// DrawLabel draws text centered on a rounded background pill within bounds.
func (c *Canvas) DrawLabel(bounds image.Rectangle, text string, f font.Face, fg, bg color.RGBA, padX, padY int) {
	tw, th := c.TextSize(f, text)
	w, h := tw+padX*2, th+padY*2
	x := bounds.Min.X + (bounds.Dx()-w)/2
	y := bounds.Min.Y + (bounds.Dy()-h)/2
	pill := image.Rect(x, y, x+w, y+h)
	c.FillRoundRect(pill, 8, bg)
	c.DrawTextCentered(pill, text, f, fg)
}

// DividerV draws a vertical rule from y0 to y1 at column x.
func (c *Canvas) DividerV(x, y0, y1 int, col color.RGBA) {
	if x < c.Img.Bounds().Min.X || x >= c.Img.Bounds().Max.X {
		return
	}
	for y := y0; y <= y1; y++ {
		c.Img.Set(x, y, col)
	}
}

// WarningIcon draws a filled warning triangle with apex at pt and the given
// height. It serves as a generic status icon primitive.
func (c *Canvas) WarningIcon(pt image.Point, size int, col color.RGBA) {
	if size <= 0 {
		return
	}
	base := size / 2
	for dy := 0; dy < size; dy++ {
		half := (dy * base) / size
		for dx := -half; dx <= half; dx++ {
			c.Img.Set(pt.X+dx, pt.Y+dy, col)
		}
	}
}

// FillCircle paints a filled circle of radius r centered at pt using a cached
// alpha mask. Circles crossing the canvas edge are clipped to the surface.
func (c *Canvas) FillCircle(pt image.Point, r int, col color.RGBA) {
	if r <= 0 {
		return
	}
	full := image.Rect(pt.X-r, pt.Y-r, pt.X+r+1, pt.Y+r+1)
	rc := full.Intersect(c.Img.Bounds())
	if rc.Empty() {
		return
	}
	mask := circleMask(2*r+1, 2*r+1, r)
	mp := rc.Min.Sub(full.Min)
	draw.DrawMask(c.Img, rc, image.NewUniform(col), image.Point{}, mask, mp, draw.Over)
}

// FitMode selects how an image is scaled into a target rectangle.
type FitMode int

const (
	// FitContain scales the image to fit entirely inside the bounds, centered,
	// preserving aspect ratio (letterboxing).
	FitContain FitMode = iota
	// FitCover scales the image to cover the bounds entirely, cropping overflow.
	FitCover
)

// DrawImageFit renders src scaled to fit dstBounds using bilinear filtering.
func (c *Canvas) DrawImageFit(dstBounds image.Rectangle, src image.Image, mode FitMode) {
	sb := src.Bounds()
	if sb.Empty() {
		return
	}
	scaled := fitRect(dstBounds, sb.Dx(), sb.Dy(), mode)
	dstRect := scaled.Intersect(c.Img.Bounds())
	if dstRect.Empty() {
		return
	}
	draw.BiLinear.Scale(c.Img, scaled, src, sb, draw.Over, nil)
}

// fitRect computes the target rectangle for an image of size sw×sh within
// bounds under the given mode.
func fitRect(bounds image.Rectangle, sw, sh int, mode FitMode) image.Rectangle {
	if sw <= 0 || sh <= 0 {
		return image.Rectangle{}
	}
	bw, bh := bounds.Dx(), bounds.Dy()
	if bw <= 0 || bh <= 0 {
		return image.Rectangle{}
	}
	sfw, sfh := float64(bw)/float64(sw), float64(bh)/float64(sh)
	var scale float64
	if mode == FitCover {
		if sfw > sfh {
			scale = sfw
		} else {
			scale = sfh
		}
	} else {
		if sfw < sfh {
			scale = sfw
		} else {
			scale = sfh
		}
	}
	w := int(float64(sw) * scale)
	h := int(float64(sh) * scale)
	x := bounds.Min.X + (bw-w)/2
	y := bounds.Min.Y + (bh-h)/2
	return image.Rect(x, y, x+w, y+h)
}

var (
	maskMu    sync.Mutex
	maskCache = map[string]*image.Alpha{}
)

// FillRoundRect fills r with col, rounding the corners to radius pixels using
// a cached alpha mask (pure Go, no cgo).
func (c *Canvas) FillRoundRect(r image.Rectangle, radius int, col color.RGBA) {
	r = r.Intersect(c.Img.Bounds())
	if r.Empty() {
		return
	}
	if radius <= 0 {
		c.FillRect(r, col)
		return
	}
	mask := roundedMask(r.Dx(), r.Dy(), radius)
	draw.DrawMask(c.Img, r, image.NewUniform(col), image.Point{}, mask, image.Point{}, draw.Over)
}

func roundedMask(w, h, radius int) *image.Alpha {
	key := fmt.Sprintf("%dx%d:%d", w, h, radius)
	maskMu.Lock()
	defer maskMu.Unlock()
	if m, ok := maskCache[key]; ok {
		return m
	}
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := byte(255)
			var dx, dy int
			switch {
			case x < radius && y < radius:
				dx, dy = radius-1-x, radius-1-y
			case x >= w-radius && y < radius:
				dx, dy = x-(w-radius), radius-1-y
			case x < radius && y >= h-radius:
				dx, dy = radius-1-x, y-(h-radius)
			case x >= w-radius && y >= h-radius:
				dx, dy = x-(w-radius), y-(h-radius)
			}
			if (dx != 0 || dy != 0) && dx*dx+dy*dy > radius*radius {
				a = 0
			}
			m.SetAlpha(x, y, color.Alpha{A: a})
		}
	}
	maskCache[key] = m
	return m
}

// circleMask returns a cached w×h mask that is opaque inside a circle of the
// given radius centered on the mask.
func circleMask(w, h, radius int) *image.Alpha {
	key := fmt.Sprintf("circle:%dx%d:%d", w, h, radius)
	maskMu.Lock()
	defer maskMu.Unlock()
	if m, ok := maskCache[key]; ok {
		return m
	}
	cx, cy := (w-1)/2, (h-1)/2
	r2 := radius * radius
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x-cx, y-cy
			a := byte(0)
			if dx*dx+dy*dy <= r2 {
				a = 255
			}
			m.SetAlpha(x, y, color.Alpha{A: a})
		}
	}
	maskCache[key] = m
	return m
}
