package interaction

import (
	"testing"
)

func TestNewTouchState(t *testing.T) {
	ts := NewTouchState()
	if ts == nil {
		t.Fatal("NewTouchState returned nil")
	}
	if ts.TouchCount() != 0 {
		t.Errorf("expected 0 touches, got %d", ts.TouchCount())
	}
	if ts.GestureType() != GestureNone {
		t.Errorf("expected GestureNone, got %v", ts.GestureType())
	}
}

func TestSingleTouchPan(t *testing.T) {
	ts := NewTouchState()

	// Start touch
	ts.HandleTouchStart(1, 100, 100, 0)
	if ts.TouchCount() != 1 {
		t.Errorf("expected 1 touch, got %d", ts.TouchCount())
	}
	if ts.GestureType() != GesturePan {
		t.Errorf("expected GesturePan, got %v", ts.GestureType())
	}

	// Move touch
	dx, dy, zoom := ts.HandleTouchMove(1, 150, 120)
	if dx != 50 || dy != 20 {
		t.Errorf("expected dx=50, dy=20, got dx=%v, dy=%v", dx, dy)
	}
	if zoom != 1.0 {
		t.Errorf("expected zoom=1.0, got %v", zoom)
	}

	// End touch - not a tap because moved too far
	isTap, isDoubleTap, _, _ := ts.HandleTouchEnd(1, 10)
	if isTap {
		t.Error("expected not a tap after moving 50px")
	}
	if isDoubleTap {
		t.Error("unexpected double tap")
	}
	if ts.TouchCount() != 0 {
		t.Errorf("expected 0 touches after end, got %d", ts.TouchCount())
	}
}

func TestTapGesture(t *testing.T) {
	ts := NewTouchState()

	// Start touch
	ts.HandleTouchStart(1, 100, 100, 0)

	// Move slightly (within tap threshold)
	ts.HandleTouchMove(1, 105, 103)

	// End quickly — per AUDIT.md debounce fix, HandleTouchEnd returns isTap=false on the first tap.
	// The tap is deferred via PollPendingTap until DoubleTapMaxInterval ticks elapse.
	isTap, isDoubleTap, _, _ := ts.HandleTouchEnd(1, 10) // Within TapMaxDuration
	if isTap {
		t.Error("HandleTouchEnd should not report isTap=true immediately (debounce window active)")
	}
	if isDoubleTap {
		t.Error("single tap should not be double tap")
	}

	// Before the window expires, PollPendingTap should not fire.
	ok, _, _ := ts.PollPendingTap(10 + DoubleTapMaxInterval - 1)
	if ok {
		t.Error("PollPendingTap should not fire before debounce window expires")
	}

	// After the window expires, PollPendingTap emits the deferred tap at the correct position.
	ok, x, y := ts.PollPendingTap(10 + DoubleTapMaxInterval)
	if !ok {
		t.Error("PollPendingTap should fire after debounce window expires")
	}
	if x != 105 || y != 103 {
		t.Errorf("expected tap at (105, 103), got (%v, %v)", x, y)
	}

	// A second poll should return false (tap consumed).
	ok2, _, _ := ts.PollPendingTap(10 + DoubleTapMaxInterval + 1)
	if ok2 {
		t.Error("PollPendingTap should not fire twice")
	}
}

func TestTapCancelledByDistance(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	// Move beyond tap threshold
	ts.HandleTouchMove(1, 130, 130) // > TapMaxDistance
	isTap, _, _, _ := ts.HandleTouchEnd(1, 10)
	if isTap {
		t.Error("tap should be cancelled by movement")
	}
}

func TestTapCancelledByDuration(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchMove(1, 102, 102)                           // Small movement
	isTap, _, _, _ := ts.HandleTouchEnd(1, TapMaxDuration+10) // Too long
	if isTap {
		t.Error("tap should be cancelled by duration")
	}
}

