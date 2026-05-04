// Package overlays - Masked Event Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: Masked Events appear as translucent domes
// on the Pulse Map. Inside the dome, all participants appear as identical
// featureless dots to maintain anonymity.
// Per ROADMAP.md line 506: "Pulse Map visualization — translucent dome with
// identical featureless dots inside".
//

package overlays

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// MaskedEventState represents the lifecycle state of a Masked Event.
type MaskedEventState uint8

const (
	MaskedEventPending MaskedEventState = iota // Waiting for start time.
	MaskedEventActive                          // Event in progress.
	MaskedEventEnded                           // Event has concluded.
)

// MaskedParticipant represents a participant inside a Masked Event.
// Per spec, participants are featureless - no identifying visual information.
type MaskedParticipant struct {
	// Participants have no identifying info in visualization.
	// Each is just an anonymous dot inside the dome.
	X, Y float64 // Position inside dome (relative to center).
}

// MaskedEventInfo contains information about a Masked Event for visualization.
type MaskedEventInfo struct {
	EventID          [32]byte            // Unique event identifier.
	Topic            string              // Event topic (shown on hover only).
	CenterX, CenterY float64             // Center position on Pulse Map.
	StartTime        time.Time           // Event start time.
	EndTime          time.Time           // Event end time.
	State            MaskedEventState    // Current event state.
	Participants     []MaskedParticipant // Anonymous participants.
}

// MaskedEventOverlay renders Masked Events on the Pulse Map.
type MaskedEventOverlay struct {
	mu sync.RWMutex

	events    map[string]*MaskedEventInfo
	domePhase map[string]float64 // Animation phase per dome.
	dotPhase  map[string]float64 // Dot movement phase per event.
}

// NewMaskedEventOverlay creates a new Masked Event overlay renderer.
func NewMaskedEventOverlay() *MaskedEventOverlay {
	return &MaskedEventOverlay{
		events:    make(map[string]*MaskedEventInfo),
		domePhase: make(map[string]float64),
		dotPhase:  make(map[string]float64),
	}
}

// AddEvent adds a Masked Event to the overlay.
func (mo *MaskedEventOverlay) AddEvent(info *MaskedEventInfo) {
	if info == nil {
		return
	}
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(info.EventID)
	mo.events[key] = info
	mo.domePhase[key] = 0
	mo.dotPhase[key] = 0
}

// RemoveEvent removes a Masked Event from the overlay.
func (mo *MaskedEventOverlay) RemoveEvent(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(eventID)
	delete(mo.events, key)
	delete(mo.domePhase, key)
	delete(mo.dotPhase, key)
}

// UpdateEvent updates event state.
func (mo *MaskedEventOverlay) UpdateEvent(info *MaskedEventInfo) {
	if info == nil {
		return
	}
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(info.EventID)
	mo.events[key] = info
}

// maskedEventKey converts an event ID to a map key.
func maskedEventKey(id [32]byte) string {
	return string(id[:])
}

// Update advances animations for all Masked Events.
func (mo *MaskedEventOverlay) Update() {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	for key := range mo.events {
		mo.updateDomePhase(key)
		mo.updateDotPhase(key)
	}
}

// updateDomePhase advances the dome shimmer animation.
func (mo *MaskedEventOverlay) updateDomePhase(key string) {
	phase := mo.domePhase[key]
	phase += 0.015 // Slower than Shadow Play for a more subtle effect.
	if phase > 2*math.Pi {
		phase -= 2 * math.Pi
	}
	mo.domePhase[key] = phase
}

// updateDotPhase advances the dot movement animation.
func (mo *MaskedEventOverlay) updateDotPhase(key string) {
	phase := mo.dotPhase[key]
	phase += 0.01 // Slow, subtle movement.
	if phase > 2*math.Pi {
		phase -= 2 * math.Pi
	}
	mo.dotPhase[key] = phase
}

