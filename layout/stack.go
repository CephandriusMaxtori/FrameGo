package layout

import "image"

// Stack splits a zone's bounding box vertically into count equal slots,
// separated by gap pixels, for modules assigned to the same zone. The returned
// slice has length count and never exceeds the input bounds.
func Stack(bounds image.Rectangle, count, gap int) []image.Rectangle {
	if count <= 0 {
		return nil
	}
	slots := make([]image.Rectangle, count)
	avail := bounds.Dy()
	total := count
	if total > 1 {
		total = count - 1
	}
	slotH := (avail - (count-1)*gap) / count
	if slotH < 1 {
		slotH = 1
	}
	for i := 0; i < count; i++ {
		y := bounds.Min.Y + i*(slotH+gap)
		slots[i] = image.Rect(bounds.Min.X, y, bounds.Max.X, y+slotH)
	}
	return slots
}
