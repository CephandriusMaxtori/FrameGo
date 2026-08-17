//go:build linux

package main

import (
	"framego/render"
)

// openBackend selects a display backend. "auto" prefers the kernel
// framebuffer and falls back to a PNG snapshot target.
func openBackend(kind, outPath, fbPath string) (render.Backend, string, error) {
	switch kind {
	case "fb":
		b, err := render.NewFramebuffer(fbPath)
		return b, "", err
	case "png":
		return render.NewPNGBackend(outPath), "", nil
	default:
		if b, err := render.NewFramebuffer(fbPath); err == nil {
			return b, "", nil
		}
		return render.NewPNGBackend(outPath), "framebuffer unavailable; falling back to PNG target", nil
	}
}