// computeDomeRadius calculates dome radius based on participant count.
func (mo *MaskedEventOverlay) computeDomeRadius(info *MaskedEventInfo) float64 {
	base := 60.0
	perParticipant := 5.0
	return base + float64(len(info.Participants))*perParticipant
}

// Draw renders Masked Events to the screen.
func (mo *MaskedEventOverlay) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	for key, info := range mo.events {
		domePhase := mo.domePhase[key]
		dotPhase := mo.dotPhase[key]
		mo.drawEvent(screen, info, domePhase, dotPhase, camX, camY, zoom)
	}
}

// drawEvent renders a single Masked Event.
func (mo *MaskedEventOverlay) drawEvent(
	screen *ebiten.Image,
	info *MaskedEventInfo,
	domePhase, dotPhase float64,
	camX, camY, zoom float64,
) {
	// Transform world coordinates to screen coordinates.
	screenX := (info.CenterX - camX) * zoom
	screenY := (info.CenterY - camY) * zoom

	radius := mo.computeDomeRadius(info) * zoom

	// Draw the translucent dome.
	mo.drawTranslucentDome(screen, screenX, screenY, radius, domePhase, info.State)

	// Draw featureless dots for participants.
	mo.drawFeaturelessDots(screen, info, screenX, screenY, radius, dotPhase, zoom)

	// Draw time indicator.
	mo.drawTimeIndicator(screen, screenX, screenY, radius, info)
}

// drawTranslucentDome renders the translucent dome effect.
func (mo *MaskedEventOverlay) drawTranslucentDome(
	screen *ebiten.Image,
	cx, cy, radius, phase float64,
	state MaskedEventState,
) {
	// Draw multiple concentric rings with low opacity for translucent effect.
	numRings := 8
	for i := 0; i < numRings; i++ {
		ringProgress := float64(i) / float64(numRings)
		ringRadius := radius * (0.4 + 0.6*ringProgress)

		// Subtle shimmer effect.
		shimmer := 0.7 + 0.2*math.Sin(phase+ringProgress*math.Pi*1.5)
		baseAlpha := uint8(25 * (1.0 - ringProgress) * shimmer)

		col := mo.getDomeColor(state, baseAlpha)
		mo.drawDomeRing(screen, cx, cy, ringRadius, 1.5, col)
	}

	// Draw outer dome boundary.
	outlineColor := mo.getDomeOutlineColor(state, phase)
	mo.drawDomeRing(screen, cx, cy, radius, 2.0, outlineColor)

	// Fill dome interior with very low alpha.
	fillColor := mo.getDomeFillColor(state)
	mo.drawFilledDome(screen, cx, cy, radius, fillColor)
}

// getDomeColor returns the dome ring color based on state.
func (mo *MaskedEventOverlay) getDomeColor(state MaskedEventState, alpha uint8) color.RGBA {
	switch state {
	case MaskedEventPending:
		return color.RGBA{100, 100, 140, alpha} // Gray-blue for pending.
	case MaskedEventActive:
		return color.RGBA{80, 140, 180, alpha} // Cyan-blue for active.
	case MaskedEventEnded:
		return color.RGBA{80, 80, 100, alpha} // Faded gray for ended.
	default:
		return color.RGBA{100, 120, 140, alpha}
	}
}

// getDomeOutlineColor returns the dome outline color with pulse effect.
func (mo *MaskedEventOverlay) getDomeOutlineColor(state MaskedEventState, phase float64) color.RGBA {
	alpha := uint8(120 + 30*math.Sin(phase*2))

	switch state {
	case MaskedEventPending:
		return color.RGBA{140, 140, 180, alpha}
	case MaskedEventActive:
		return color.RGBA{100, 180, 220, alpha} // Bright cyan.
	case MaskedEventEnded:
		return color.RGBA{100, 100, 120, alpha}
	default:
		return color.RGBA{120, 140, 160, alpha}
	}
}

