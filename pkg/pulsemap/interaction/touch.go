// Package interaction provides pan, zoom, node selection, and navigation.
// This file implements touch input handling for mobile platforms.
// Per ROADMAP.md Priority 3, mobile platform support requires touch input adaptation.
package interaction

import (
	"math"
)

// TouchState tracks touch input for gesture recognition.
// Supports single-touch pan, two-finger pinch-to-zoom, and tap gestures.
type TouchState struct {
	// Active touches indexed by touch ID
	touches map[int]*Touch

	// Gesture state
	gestureType     GestureType
	pinchStartDist  float64
	pinchStartScale float64

	// Tap detection
	tapStartX, tapStartY float64
	tapStartTime         int64
	tapMoved             bool

	// Double-tap detection
	lastTapX, lastTapY float64
	lastTapTime        int64
}

// Touch represents a single touch point.
type Touch struct {
	ID   int
	X, Y float64
}

// GestureType indicates the current gesture being performed.
type GestureType int

const (
	// GestureNone indicates no active gesture.
	GestureNone GestureType = iota
	// GesturePan indicates a single-finger pan gesture.
	GesturePan
	// GesturePinch indicates a two-finger pinch-to-zoom gesture.
	GesturePinch
)

// Tap detection thresholds
const (
	// TapMaxDistance is the maximum movement (in pixels) allowed for a tap.
	TapMaxDistance = 20.0
	// TapMaxDuration is the maximum duration (in ticks at 60fps) for a tap.
	TapMaxDuration = 30 // ~500ms at 60fps
	// DoubleTapMaxInterval is the maximum interval between taps for a double-tap (in ticks at 60fps).
	DoubleTapMaxInterval = 30 // ~500ms at 60fps
	// DoubleTapMaxDistance is the maximum distance between tap positions for a double-tap.
	DoubleTapMaxDistance = 50.0
)

// NewTouchState creates a new touch state tracker.
func NewTouchState() *TouchState {
	return &TouchState{
		touches: make(map[int]*Touch),
	}
}

// TouchCount returns the number of active touches.
func (t *TouchState) TouchCount() int {
	return len(t.touches)
}

// GestureType returns the current gesture type.
func (t *TouchState) GestureType() GestureType {
	return t.gestureType
}

// HandleTouchStart processes a touch start event.
func (t *TouchState) HandleTouchStart(id int, x, y float64, tickCount int64) {
	t.touches[id] = &Touch{ID: id, X: x, Y: y}

	switch len(t.touches) {
	case 1:
		// Single touch: could be pan or tap
		t.gestureType = GesturePan
		t.tapStartX = x
		t.tapStartY = y
		t.tapStartTime = tickCount
		t.tapMoved = false
	case 2:
		// Two touches: pinch gesture
		t.gestureType = GesturePinch
		t.pinchStartDist = t.twoTouchDistance()
		t.tapMoved = true // Cancel tap detection
	default:
		// More than two touches: cancel all gestures
		t.gestureType = GestureNone
		t.tapMoved = true
	}
}

// HandleTouchMove processes a touch move event.
// Returns pan delta (dx, dy) for pan gesture, or zoom factor for pinch gesture.
func (t *TouchState) HandleTouchMove(id int, x, y float64) (dx, dy, zoomFactor float64) {
	touch, ok := t.touches[id]
	if !ok {
		return 0, 0, 1.0
	}

	prevX, prevY := touch.X, touch.Y
	touch.X = x
	touch.Y = y

	// Check if moved too far for tap
	if !t.tapMoved {
		tapDist := math.Sqrt((x-t.tapStartX)*(x-t.tapStartX) + (y-t.tapStartY)*(y-t.tapStartY))
		if tapDist > TapMaxDistance {
			t.tapMoved = true
		}
	}

	switch t.gestureType {
	case GesturePan:
		// Return delta for camera panning
		return x - prevX, y - prevY, 1.0

	case GesturePinch:
		// Calculate zoom factor from pinch distance change
		if len(t.touches) == 2 {
			currentDist := t.twoTouchDistance()
			if t.pinchStartDist > 0 {
				zoomFactor = currentDist / t.pinchStartDist
				t.pinchStartDist = currentDist // Reset for incremental zoom
				return 0, 0, zoomFactor
			}
		}
	}

	return 0, 0, 1.0
}

// HandleTouchEnd processes a touch end event.
// Returns isTap (true for single tap), isDoubleTap (true for double tap), and (x, y) position.
func (t *TouchState) HandleTouchEnd(id int, tickCount int64) (isTap, isDoubleTap bool, x, y float64) {
	touch, ok := t.touches[id]
	if ok {
		x = touch.X
		y = touch.Y
		delete(t.touches, id)
	}

	// Check for tap gesture
	isTap = !t.tapMoved && (tickCount-t.tapStartTime) < TapMaxDuration && len(t.touches) == 0

	// Check for double-tap gesture
	if isTap {
		dx := x - t.lastTapX
		dy := y - t.lastTapY
		dist := math.Sqrt(dx*dx + dy*dy)
		interval := tickCount - t.lastTapTime

		if interval < DoubleTapMaxInterval && dist < DoubleTapMaxDistance && t.lastTapTime > 0 {
			isDoubleTap = true
			// Reset double-tap state after detecting one
			t.lastTapTime = 0
		} else {
			// Record this tap for potential double-tap
			t.lastTapX = x
			t.lastTapY = y
			t.lastTapTime = tickCount
		}
	}

	// Update gesture state
	switch len(t.touches) {
	case 0:
		t.gestureType = GestureNone
	case 1:
		t.gestureType = GesturePan
	default:
		t.gestureType = GesturePinch
		t.pinchStartDist = t.twoTouchDistance()
	}

	return isTap, isDoubleTap, x, y
}

// twoTouchDistance calculates the distance between two touch points.
func (t *TouchState) twoTouchDistance() float64 {
	if len(t.touches) != 2 {
		return 0
	}

	var touches []*Touch
	for _, touch := range t.touches {
		touches = append(touches, touch)
	}

	dx := touches[0].X - touches[1].X
	dy := touches[0].Y - touches[1].Y
	return math.Sqrt(dx*dx + dy*dy)
}

// PinchCenter returns the center point between two touch points.
// Used as the zoom focus point for pinch-to-zoom.
func (t *TouchState) PinchCenter() (x, y float64) {
	if len(t.touches) != 2 {
		return 0, 0
	}

	var touches []*Touch
	for _, touch := range t.touches {
		touches = append(touches, touch)
	}

	return (touches[0].X + touches[1].X) / 2, (touches[0].Y + touches[1].Y) / 2
}

// Reset clears all touch state.
func (t *TouchState) Reset() {
	t.touches = make(map[int]*Touch)
	t.gestureType = GestureNone
	t.pinchStartDist = 0
	t.tapMoved = false
	t.lastTapTime = 0
}
