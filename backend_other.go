//go:build !linux

package main

import (
	"fmt"

	"framego/render"
)

// openBackend only supports the PNG snapshot target off Linux.
func openBackend(kind, outPath, fbPath string) (render.Backend, string, error) {
	if kind == "fb" {
		return nil, "", fmt.Errorf("framebuffer backend requires linux")
	}
	return render.NewPNGBackend(outPath), "", nil
}