func TestPinchToZoom(t *testing.T) {
	ts := NewTouchState()

	// Start two touches
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchStart(2, 200, 100, 0) // 100px apart
	if ts.GestureType() != GesturePinch {
		t.Errorf("expected GesturePinch, got %v", ts.GestureType())
	}
	if ts.TouchCount() != 2 {
		t.Errorf("expected 2 touches, got %d", ts.TouchCount())
	}

	// Move touches apart (zoom in)
	ts.HandleTouchMove(1, 50, 100)                // Move left
	_, _, zoom := ts.HandleTouchMove(2, 250, 100) // Move right, now 200px apart
	if zoom <= 1.0 {
		t.Errorf("expected zoom > 1.0 for pinch out, got %v", zoom)
	}

	// Check pinch center
	cx, cy := ts.PinchCenter()
	if cx != 150 || cy != 100 {
		t.Errorf("expected center (150, 100), got (%v, %v)", cx, cy)
	}
}

func TestPinchCenter(t *testing.T) {
	ts := NewTouchState()

	// No touches
	cx, cy := ts.PinchCenter()
	if cx != 0 || cy != 0 {
		t.Errorf("expected (0, 0) with no touches, got (%v, %v)", cx, cy)
	}

	// Single touch
	ts.HandleTouchStart(1, 100, 100, 0)
	cx, cy = ts.PinchCenter()
	if cx != 0 || cy != 0 {
		t.Errorf("expected (0, 0) with single touch, got (%v, %v)", cx, cy)
	}

	// Two touches
	ts.HandleTouchStart(2, 200, 200, 0)
	cx, cy = ts.PinchCenter()
	if cx != 150 || cy != 150 {
		t.Errorf("expected (150, 150), got (%v, %v)", cx, cy)
	}
}

func TestTouchReset(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchStart(2, 200, 200, 0)

	ts.Reset()

	if ts.TouchCount() != 0 {
		t.Errorf("expected 0 touches after reset, got %d", ts.TouchCount())
	}
	if ts.GestureType() != GestureNone {
		t.Errorf("expected GestureNone after reset, got %v", ts.GestureType())
	}
}

func TestTransitionPinchToPan(t *testing.T) {
	ts := NewTouchState()

	// Start with two touches (pinch)
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchStart(2, 200, 200, 0)
	if ts.GestureType() != GesturePinch {
		t.Errorf("expected GesturePinch, got %v", ts.GestureType())
	}

	// Release one touch
	ts.HandleTouchEnd(2, 10)
	if ts.GestureType() != GesturePan {
		t.Errorf("expected GesturePan after releasing one touch, got %v", ts.GestureType())
	}
	if ts.TouchCount() != 1 {
		t.Errorf("expected 1 touch, got %d", ts.TouchCount())
	}
}

func TestHandleTouchMoveUnknownID(t *testing.T) {
	ts := NewTouchState()

	// Move non-existent touch
	dx, dy, zoom := ts.HandleTouchMove(999, 100, 100)
	if dx != 0 || dy != 0 || zoom != 1.0 {
		t.Errorf("expected (0, 0, 1.0) for unknown touch, got (%v, %v, %v)", dx, dy, zoom)
	}
}

func TestThreeTouchCancelsGesture(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchStart(2, 200, 200, 0)
	ts.HandleTouchStart(3, 300, 300, 0) // Three touches

	if ts.GestureType() != GestureNone {
		t.Errorf("expected GestureNone with 3 touches, got %v", ts.GestureType())
	}
}

func TestDoubleTapGesture(t *testing.T) {
	ts := NewTouchState()

	// First tap — deferred by debounce window; HandleTouchEnd returns isTap=false.
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchMove(1, 102, 102) // Small movement
	isTap, isDoubleTap, _, _ := ts.HandleTouchEnd(1, 10)
	if isTap {
		t.Error("first tap should be deferred (isTap must be false from HandleTouchEnd)")
	}
	if isDoubleTap {
		t.Error("first tap should not be double tap")
	}

	// Second tap quickly and nearby — triggers double-tap; pending single-tap is cancelled.
	ts.HandleTouchStart(2, 105, 103, 12) // Within interval and distance
	ts.HandleTouchMove(2, 106, 104)
	isTap2, isDoubleTap2, x, y := ts.HandleTouchEnd(2, 20)
	if isTap2 {
		t.Error("second tap in double-tap sequence should have isTap=false")
	}
	if !isDoubleTap2 {
		t.Error("expected double tap on second tap")
	}
	if x != 106 || y != 104 {
		t.Errorf("expected double tap at (106, 104), got (%v, %v)", x, y)
	}

	// Pending single-tap should have been cancelled by the double-tap.
	ok, _, _ := ts.PollPendingTap(20 + DoubleTapMaxInterval)
	if ok {
		t.Error("pending single-tap should be cancelled when double-tap fires")
	}
}

