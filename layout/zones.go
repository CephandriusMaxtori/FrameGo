// Package layout implements the 12-zone grid compositor that resolves module
// placements against a display's pixel dimensions.
package layout

// ZoneID identifies one spatial container of the display grid.
type ZoneID string

// The full zone set mirrors the smart-mirror placement paradigm documented in
// the FrameGo technical design: five horizontal bands with column slots, plus
// full-width bars on the top and bottom bands.
const (
	ZoneTopLeft   ZoneID = "top-left"
	ZoneTopCenter ZoneID = "top-center"
	ZoneTopRight  ZoneID = "top-right"
	ZoneTopBar    ZoneID = "top-bar"

	ZoneUpperLeft   ZoneID = "upper-left"
	ZoneUpperCenter ZoneID = "upper-center"
	ZoneUpperRight  ZoneID = "upper-right"

	ZoneMiddleLeft   ZoneID = "middle-left"
	ZoneMiddleCenter ZoneID = "middle-center"
	ZoneMiddleRight  ZoneID = "middle-right"

	ZoneLowerLeft   ZoneID = "lower-left"
	ZoneLowerCenter ZoneID = "lower-center"
	ZoneLowerRight  ZoneID = "lower-right"

	ZoneBottomLeft   ZoneID = "bottom-left"
	ZoneBottomCenter ZoneID = "bottom-center"
	ZoneBottomRight  ZoneID = "bottom-right"
	ZoneBottomBar    ZoneID = "bottom-bar"
)

// AllZones lists every zone in top-to-bottom, left-to-right order.
var AllZones = []ZoneID{
	ZoneTopLeft, ZoneTopCenter, ZoneTopRight, ZoneTopBar,
	ZoneUpperLeft, ZoneUpperCenter, ZoneUpperRight,
	ZoneMiddleLeft, ZoneMiddleCenter, ZoneMiddleRight,
	ZoneLowerLeft, ZoneLowerCenter, ZoneLowerRight,
	ZoneBottomLeft, ZoneBottomCenter, ZoneBottomRight, ZoneBottomBar,
}

// ValidZone reports whether id names a known zone.
func ValidZone(id string) bool {
	for _, z := range AllZones {
		if ZoneID(id) == z {
			return true
		}
	}
	return false
}

// bands is the vertical ordering of the five display bands.
var bands = []struct {
	zones []ZoneID
}{
	{zones: []ZoneID{ZoneTopLeft, ZoneTopCenter, ZoneTopRight, ZoneTopBar}},
	{zones: []ZoneID{ZoneUpperLeft, ZoneUpperCenter, ZoneUpperRight}},
	{zones: []ZoneID{ZoneMiddleLeft, ZoneMiddleCenter, ZoneMiddleRight}},
	{zones: []ZoneID{ZoneLowerLeft, ZoneLowerCenter, ZoneLowerRight}},
	{zones: []ZoneID{ZoneBottomLeft, ZoneBottomCenter, ZoneBottomRight, ZoneBottomBar}},
}
