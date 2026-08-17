// Package fonts loads and caches TrueType/OpenType faces from fonts embedded
// in the binary or loaded from disk, providing a zero-dependency typography
// pipeline.
package fonts

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// Weight selects a glyph variant from the embedded Go fonts.
type Weight int

const (
	Regular Weight = iota
	Medium
	Bold
)

var (
	mu    sync.Mutex
	cache = map[string]font.Face{}
)

// Face returns a cached face for the given point size and weight.
func Face(size float64, weight Weight) font.Face {
	key := fmt.Sprintf("embedded:%d:%g", weight, size)
	mu.Lock()
	defer mu.Unlock()
	if f, ok := cache[key]; ok {
		return f
	}
	var src []byte
	switch weight {
	case Bold:
		src = gobold.TTF
	case Medium:
		src = gomedium.TTF
	default:
		src = goregular.TTF
	}
	face, err := newFace(key, src, size)
	if err != nil {
		panic(err) // embedded fonts are valid by construction
	}
	return face
}

// LoadFile parses a TrueType/OpenType font file from disk and returns a cached
// face. Unlike the embedded faces, failures return an error instead of
// panicking so misconfigured fonts degrade gracefully.
func LoadFile(path string, size float64) (font.Face, error) {
	key := fmt.Sprintf("file:%s:%g", path, size)
	mu.Lock()
	if f, ok := cache[key]; ok {
		mu.Unlock()
		return f, nil
	}
	mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read font %q: %w", path, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if f, ok := cache[key]; ok {
		return f, nil
	}
	face, err := newFace(key, data, size)
	if err != nil {
		return nil, fmt.Errorf("parse font %q: %w", path, err)
	}
	return face, nil
}

// newFace parses sfnt data and caches an opentype-backed face under key.
// The caller must hold mu.
func newFace(key string, data []byte, size float64) (font.Face, error) {
	fnt, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	cache[key] = face
	return face, nil
}
