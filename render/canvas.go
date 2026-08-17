// Package render provides the pure-Go 2D rasterizer that modules draw into.
package render

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Canvas is a drawable RGBA surface backed by an in-memory image.
type Canvas struct {
	Img *image.RGBA
}

// NewCanvas allocates a canvas of the given dimensions.
func NewCanvas(width, height int) *Canvas {
	return &Canvas{Img: image.NewRGBA(image.Rect(0, 0, width, height))}
}

// ParseHexColor converts a "#rrggbb" (or "#rgb") string into an opaque RGBA
// color. It returns a fallback color when s is unparseable.
func ParseHexColor(s string, fallback color.RGBA) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return fallback
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return fallback
	}
	return color.RGBA{R: byte(v >> 16), G: byte(v >> 8), B: byte(v), A: 0xff}
}

// Fill paints the whole canvas with col.
func (c *Canvas) Fill(col color.RGBA) {
	drawRect(c.Img, c.Img.Bounds(), col)
}

// FillRect paints the intersection of r and the canvas with col.
func (c *Canvas) FillRect(r image.Rectangle, col color.RGBA) {
	r = r.Intersect(c.Img.Bounds())
	if r.Empty() {
		return
	}
	drawRect(c.Img, r, col)
}

// FaceMetrics returns the ascent, descent, and total line height of face.
func (c *Canvas) FaceMetrics(f font.Face) (ascent, descent, height int) {
	m := f.Metrics()
	return m.Ascent.Ceil(), m.Descent.Ceil(), m.Height.Ceil()
}

// TextSize measures the rendered width and line height of s with face.
func (c *Canvas) TextSize(f font.Face, s string) (w, h int) {
	_, _, h = c.FaceMetrics(f)
	return font.MeasureString(f, s).Ceil(), h
}

// DrawText draws s with its baseline at pt (top-left glyph origin).
func (c *Canvas) DrawText(pt image.Point, s string, f font.Face, col color.RGBA) {
	d := &font.Drawer{Dst: c.Img, Src: image.NewUniform(col), Face: f, Dot: fixed.P(pt.X, pt.Y)}
	d.DrawString(s)
}

// DrawTextCentered draws s horizontally and vertically centered in bounds.
func (c *Canvas) DrawTextCentered(bounds image.Rectangle, s string, f font.Face, col color.RGBA) {
	w, h := c.TextSize(f, s)
	_, _, _, ascent := lineBox(f)
	x := bounds.Min.X + (bounds.Dx()-w)/2
	y := bounds.Min.Y + (bounds.Dy()-h)/2 + ascent
	c.DrawText(image.Pt(x, y), s, f, col)
}

// lineBox returns (ascent, descent, height, ascent) for a face.
func lineBox(f font.Face) (int, int, int, int) {
	m := f.Metrics()
	ascent := m.Ascent.Ceil()
	descent := m.Descent.Ceil()
	return ascent, descent, ascent + descent, ascent
}

// DividerH draws a horizontal rule from x0 to x1 at row y.
func (c *Canvas) DividerH(x0, x1, y int, col color.RGBA) {
	for x := x0; x <= x1; x++ {
		c.Img.Set(x, y, col)
	}
}

// StatusDot draws a filled circle of the given radius centered at pt.
func (c *Canvas) StatusDot(pt image.Point, radius int, col color.RGBA) {
	r := radius
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				c.Img.Set(pt.X+dx, pt.Y+dy, col)
			}
		}
	}
}

func drawRect(dst *image.RGBA, r image.Rectangle, col color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.Pix[i+0] = col.R
			dst.Pix[i+1] = col.G
			dst.Pix[i+2] = col.B
			dst.Pix[i+3] = col.A
			i += 4
		}
	}
}
