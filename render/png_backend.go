package render

import (
	"image"
	"image/png"
	"os"
)

// PNGBackend writes each presented frame to a PNG file. It is used for
// development previews and offline snapshotting on non-Linux hosts.
type PNGBackend struct {
	// Path is the destination file. If empty, frames are discarded.
	Path string
}

// NewPNGBackend creates a PNG backend targeting path.
func NewPNGBackend(path string) *PNGBackend {
	return &PNGBackend{Path: path}
}

// Present encodes img to a PNG file.
func (p *PNGBackend) Present(img *image.RGBA) error {
	if p.Path == "" {
		return nil
	}
	f, err := os.Create(p.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
