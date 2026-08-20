//go:build linux && (amd64 || arm64)

package input

import (
	"testing"
	"time"

	"framego/engine"
)

func newTestEvdev(w, h int) (*Evdev, *engine.Bus) {
	bus := engine.NewBus()
	log := engine.NewLogger(nil)
	return &Evdev{
		bus:    bus,
		log:    log,
		width:  w,
		height: h,
		curX:   w / 2,
		curY:   h / 2,
		done:   make(chan struct{}),
	}, bus
}

func TestHandleEventMultitouchAbsY(t *testing.T) {
	ev, bus := newTestEvdev(1920, 1080)
	ch := bus.Subscribe(engine.TopicTouch)
	defer bus.Unsubscribe(engine.TopicTouch, ch)

	ev.handleEvent(inputEvent{Type: evAbs, Code: absMtPosX, Value: 2000})
	ev.handleEvent(inputEvent{Type: evAbs, Code: absMtPosY, Value: 3000})

	select {
	case ev := <-ch:
		te, ok := ev.Data.(TouchEvent)
		if !ok {
			t.Fatalf("expected TouchEvent, got %T", ev.Data)
		}
		if te.Phase != TouchDown {
			t.Fatalf("expected TouchDown, got %v", te.Phase)
		}
		expectedX := 2000 * 1920 / 4096
		expectedY := 3000 * 1080 / 4096
		if te.X != expectedX || te.Y != expectedY {
			t.Fatalf("expected (%d,%d), got (%d,%d)", expectedX, expectedY, te.X, te.Y)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for touch event")
	}
}

func TestHandleEventSingleTouchAbsY(t *testing.T) {
	ev, bus := newTestEvdev(800, 480)
	ch := bus.Subscribe(engine.TopicTouch)
	defer bus.Unsubscribe(engine.TopicTouch, ch)

	ev.handleEvent(inputEvent{Type: evAbs, Code: absX, Value: 1000})
	ev.handleEvent(inputEvent{Type: evAbs, Code: absY, Value: 2000})

	select {
	case ev := <-ch:
		te := ev.Data.(TouchEvent)
		if te.Phase != TouchDown {
			t.Fatalf("expected TouchDown, got %v", te.Phase)
		}
		expectedX := 1000 * 800 / 4096
		expectedY := 2000 * 480 / 4096
		if te.X != expectedX || te.Y != expectedY {
			t.Fatalf("expected (%d,%d), got (%d,%d)", expectedX, expectedY, te.X, te.Y)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for touch event")
	}
}

func TestHandleEventTouchUp(t *testing.T) {
	ev, bus := newTestEvdev(800, 480)
	ch := bus.Subscribe(engine.TopicTouch)
	defer bus.Unsubscribe(engine.TopicTouch, ch)

	ev.handleEvent(inputEvent{Type: evAbs, Code: absMtPosX, Value: 1000})
	ev.handleEvent(inputEvent{Type: evAbs, Code: absMtPosY, Value: 2000})
	<-ch // consume TouchDown

	ev.handleEvent(inputEvent{Type: evKey, Code: btnTouch, Value: 0})

	select {
	case ev := <-ch:
		te := ev.Data.(TouchEvent)
		if te.Phase != TouchUp {
			t.Fatalf("expected TouchUp, got %v", te.Phase)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for touch up event")
	}
}

func TestHandleEventRelativeMouse(t *testing.T) {
	ev, bus := newTestEvdev(1920, 1080)
	ch := bus.Subscribe(engine.TopicTouch)
	defer bus.Unsubscribe(engine.TopicTouch, ch)

	ev.handleEvent(inputEvent{Type: evRel, Code: relX, Value: 100})
	select {
	case ev := <-ch:
		te := ev.Data.(TouchEvent)
		if te.Phase != TouchDown {
			t.Fatalf("expected TouchDown on first rel move, got %v", te.Phase)
		}
		if te.X != 960+100 || te.Y != 540 {
			t.Fatalf("expected (%d,540), got (%d,%d)", 960+100, te.X, te.Y)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestRelativeMouseClamp(t *testing.T) {
	ev, bus := newTestEvdev(800, 480)
	_ = bus.Subscribe(engine.TopicTouch)

	ev.handleEvent(inputEvent{Type: evRel, Code: relX, Value: -9999})
	ev.mu.Lock()
	if ev.curX != 0 {
		t.Fatalf("expected curX=0 after clamping, got %d", ev.curX)
	}
	ev.mu.Unlock()

	ev.handleEvent(inputEvent{Type: evRel, Code: relX, Value: 99999})
	ev.mu.Lock()
	if ev.curX != 799 {
		t.Fatalf("expected curX=799 after clamping, got %d", ev.curX)
	}
	ev.mu.Unlock()
}

func TestRelativeMouseUp(t *testing.T) {
	ev, bus := newTestEvdev(800, 480)
	ch := bus.Subscribe(engine.TopicTouch)
	defer bus.Unsubscribe(engine.TopicTouch, ch)

	ev.handleEvent(inputEvent{Type: evRel, Code: relX, Value: 10})
	<-ch // consume TouchDown

	ev.handleEvent(inputEvent{Type: evKey, Code: btnLeft, Value: 0})

	select {
	case ev := <-ch:
		te := ev.Data.(TouchEvent)
		if te.Phase != TouchUp {
			t.Fatalf("expected TouchUp, got %v", te.Phase)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestHasBit(t *testing.T) {
	var bits [eviocgbitLen]byte
	bits[btnLeft/8] |= 1 << (btnLeft % 8)
	if !hasBit(bits, btnLeft) {
		t.Fatal("expected hasBit to be true")
	}
	if hasBit(bits, btnTouch) {
		t.Fatal("expected hasBit to be false")
	}
}
