package render

import "image"

// Backend presents rendered frames to a physical or virtual display target.
type Backend interface {
	// Present delivers a completed frame to the display.
	Present(img *image.RGBA) error
}
