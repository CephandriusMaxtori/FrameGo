package render

import (
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font"

	"framego/fonts"
)

func TestWrapText(t *testing.T) {
	f := fonts.Face(12, fonts.Regular)
	maxW := 80
	lines := WrapText(f, "the quick brown fox jumps", maxW)
	if len(lines) < 2 {
		t.Fatalf("expected multiple lines, got %d: %v", len(lines), lines)
	}
	for _, l := range lines {
		if w := font.MeasureString(f, l).Ceil(); w > maxW {
			t.Errorf("line %q width %d exceeds %d", l, w, maxW)
		}
	}
}

func TestWrapTextPreservesNewlines(t *testing.T) {
	f := fonts.Face(12, fonts.Regular)
	lines := WrapText(f, "line one\nline two", 10000)
	if len(lines) != 2 || lines[0] != "line one" || lines[1] != "line two" {
		t.Errorf("lines = %v", lines)
	}
}

func TestWrapTextLongWord(t *testing.T) {
	f := fonts.Face(12, fonts.Regular)
	lines := WrapText(f, "x xsupercalifragilisticexpialidocious y", 40)
	if len(lines) < 2 {
		t.Errorf("long word should wrap: %v", lines)
	}
}

func TestDrawTextBlock(t *testing.T) {
	c := NewCanvas(140, 90)
	c.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	bounds := image.Rect(10, 10, 130, 80)
	f := fonts.Face(13, fonts.Regular)
	c.DrawTextBlock(bounds, "the quick brown fox jumps over the lazy dog", f, 0, color.RGBA{R: 255, A: 255})
	if countLit(c, bounds) == 0 {
		t.Error("no text pixels rendered")
	}
}

func TestDrawLabel(t *testing.T) {
	c := NewCanvas(120, 60)
	c.Fill(color.RGBA{})
	f := fonts.Face(12, fonts.Medium)
	c.DrawLabel(image.Rect(0, 0, 120, 60), "status", f, color.RGBA{R: 255, A: 255}, color.RGBA{R: 30, G: 30, B: 80, A: 255}, 8, 4)
	if at := c.Img.At(60, 30); at.(color.RGBA).A == 0 {
		t.Error("label center is transparent")
	}
}

func TestFillRoundRect(t *testing.T) {
	c := NewCanvas(60, 60)
	bg := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	c.Fill(bg)
	rect := image.Rect(10, 10, 50, 50)
	c.FillRoundRect(rect, 6, color.RGBA{R: 200, G: 200, B: 200, A: 255})

	if at := c.Img.At(30, 30); at.(color.RGBA) == bg {
		t.Error("center not filled")
	}
	if at := c.Img.At(10, 10); at.(color.RGBA) != bg {
		t.Error("corner not clipped")
	}
}

func TestDividerV(t *testing.T) {
	c := NewCanvas(40, 40)
	c.DividerV(20, 5, 30, color.RGBA{R: 255, A: 255})
	for y := 5; y <= 30; y++ {
		if at := c.Img.At(20, y); at.(color.RGBA).R != 255 {
			t.Fatalf("pixel (20,%d) not set", y)
		}
	}
}

func TestWarningIcon(t *testing.T) {
	c := NewCanvas(30, 30)
	c.WarningIcon(image.Pt(15, 10), 8, color.RGBA{R: 255, A: 255})
	if at := c.Img.At(15, 10); at.(color.RGBA).R != 255 {
		t.Error("apex not drawn")
	}
	if at := c.Img.At(15, 17); at.(color.RGBA).R != 255 {
		t.Error("base center not drawn")
	}
	if at := c.Img.At(11, 17); at.(color.RGBA).R != 0 {
		t.Error("base should not extend beyond half width")
	}
}

func TestFillCircle(t *testing.T) {
	c := NewCanvas(60, 60)
	bg := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	c.Fill(bg)
	c.FillCircle(image.Pt(30, 30), 12, color.RGBA{R: 200, G: 200, B: 200, A: 255})

	if at := c.Img.At(30, 30); at.(color.RGBA) == bg {
		t.Error("center not filled")
	}
	if at := c.Img.At(30, 43); at.(color.RGBA) != bg {
		t.Error("outside radius should stay background")
	}
	// Circle clipping: a circle centered off-canvas still paints its on-canvas part.
	c2 := NewCanvas(30, 30)
	c2.FillCircle(image.Pt(5, 5), 6, color.RGBA{R: 255, A: 255})
	if at := c2.Img.At(5, 0); at.(color.RGBA).R != 255 {
		t.Error("clipped circle edge not painted")
	}
}

func TestFitRect(t *testing.T) {
	// Landscape source into portrait bounds, contain => letterboxed vertically.
	r := fitRect(image.Rect(0, 0, 100, 200), 200, 100, FitContain)
	if r.Dx() != 100 || r.Dy() != 50 {
		t.Errorf("contain = %v, want 100x50", r)
	}
	if r.Min.Y != (200-50)/2 {
		t.Errorf("contain not vertically centered: %v", r)
	}
	// Cover always fills bounds (cropping overflow) and preserves aspect ratio.
	r = fitRect(image.Rect(0, 0, 100, 200), 200, 100, FitCover)
	if r.Dx() < 100 || r.Dy() < 200 {
		t.Errorf("cover does not fill bounds: %v", r)
	}
	if r.Dx() != r.Dy()*2 {
		t.Errorf("cover aspect not preserved: %v", r)
	}
}

func TestDrawImageFit(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			src.Set(x, y, color.RGBA{R: 200, G: 30, B: 30, A: 255})
		}
	}
	c := NewCanvas(80, 80)
	c.Fill(color.RGBA{R: 5, G: 5, B: 5, A: 255})
	c.DrawImageFit(image.Rect(0, 0, 80, 80), src, FitContain)
	if at := c.Img.At(40, 40); at.(color.RGBA).R < 100 {
		t.Error("scaled image not rendered")
	}
	if at := c.Img.At(40, 1); at.(color.RGBA).R > 50 {
		t.Error("letterbox margin not background")
	}
}

func countLit(c *Canvas, bounds image.Rectangle) int {
	n := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			if c.Img.At(x, y).(color.RGBA).A != 0 {
				n++
			}
		}
	}
	return n
}