// getDomeFillColor returns the dome interior fill color (very low alpha).
func (mo *MaskedEventOverlay) getDomeFillColor(state MaskedEventState) color.RGBA {
	switch state {
	case MaskedEventPending:
		return color.RGBA{80, 80, 120, 15}
	case MaskedEventActive:
		return color.RGBA{60, 120, 160, 20}
	case MaskedEventEnded:
		return color.RGBA{60, 60, 80, 10}
	default:
		return color.RGBA{80, 100, 120, 15}
	}
}

// drawDomeRing renders a circular ring for the dome.
func (mo *MaskedEventOverlay) drawDomeRing(
	screen *ebiten.Image,
	cx, cy, radius, strokeWidth float64,
	col color.RGBA,
) {
	if col.A == 0 {
		return
	}

	numSegments := int(math.Max(32, radius/2))
	angleStep := 2 * math.Pi / float64(numSegments)

	for i := 0; i < numSegments; i++ {
		angle1 := float64(i) * angleStep
		angle2 := float64(i+1) * angleStep

		x1 := float32(cx + radius*math.Cos(angle1))
		y1 := float32(cy + radius*math.Sin(angle1))
		x2 := float32(cx + radius*math.Cos(angle2))
		y2 := float32(cy + radius*math.Sin(angle2))

		vector.StrokeLine(screen, x1, y1, x2, y2, float32(strokeWidth), col, false)
	}
}

// drawFilledDome renders a filled circle for the dome interior.
func (mo *MaskedEventOverlay) drawFilledDome(
	screen *ebiten.Image,
	cx, cy, radius float64,
	col color.RGBA,
) {
	if col.A == 0 {
		return
	}
	vector.DrawFilledCircle(screen, float32(cx), float32(cy), float32(radius), col, false)
}

// drawFeaturelessDots renders identical anonymous dots for all participants.
// Per spec: "identical featureless dots inside" to maintain anonymity.
func (mo *MaskedEventOverlay) drawFeaturelessDots(
	screen *ebiten.Image,
	info *MaskedEventInfo,
	screenCX, screenCY, screenRadius float64,
	dotPhase float64,
	zoom float64,
) {
	if len(info.Participants) == 0 {
		return
	}

	// All dots are identical - same size, same color.
	dotRadius := float32(4 * zoom)
	if dotRadius < 2 {
		dotRadius = 2
	}
	if dotRadius > 8 {
		dotRadius = 8
	}

	// Dot color depends on event state but is the same for all participants.
	dotColor := mo.getDotColor(info.State)

	// Distribute dots evenly inside the dome with subtle movement.
	usableRadius := screenRadius * 0.75 // Keep dots inside inner dome area.

	for i, participant := range info.Participants {
		// Calculate position with subtle floating animation.
		// Each dot gets a slightly different phase based on index.
		particlePhase := dotPhase + float64(i)*0.3

		// Participant positions are relative offsets from center.
		baseX := participant.X * zoom
		baseY := participant.Y * zoom

		// Add subtle floating movement.
		floatX := math.Sin(particlePhase) * 3 * zoom
		floatY := math.Cos(particlePhase*1.3) * 2 * zoom

		dotX := screenCX + baseX + floatX
		dotY := screenCY + baseY + floatY

		// Clamp to dome radius.
		dx := dotX - screenCX
		dy := dotY - screenCY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > usableRadius {
			scale := usableRadius / dist
			dotX = screenCX + dx*scale
			dotY = screenCY + dy*scale
		}

		// Draw the featureless dot.
		vector.DrawFilledCircle(screen, float32(dotX), float32(dotY), dotRadius, dotColor, false)
	}
}

// getDotColor returns the color for participant dots.
// All dots are identical to preserve anonymity.
func (mo *MaskedEventOverlay) getDotColor(state MaskedEventState) color.RGBA {
	switch state {
	case MaskedEventPending:
		return color.RGBA{160, 160, 180, 180}
	case MaskedEventActive:
		return color.RGBA{180, 220, 240, 200} // Bright dots for active.
	case MaskedEventEnded:
		return color.RGBA{120, 120, 140, 150}
	default:
		return color.RGBA{160, 180, 200, 180}
	}
}

