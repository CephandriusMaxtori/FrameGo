//go:build !linux || (!amd64 && !arm64)

package render

import "errors"

// Framebuffer is unavailable off supported Linux targets (amd64/arm64).
type Framebuffer struct{}

// NewFramebuffer always fails off the supported Linux targets.
func NewFramebuffer(path string) (*Framebuffer, error) {
	return nil, errors.New("framebuffer target requires linux/amd64 or linux/arm64")
}

// Close is a no-op on unsupported platforms.
func (f *Framebuffer) Close() error { return nil }
