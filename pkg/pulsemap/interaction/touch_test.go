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
	isTap, _, _ := ts.HandleTouchEnd(1, 10)
	if isTap {
		t.Error("expected not a tap after moving 50px")
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

	// End quickly
	isTap, x, y := ts.HandleTouchEnd(1, 10) // Within TapMaxDuration
	if !isTap {
		t.Error("expected tap gesture")
	}
	if x != 105 || y != 103 {
		t.Errorf("expected tap at (105, 103), got (%v, %v)", x, y)
	}
}

func TestTapCancelledByDistance(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	// Move beyond tap threshold
	ts.HandleTouchMove(1, 130, 130) // > TapMaxDistance
	isTap, _, _ := ts.HandleTouchEnd(1, 10)
	if isTap {
		t.Error("tap should be cancelled by movement")
	}
}

func TestTapCancelledByDuration(t *testing.T) {
	ts := NewTouchState()

	ts.HandleTouchStart(1, 100, 100, 0)
	ts.HandleTouchMove(1, 102, 102)                        // Small movement
	isTap, _, _ := ts.HandleTouchEnd(1, TapMaxDuration+10) // Too long
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
