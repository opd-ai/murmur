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

	// Pending single-tap debounce: per AUDIT.md MEDIUM fix, first tap is held
	// until DoubleTapMaxInterval expires to avoid firing single-tap on double-taps.
	pendingTapX, pendingTapY float64
	pendingTapTick           int64 // tick when pending tap was recorded; 0 = none

	// Long-press detection state.
	lastLongPressTouchID int // touch ID that already fired long-press for current hold; 0 = none
}

// Touch represents a single touch point.
type Touch struct {
	ID int
	X  float64
	Y  float64

	StartX, StartY float64
	StartTick      int64
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
	// This value also serves as the single-tap debounce window: HandleTouchEnd defers the single-tap
	// event by this many ticks (via PollPendingTap) to avoid firing a single-tap prematurely on
	// the first tap of a double-tap sequence.
	DoubleTapMaxInterval = 30 // ~500ms at 60fps
	// DoubleTapMaxDistance is the maximum distance between tap positions for a double-tap.
	DoubleTapMaxDistance = 50.0
	// LongPressMinDuration is the minimum hold time (in ticks at 60fps) to trigger long-press.
	LongPressMinDuration = 36 // ~600ms at 60fps
	// LongPressMaxDistance is the maximum finger movement allowed while holding.
	LongPressMaxDistance = 16.0
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
	t.touches[id] = &Touch{ID: id, X: x, Y: y, StartX: x, StartY: y, StartTick: tickCount}
	t.lastLongPressTouchID = 0

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

	t.updateTapMovementState(x, y)

	return t.computeGestureResult(prevX, prevY, x, y)
}

// updateTapMovementState checks if touch moved beyond tap threshold.
func (t *TouchState) updateTapMovementState(x, y float64) {
	if !t.tapMoved {
		tapDist := math.Sqrt((x-t.tapStartX)*(x-t.tapStartX) + (y-t.tapStartY)*(y-t.tapStartY))
		if tapDist > TapMaxDistance {
			t.tapMoved = true
		}
	}
}

// computeGestureResult calculates deltas and zoom based on gesture type.
func (t *TouchState) computeGestureResult(prevX, prevY, x, y float64) (dx, dy, zoomFactor float64) {
	switch t.gestureType {
	case GesturePan:
		return t.computePanDelta(prevX, prevY, x, y)
	case GesturePinch:
		return t.computePinchZoom()
	}
	return 0, 0, 1.0
}

// computePanDelta returns camera pan delta.
func (t *TouchState) computePanDelta(prevX, prevY, x, y float64) (dx, dy, zoomFactor float64) {
	return x - prevX, y - prevY, 1.0
}

// computePinchZoom calculates zoom factor from pinch gesture.
func (t *TouchState) computePinchZoom() (dx, dy, zoomFactor float64) {
	if len(t.touches) != 2 {
		return 0, 0, 1.0
	}

	currentDist := t.twoTouchDistance()
	if t.pinchStartDist <= 0 {
		return 0, 0, 1.0
	}

	zoomFactor = currentDist / t.pinchStartDist
	t.pinchStartDist = currentDist
	return 0, 0, zoomFactor
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
	if t.lastLongPressTouchID == id {
		t.lastLongPressTouchID = 0
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
			isTap = false // Double-tap does not also fire as a single-tap.
			// Double-tap consumed: discard the pending single-tap.
			t.pendingTapTick = 0
			// Reset double-tap state after detecting one.
			t.lastTapTime = 0
		} else {
			// Record as pending single-tap; defer emission until DoubleTapMaxInterval expires.
			t.pendingTapX = x
			t.pendingTapY = y
			t.pendingTapTick = tickCount
			// Also record for future double-tap detection.
			t.lastTapX = x
			t.lastTapY = y
			t.lastTapTime = tickCount
			// Single-tap is deferred; don't report it yet.
			isTap = false
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

// PollLongPress checks whether the active single-touch hold has become a long-press.
// Returns (true, x, y) exactly once per touch hold when the threshold is crossed.
func (t *TouchState) PollLongPress(tickCount int64) (isLongPress bool, x, y float64) {
	if len(t.touches) != 1 || t.gestureType != GesturePan {
		return false, 0, 0
	}

	var touch *Touch
	for _, candidate := range t.touches {
		touch = candidate
		break
	}
	if touch == nil {
		return false, 0, 0
	}
	if t.lastLongPressTouchID == touch.ID {
		return false, 0, 0
	}
	if tickCount-touch.StartTick < LongPressMinDuration {
		return false, 0, 0
	}

	dx := touch.X - touch.StartX
	dy := touch.Y - touch.StartY
	if math.Sqrt(dx*dx+dy*dy) > LongPressMaxDistance {
		return false, 0, 0
	}

	t.lastLongPressTouchID = touch.ID
	return true, touch.X, touch.Y
}

// PollPendingTap checks whether a deferred single-tap has waited long enough
// to be emitted (double-tap window expired). Call this once per game tick.
// Returns (true, x, y) when the pending tap should now fire; (false, 0, 0) otherwise.
// Per AUDIT.md MEDIUM finding: single-taps are deferred by DoubleTapMaxInterval ticks
// so they are not fired prematurely on double-taps.
func (t *TouchState) PollPendingTap(tickCount int64) (isTap bool, x, y float64) {
	if t.pendingTapTick == 0 {
		return false, 0, 0
	}
	if tickCount-t.pendingTapTick < DoubleTapMaxInterval {
		return false, 0, 0
	}
	// Window expired — emit the single-tap now.
	x, y = t.pendingTapX, t.pendingTapY
	t.pendingTapTick = 0
	return true, x, y
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
	t.lastLongPressTouchID = 0
}