func TestLongPressPollFiresOnce(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)

	// Before threshold: should not fire.
	ok, _, _ := ts.PollLongPress(LongPressMinDuration - 1)
	if ok {
		t.Fatal("long-press should not fire before threshold")
	}

	// At threshold: fires once.
	ok, x, y := ts.PollLongPress(LongPressMinDuration)
	if !ok {
		t.Fatal("expected long-press to fire at threshold")
	}
	if x != 100 || y != 100 {
		t.Fatalf("expected long-press at (100,100), got (%v,%v)", x, y)
	}

	// Re-poll while same touch is held: should not fire again.
	ok2, _, _ := ts.PollLongPress(LongPressMinDuration + 10)
	if ok2 {
		t.Fatal("long-press should fire only once per hold")
	}
}

func TestLongPressCancelledByMovement(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchMove(1, 140, 100) // Move beyond LongPressMaxDistance.

	ok, _, _ := ts.PollLongPress(LongPressMinDuration + 1)
	if ok {
		t.Fatal("long-press should not fire when movement exceeds threshold")
	}
}

func TestDoubleTapCancelledByDistance(t *testing.T) {
	ts := NewTouchState()

	// First tap
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchEnd(1, 10)

	// Second tap too far away
	ts.HandleTouchStart(2, 200, 200, 12) // > DoubleTapMaxDistance
	_, isDoubleTap, _, _ := ts.HandleTouchEnd(2, 20)
	if isDoubleTap {
		t.Error("double tap should be cancelled by distance")
	}
}

func TestDoubleTapCancelledByInterval(t *testing.T) {
	ts := NewTouchState()

	// First tap
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchEnd(1, 10)

	// Second tap too late
	ts.HandleTouchStart(2, 105, 103, DoubleTapMaxInterval+20) // Too late
	_, isDoubleTap, _, _ := ts.HandleTouchEnd(2, DoubleTapMaxInterval+25)
	if isDoubleTap {
		t.Error("double tap should be cancelled by interval")
	}
}

func TestDoubleTapResets(t *testing.T) {
	ts := NewTouchState()

	// First double-tap sequence
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchEnd(1, 10)
	ts.HandleTouchStart(2, 105, 103, 12)
	_, isDoubleTap, _, _ := ts.HandleTouchEnd(2, 20)
	if !isDoubleTap {
		t.Error("expected double tap")
	}

	// Third tap should not be a triple tap
	ts.HandleTouchStart(3, 108, 105, 25)
	_, isDoubleTap2, _, _ := ts.HandleTouchEnd(3, 30)
	if isDoubleTap2 {
		t.Error("third tap should not trigger double tap")
	}
}

func TestResetClearsDoubleTapState(t *testing.T) {
	ts := NewTouchState()

	// First tap
	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchEnd(1, 10)

	// Reset
	ts.Reset()

	// Second tap after reset should not trigger double tap
	ts.HandleTouchStart(2, 105, 103, 12)
	_, isDoubleTap, _, _ := ts.HandleTouchEnd(2, 20)
	if isDoubleTap {
		t.Error("double tap should be cleared by Reset")
	}
}

func TestCameraAnimateToWithZoom(t *testing.T) {
	c := NewCamera()

	// Animate to a position with zoom
	c.AnimateToWithZoom(200, 300, 3.0)

	if !c.Animating {
		t.Error("expected Animating to be true")
	}
	if c.TargetX != 200 || c.TargetY != 300 {
		t.Errorf("expected target (200, 300), got (%f, %f)", c.TargetX, c.TargetY)
	}
	if c.TargetScale != 3.0 {
		t.Errorf("expected target scale 3.0, got %f", c.TargetScale)
	}

	// Run animation
	for c.Animating {
		c.Update()
	}

	// Should be at target
	if c.X != 200 || c.Y != 300 {
		t.Errorf("expected position (200, 300), got (%f, %f)", c.X, c.Y)
	}
	if c.Scale != 3.0 {
		t.Errorf("expected scale 3.0, got %f", c.Scale)
	}
}