// drawTimeIndicator shows the event time status.
func (mo *MaskedEventOverlay) drawTimeIndicator(
	screen *ebiten.Image,
	cx, cy, radius float64,
	info *MaskedEventInfo,
) {
	// Draw at top of dome.
	indicatorY := cy - radius - 15

	// Calculate progress arc for active events.
	if info.State == MaskedEventActive {
		mo.drawProgressArc(screen, cx, indicatorY, info)
	} else if info.State == MaskedEventPending {
		mo.drawCountdownIndicator(screen, cx, indicatorY)
	}

	// Draw participant count indicator.
	mo.drawParticipantCount(screen, cx, cy+radius+15, len(info.Participants))
}

// drawProgressArc renders a circular progress indicator.
func (mo *MaskedEventOverlay) drawProgressArc(
	screen *ebiten.Image,
	cx, cy float64,
	info *MaskedEventInfo,
) {
	now := time.Now()
	total := info.EndTime.Sub(info.StartTime)
	elapsed := now.Sub(info.StartTime)

	progress := float64(elapsed) / float64(total)
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	arcRadius := float32(10)
	strokeWidth := float32(3)

	// Background arc (full circle, dim).
	bgColor := color.RGBA{60, 60, 80, 100}
	mo.drawArc(screen, float32(cx), float32(cy), arcRadius, 0, 2*math.Pi, strokeWidth, bgColor)

	// Progress arc.
	progressColor := color.RGBA{100, 200, 180, 200}
	endAngle := 2 * math.Pi * progress
	mo.drawArc(screen, float32(cx), float32(cy), arcRadius, -math.Pi/2, -math.Pi/2+endAngle, strokeWidth, progressColor)
}

// drawArc draws an arc segment.
func (mo *MaskedEventOverlay) drawArc(
	screen *ebiten.Image,
	cx, cy, radius float32,
	startAngle, endAngle float64,
	strokeWidth float32,
	col color.RGBA,
) {
	numSegments := 32
	angleRange := endAngle - startAngle
	angleStep := angleRange / float64(numSegments)

	for i := 0; i < numSegments; i++ {
		a1 := startAngle + float64(i)*angleStep
		a2 := startAngle + float64(i+1)*angleStep

		x1 := cx + radius*float32(math.Cos(a1))
		y1 := cy + radius*float32(math.Sin(a1))
		x2 := cx + radius*float32(math.Cos(a2))
		y2 := cy + radius*float32(math.Sin(a2))

		vector.StrokeLine(screen, x1, y1, x2, y2, strokeWidth, col, false)
	}
}

// drawCountdownIndicator shows that event is pending.
func (mo *MaskedEventOverlay) drawCountdownIndicator(screen *ebiten.Image, cx, cy float64) {
	// Draw a pulsing hourglass icon.
	t := float64(time.Now().UnixMilli()) / 1000.0
	pulse := float32(0.7 + 0.3*math.Sin(t*2))
	col := color.RGBA{140, 140, 160, uint8(180 * pulse)}

	size := float32(6)

	// Hourglass shape.
	vector.StrokeLine(screen, float32(cx)-size, float32(cy)-size, float32(cx)+size, float32(cy)-size, 2, col, false)
	vector.StrokeLine(screen, float32(cx)-size, float32(cy)-size, float32(cx), float32(cy), 2, col, false)
	vector.StrokeLine(screen, float32(cx)+size, float32(cy)-size, float32(cx), float32(cy), 2, col, false)
	vector.StrokeLine(screen, float32(cx)-size, float32(cy)+size, float32(cx)+size, float32(cy)+size, 2, col, false)
	vector.StrokeLine(screen, float32(cx)-size, float32(cy)+size, float32(cx), float32(cy), 2, col, false)
	vector.StrokeLine(screen, float32(cx)+size, float32(cy)+size, float32(cx), float32(cy), 2, col, false)
}

