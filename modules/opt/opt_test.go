package opt

import (
	"image/color"
	"testing"
	"time"
)

func TestReaders(t *testing.T) {
	opts := map[string]any{
		"name":  "Berlin",
		"count": 5.0,
		"ratio": 0.5,
		"on":    true,
		"color": "#ff0000",
	}
	if got := Str(opts, "name", "x"); got != "Berlin" {
		t.Errorf("Str = %q", got)
	}
	if got := Str(opts, "missing", "x"); got != "x" {
		t.Errorf("Str default = %q", got)
	}
	if got := Int(opts, "count", 9); got != 5 {
		t.Errorf("Int = %d", got)
	}
	if got := Int(opts, "missing", 9); got != 9 {
		t.Errorf("Int default = %d", got)
	}
	if got := Float(opts, "ratio", 1); got != 0.5 {
		t.Errorf("Float = %v", got)
	}
	if got := Bool(opts, "on", false); !got {
		t.Errorf("Bool = %v", got)
	}
	if got := Color(opts, "color", color.RGBA{}); got.R != 0xff || got.G != 0 || got.B != 0 {
		t.Errorf("Color = %v", got)
	}
	if got := Duration(map[string]any{"s": 2.0}, "s", 1); got != 2*time.Second {
		t.Errorf("Duration = %v", got)
	}
}

func TestStringEncoded(t *testing.T) {
	opts := map[string]any{"count": "7", "on": "true", "ratio": "1.5"}
	if got := Int(opts, "count", 0); got != 7 {
		t.Errorf("Int = %d", got)
	}
	if got := Bool(opts, "on", false); !got {
		t.Errorf("Bool = %v", got)
	}
	if got := Float(opts, "ratio", 0); got != 1.5 {
		t.Errorf("Float = %v", got)
	}
}
