// Package opt provides typed readers for module option maps so widgets can
// decode configuration values from the free-form map[string]any surface used
// by config.json and the admin UI.
package opt

import (
	"image/color"
	"strconv"
	"time"

	"framego/render"
)

// Str returns the string value at key, or def when missing or empty.
func Str(opts map[string]any, key, def string) string {
	if v, ok := opts[key].(string); ok && v != "" {
		return v
	}
	return def
}

// Int returns the integer value at key, or def when missing or invalid.
func Int(opts map[string]any, key string, def int) int {
	if v, ok := opts[key].(float64); ok {
		return int(v)
	}
	if v, ok := opts[key].(int); ok {
		return v
	}
	if s, ok := opts[key].(string); ok {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return def
}

// Float returns the float value at key, or def when missing or invalid.
func Float(opts map[string]any, key string, def float64) float64 {
	if v, ok := opts[key].(float64); ok {
		return v
	}
	if v, ok := opts[key].(int); ok {
		return float64(v)
	}
	if s, ok := opts[key].(string); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return def
}

// Bool returns the boolean value at key, or def when missing or invalid.
func Bool(opts map[string]any, key string, def bool) bool {
	if v, ok := opts[key].(bool); ok {
		return v
	}
	if s, ok := opts[key].(string); ok {
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	}
	return def
}

// Color parses a "#rrggbb" string option into an opaque RGBA color, falling
// back to def on missing or unparseable input.
func Color(opts map[string]any, key string, def color.RGBA) color.RGBA {
	return render.ParseHexColor(Str(opts, key, ""), def)
}

// Duration returns the number of seconds at key as a duration, or def.
func Duration(opts map[string]any, key string, def float64) time.Duration {
	return time.Duration(Float(opts, key, def) * float64(time.Second))
}