// drawParticipantCount shows the number of participants.
func (mo *MaskedEventOverlay) drawParticipantCount(screen *ebiten.Image, cx, cy float64, count int) {
	// Draw dots to represent count (max 10, then a + indicator).
	dotRadius := float32(2)
	spacing := float32(6)

	displayCount := count
	if displayCount > 10 {
		displayCount = 10
	}

	totalWidth := float32(displayCount-1) * spacing
	startX := float32(cx) - totalWidth/2

	col := color.RGBA{140, 160, 180, 180}

	for i := 0; i < displayCount; i++ {
		dotX := startX + float32(i)*spacing
		vector.DrawFilledCircle(screen, dotX, float32(cy), dotRadius, col, false)
	}

	// If more than 10, draw a "+" indicator.
	if count > 10 {
		plusX := startX + float32(displayCount)*spacing + 4
		plusCol := color.RGBA{180, 180, 200, 180}
		vector.StrokeLine(screen, plusX-3, float32(cy), plusX+3, float32(cy), 2, plusCol, false)
		vector.StrokeLine(screen, plusX, float32(cy)-3, plusX, float32(cy)+3, 2, plusCol, false)
	}
}

// GetEvent retrieves event info by ID.
func (mo *MaskedEventOverlay) GetEvent(eventID [32]byte) *MaskedEventInfo {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	return mo.events[maskedEventKey(eventID)]
}

// EventCount returns the number of events being rendered.
func (mo *MaskedEventOverlay) EventCount() int {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	return len(mo.events)
}

// Clear removes all events from the overlay.
func (mo *MaskedEventOverlay) Clear() {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	mo.events = make(map[string]*MaskedEventInfo)
	mo.domePhase = make(map[string]float64)
	mo.dotPhase = make(map[string]float64)
}

// SetParticipantPositions distributes participants evenly in a circular pattern.
// This can be called to initialize positions when participants are added.
func (mo *MaskedEventOverlay) SetParticipantPositions(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(eventID)
	info, ok := mo.events[key]
	if !ok || len(info.Participants) == 0 {
		return
	}

	radius := mo.computeDomeRadius(info) * 0.5 // Spread within inner dome.
	count := len(info.Participants)

	for i := range info.Participants {
		angle := 2 * math.Pi * float64(i) / float64(count)
		// Add some randomness to prevent perfect circle appearance.
		radiusJitter := radius * (0.3 + 0.7*float64((i*7+3)%10)/10)
		info.Participants[i].X = radiusJitter * math.Cos(angle)
		info.Participants[i].Y = radiusJitter * math.Sin(angle)
	}
}

// AddParticipant adds a participant to an event with auto-positioning.
func (mo *MaskedEventOverlay) AddParticipant(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(eventID)
	info, ok := mo.events[key]
	if !ok {
		return
	}

	// Add new participant with random position inside dome.
	radius := mo.computeDomeRadius(info) * 0.5
	angle := float64(len(info.Participants)) * 2.39996 // Golden angle for even distribution.

	info.Participants = append(info.Participants, MaskedParticipant{
		X: radius * 0.6 * math.Cos(angle),
		Y: radius * 0.6 * math.Sin(angle),
	})
}

// RemoveParticipant removes a participant (maintains anonymity by not specifying which).
func (mo *MaskedEventOverlay) RemoveParticipant(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(eventID)
	info, ok := mo.events[key]
	if !ok || len(info.Participants) == 0 {
		return
	}

	// Remove the last participant (they're all identical anyway).
	info.Participants = info.Participants[:len(info.Participants)-1]
}

// SetState updates the event state.
func (mo *MaskedEventOverlay) SetState(eventID [32]byte, state MaskedEventState) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := maskedEventKey(eventID)
	if info, ok := mo.events[key]; ok {
		info.State = state
	}
}
