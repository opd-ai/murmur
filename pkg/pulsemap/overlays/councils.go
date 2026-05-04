// Package overlays — Phantom Council Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md and ROADMAP.md line 549:
// "Pulse Map visualization — constellation pattern connecting member nodes, unique color threads".
// Councils appear as constellations with connecting threads between members, unique council colors,
// and glowing effects during active communication.
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

// CouncilMemberInfo contains information about a council member for visualization.
type CouncilMemberInfo struct {
	SpecterKey      [32]byte // Member's Specter identity.
	X, Y            float64  // Position on Pulse Map.
	IsCommunicating bool     // True if member is currently active in council chat.
}

// CouncilInfo contains information about a Phantom Council for visualization.
type CouncilInfo struct {
	ID             [32]byte // Unique council identifier.
	Name           string   // Council display name.
	Members        []CouncilMemberInfo
	Color          color.RGBA // Unique council thread color.
	IsActive       bool       // True if council is in active session.
	AnimationPhase float64    // Current animation phase.
}

// CouncilOverlay renders Phantom Council visualizations on the Pulse Map.
// Per ROADMAP.md line 549: "constellation pattern connecting member nodes, unique color threads".
type CouncilOverlay struct {
	mu sync.RWMutex

	councils   map[string]*CouncilInfo
	starPhase  map[string]float64 // Per-member star twinkle phase.
	pulsePhase map[string]float64 // Per-council pulse phase for active communication.

	Visible bool
	Opacity float32
}

// NewCouncilOverlay creates a new council overlay renderer.
func NewCouncilOverlay() *CouncilOverlay {
	return &CouncilOverlay{
		councils:   make(map[string]*CouncilInfo),
		starPhase:  make(map[string]float64),
		pulsePhase: make(map[string]float64),
		Visible:    true,
		Opacity:    0.8,
	}
}

// AddCouncil adds a council to the overlay.
func (co *CouncilOverlay) AddCouncil(info *CouncilInfo) {
	if info == nil {
		return
	}
	co.mu.Lock()
	defer co.mu.Unlock()

	key := councilKey(info.ID)
	co.councils[key] = info
	co.pulsePhase[key] = 0

	// Initialize star phases for members.
	for i := range info.Members {
		memberKey := memberStarKey(info.ID, info.Members[i].SpecterKey)
		co.starPhase[memberKey] = float64(i) * 0.5 // Stagger initial phases.
	}
}

// RemoveCouncil removes a council from the overlay.
func (co *CouncilOverlay) RemoveCouncil(councilID [32]byte) {
	co.mu.Lock()
	defer co.mu.Unlock()

	key := councilKey(councilID)
	info := co.councils[key]
	if info != nil {
		// Clean up star phases.
		for _, member := range info.Members {
			delete(co.starPhase, memberStarKey(councilID, member.SpecterKey))
		}
	}
	delete(co.councils, key)
	delete(co.pulsePhase, key)
}

// UpdateCouncil updates council state.
func (co *CouncilOverlay) UpdateCouncil(info *CouncilInfo) {
	if info == nil {
		return
	}
	co.mu.Lock()
	defer co.mu.Unlock()

	key := councilKey(info.ID)
	oldInfo := co.councils[key]
	co.councils[key] = info

	// Initialize star phases for new members.
	for i := range info.Members {
		memberKey := memberStarKey(info.ID, info.Members[i].SpecterKey)
		if _, exists := co.starPhase[memberKey]; !exists {
			co.starPhase[memberKey] = float64(i) * 0.5
		}
	}

	// Clean up star phases for removed members.
	if oldInfo != nil {
		newMembers := make(map[string]bool)
		for _, m := range info.Members {
			newMembers[memberStarKey(info.ID, m.SpecterKey)] = true
		}
		for _, m := range oldInfo.Members {
			mKey := memberStarKey(info.ID, m.SpecterKey)
			if !newMembers[mKey] {
				delete(co.starPhase, mKey)
			}
		}
	}
}

// councilKey converts a council ID to a map key.
func councilKey(id [32]byte) string {
	return string(id[:])
}

// memberStarKey creates a unique key for a member's star phase.
func memberStarKey(councilID, specterKey [32]byte) string {
	var combined [64]byte
	copy(combined[:32], councilID[:])
	copy(combined[32:], specterKey[:])
	return string(combined[:])
}

// Update advances animations for all councils.
func (co *CouncilOverlay) Update(dt float64) {
	co.mu.Lock()
	defer co.mu.Unlock()

	for key, info := range co.councils {
		// Advance council animation phase.
		info.AnimationPhase += dt * 0.5
		if info.AnimationPhase > 2*math.Pi {
			info.AnimationPhase -= 2 * math.Pi
		}

		// Advance pulse phase for active councils.
		if info.IsActive {
			phase := co.pulsePhase[key]
			phase += dt * 2.0 // Faster pulse when active.
			if phase > 2*math.Pi {
				phase -= 2 * math.Pi
			}
			co.pulsePhase[key] = phase
		}

		// Advance star twinkle phases.
		for _, member := range info.Members {
			memberKey := memberStarKey(info.ID, member.SpecterKey)
			phase := co.starPhase[memberKey]
			phase += dt * (1.0 + 0.3*math.Sin(float64(member.SpecterKey[0]))) // Slightly different rates.
			if phase > 2*math.Pi {
				phase -= 2 * math.Pi
			}
			co.starPhase[memberKey] = phase
		}
	}
}

// Draw renders all councils to the screen.
func (co *CouncilOverlay) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if !co.Visible {
		return
	}

	co.mu.RLock()
	defer co.mu.RUnlock()

	screenW := float64(screen.Bounds().Dx())
	screenH := float64(screen.Bounds().Dy())

	for key, info := range co.councils {
		if len(info.Members) < 2 {
			continue // Need at least 2 members for constellation.
		}

		pulsePhase := co.pulsePhase[key]
		co.drawCouncil(screen, info, pulsePhase, camX, camY, zoom, screenW, screenH)
	}
}

// drawCouncil renders a single council constellation.
func (co *CouncilOverlay) drawCouncil(
	screen *ebiten.Image,
	info *CouncilInfo,
	pulsePhase float64,
	camX, camY, zoom, screenW, screenH float64,
) {
	// Calculate base opacity with active communication glow.
	baseOpacity := float32(co.Opacity)
	if info.IsActive {
		baseOpacity *= float32(0.8 + 0.2*math.Sin(pulsePhase*2))
	}

	// Draw constellation threads connecting all members.
	co.drawConstellationThreads(screen, info, camX, camY, zoom, screenW, screenH, baseOpacity)

	// Draw member star nodes.
	co.drawMemberStars(screen, info, camX, camY, zoom, screenW, screenH, baseOpacity)

	// Draw council name label at centroid (when zoomed in enough).
	if zoom > 0.5 {
		co.drawCouncilLabel(screen, info, camX, camY, zoom, screenW, screenH, baseOpacity)
	}
}

// drawConstellationThreads draws the connecting threads between council members.
func (co *CouncilOverlay) drawConstellationThreads(
	screen *ebiten.Image,
	info *CouncilInfo,
	camX, camY, zoom, screenW, screenH float64,
	opacity float32,
) {
	members := info.Members
	threadColor := applyOpacity(info.Color, uint8(float32(180)*opacity))

	// Draw threads in a constellation pattern (connect each member to neighbors).
	// For small councils (<=5), fully connect. For larger, use nearest-neighbor pattern.
	if len(members) <= 5 {
		// Full mesh for small councils.
		for i := 0; i < len(members); i++ {
			for j := i + 1; j < len(members); j++ {
				co.drawThread(screen, &members[i], &members[j], info, threadColor, camX, camY, zoom, screenW, screenH)
			}
		}
	} else {
		// Ring pattern for larger councils: connect each to next + opposite.
		for i := 0; i < len(members); i++ {
			next := (i + 1) % len(members)
			opposite := (i + len(members)/2) % len(members)

			co.drawThread(screen, &members[i], &members[next], info, threadColor, camX, camY, zoom, screenW, screenH)
			co.drawThread(screen, &members[i], &members[opposite], info, threadColor, camX, camY, zoom, screenW, screenH)
		}
	}
}

// drawThread draws a single constellation thread between two members.
func (co *CouncilOverlay) drawThread(
	screen *ebiten.Image,
	m1, m2 *CouncilMemberInfo,
	info *CouncilInfo,
	threadColor color.RGBA,
	camX, camY, zoom, screenW, screenH float64,
) {
	// Transform to screen coordinates.
	x1 := float32((m1.X-camX)*zoom + screenW/2)
	y1 := float32((m1.Y-camY)*zoom + screenH/2)
	x2 := float32((m2.X-camX)*zoom + screenW/2)
	y2 := float32((m2.Y-camY)*zoom + screenH/2)

	// Base line width scales with zoom.
	lineWidth := float32(1.5 * zoom)
	if lineWidth < 1 {
		lineWidth = 1
	}
	if lineWidth > 3 {
		lineWidth = 3
	}

	// If both members are communicating, draw a brighter, pulsing thread.
	if m1.IsCommunicating && m2.IsCommunicating {
		pulse := float32(0.7 + 0.3*math.Sin(info.AnimationPhase*4))
		activeColor := threadColor
		activeColor.A = uint8(float32(activeColor.A) * pulse)
		activeColor.R = uint8(math.Min(255, float64(activeColor.R)+30))
		activeColor.G = uint8(math.Min(255, float64(activeColor.G)+30))
		activeColor.B = uint8(math.Min(255, float64(activeColor.B)+30))
		vector.StrokeLine(screen, x1, y1, x2, y2, lineWidth*1.5, activeColor, true)
	} else {
		vector.StrokeLine(screen, x1, y1, x2, y2, lineWidth, threadColor, true)
	}
}

// drawMemberStars draws star icons at each member's position.
func (co *CouncilOverlay) drawMemberStars(
	screen *ebiten.Image,
	info *CouncilInfo,
	camX, camY, zoom, screenW, screenH float64,
	opacity float32,
) {
	for _, member := range info.Members {
		memberKey := memberStarKey(info.ID, member.SpecterKey)
		starPhase := co.starPhase[memberKey]

		co.drawMemberStar(screen, &member, info, starPhase, camX, camY, zoom, screenW, screenH, opacity)
	}
}

// drawMemberStar draws a single member star with twinkle effect.
func (co *CouncilOverlay) drawMemberStar(
	screen *ebiten.Image,
	member *CouncilMemberInfo,
	info *CouncilInfo,
	starPhase float64,
	camX, camY, zoom, screenW, screenH float64,
	opacity float32,
) {
	// Transform to screen coordinates.
	sx := float32((member.X-camX)*zoom + screenW/2)
	sy := float32((member.Y-camY)*zoom + screenH/2)

	// Star size based on zoom and communication state.
	baseSize := float32(6 * zoom)
	if baseSize < 3 {
		baseSize = 3
	}
	if baseSize > 12 {
		baseSize = 12
	}

	// Twinkle effect.
	twinkle := float32(0.7 + 0.3*math.Sin(starPhase*3))

	// Star color based on council color.
	starColor := info.Color
	starColor.A = uint8(float32(220) * opacity * twinkle)

	// Draw outer glow.
	glowColor := starColor
	glowColor.A = uint8(float32(glowColor.A) * 0.3)
	glowSize := baseSize * 2
	if member.IsCommunicating {
		// Larger glow when communicating.
		glowSize = baseSize * 3
		glowColor.A = uint8(float32(glowColor.A) * 1.5)
	}
	vector.DrawFilledCircle(screen, sx, sy, glowSize, glowColor, true)

	// Draw star shape (4-pointed).
	co.drawStarShape(screen, sx, sy, baseSize, starColor)

	// Draw inner bright core.
	coreColor := color.RGBA{255, 255, 255, uint8(float32(200) * opacity * twinkle)}
	vector.DrawFilledCircle(screen, sx, sy, baseSize*0.3, coreColor, true)
}

// drawStarShape draws a 4-pointed star.
func (co *CouncilOverlay) drawStarShape(screen *ebiten.Image, cx, cy, size float32, col color.RGBA) {
	// 4-pointed star: draw two overlapping diamonds.
	// Vertical diamond.
	points := []float32{
		cx, cy - size, // Top.
		cx + size*0.3, cy, // Right.
		cx, cy + size, // Bottom.
		cx - size*0.3, cy, // Left.
	}
	drawPolygon(screen, points, col)

	// Horizontal diamond.
	points = []float32{
		cx - size, cy, // Left.
		cx, cy + size*0.3, // Bottom.
		cx + size, cy, // Right.
		cx, cy - size*0.3, // Top.
	}
	drawPolygon(screen, points, col)
}

// drawPolygon draws a filled polygon from a slice of x,y coordinate pairs.
func drawPolygon(screen *ebiten.Image, points []float32, col color.RGBA) {
	if len(points) < 6 { // Need at least 3 points (6 values).
		return
	}

	// Draw using triangles from center.
	n := len(points) / 2
	var cx, cy float32
	for i := 0; i < n; i++ {
		cx += points[i*2]
		cy += points[i*2+1]
	}
	cx /= float32(n)
	cy /= float32(n)

	// Draw triangles fan-style.
	for i := 0; i < n; i++ {
		x1 := points[i*2]
		y1 := points[i*2+1]
		next := (i + 1) % n
		x2 := points[next*2]
		y2 := points[next*2+1]

		// Draw triangle edges.
		vector.StrokeLine(screen, x1, y1, x2, y2, 1, col, true)
	}

	// Fill with smaller circle at center for visual effect.
	vector.DrawFilledCircle(screen, cx, cy, 2, col, true)
}

// drawCouncilLabel draws the council name near the centroid.
func (co *CouncilOverlay) drawCouncilLabel(
	screen *ebiten.Image,
	info *CouncilInfo,
	camX, camY, zoom, screenW, screenH float64,
	opacity float32,
) {
	if len(info.Members) == 0 || info.Name == "" {
		return
	}

	// Calculate centroid.
	var cx, cy float64
	for _, m := range info.Members {
		cx += m.X
		cy += m.Y
	}
	cx /= float64(len(info.Members))
	cy /= float64(len(info.Members))

	// Transform to screen coordinates.
	sx := float32((cx-camX)*zoom + screenW/2)
	sy := float32((cy-camY)*zoom + screenH/2)

	// Draw background for label.
	labelWidth := float32(len(info.Name) * 6)
	labelHeight := float32(12)
	bgColor := color.RGBA{20, 20, 30, uint8(180 * opacity)}
	vector.DrawFilledRect(screen, sx-labelWidth/2-2, sy-labelHeight/2-2, labelWidth+4, labelHeight+4, bgColor, true)

	// Draw border.
	borderColor := applyOpacity(info.Color, uint8(float32(200)*opacity))
	vector.StrokeRect(screen, sx-labelWidth/2-2, sy-labelHeight/2-2, labelWidth+4, labelHeight+4, 1, borderColor, true)
}

// applyOpacity creates a new color with modified alpha.
func applyOpacity(c color.RGBA, alpha uint8) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, alpha}
}

// GenerateCouncilColor creates a unique color for a council based on its ID.
// Per ROADMAP.md: "unique color threads" for each council.
func GenerateCouncilColor(councilID [32]byte) color.RGBA {
	// Use council ID bytes to generate a hue in the 200-280° range (cool tones for anonymous).
	hue := 200 + float64(councilID[0]%80)

	// Saturation and value for visibility.
	saturation := 0.6 + 0.2*float64(councilID[1]%100)/100.0
	value := 0.7 + 0.2*float64(councilID[2]%100)/100.0

	// Convert HSV to RGB.
	r, g, b := hsvToRGB(hue, saturation, value)

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// hsvToRGB converts HSV color values to RGB.
func hsvToRGB(h, s, v float64) (r, g, b float64) {
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return r + m, g + m, b + m
}

// GetCouncil retrieves council info by ID.
func (co *CouncilOverlay) GetCouncil(councilID [32]byte) *CouncilInfo {
	co.mu.RLock()
	defer co.mu.RUnlock()
	return co.councils[councilKey(councilID)]
}

// CouncilCount returns the number of councils in the overlay.
func (co *CouncilOverlay) CouncilCount() int {
	co.mu.RLock()
	defer co.mu.RUnlock()
	return len(co.councils)
}

// Clear removes all councils from the overlay.
func (co *CouncilOverlay) Clear() {
	co.mu.Lock()
	defer co.mu.Unlock()

	co.councils = make(map[string]*CouncilInfo)
	co.starPhase = make(map[string]float64)
	co.pulsePhase = make(map[string]float64)
}

// SetVisible sets the overlay visibility.
func (co *CouncilOverlay) SetVisible(visible bool) {
	co.mu.Lock()
	defer co.mu.Unlock()
	co.Visible = visible
}

// SetOpacity sets the global opacity.
func (co *CouncilOverlay) SetOpacity(opacity float32) {
	co.mu.Lock()
	defer co.mu.Unlock()
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	co.Opacity = opacity
}

// SetCouncilActive sets whether a council is in active communication.
// Per ROADMAP.md line 645: "glow during active communication".
func (co *CouncilOverlay) SetCouncilActive(councilID [32]byte, active bool) {
	co.mu.Lock()
	defer co.mu.Unlock()

	key := councilKey(councilID)
	if info, exists := co.councils[key]; exists {
		info.IsActive = active
	}
}

// SetMemberCommunicating marks a member as currently communicating.
func (co *CouncilOverlay) SetMemberCommunicating(councilID, specterKey [32]byte, communicating bool) {
	co.mu.Lock()
	defer co.mu.Unlock()

	key := councilKey(councilID)
	if info, exists := co.councils[key]; exists {
		for i := range info.Members {
			if info.Members[i].SpecterKey == specterKey {
				info.Members[i].IsCommunicating = communicating
				return
			}
		}
	}
}

// UpdateMemberPosition updates a member's position on the Pulse Map.
func (co *CouncilOverlay) UpdateMemberPosition(councilID, specterKey [32]byte, x, y float64) {
	co.mu.Lock()
	defer co.mu.Unlock()

	key := councilKey(councilID)
	if info, exists := co.councils[key]; exists {
		for i := range info.Members {
			if info.Members[i].SpecterKey == specterKey {
				info.Members[i].X = x
				info.Members[i].Y = y
				return
			}
		}
	}
}

// NewCouncilInfo creates a new CouncilInfo with a generated color.
func NewCouncilInfo(id [32]byte, name string) *CouncilInfo {
	return &CouncilInfo{
		ID:      id,
		Name:    name,
		Members: make([]CouncilMemberInfo, 0),
		Color:   GenerateCouncilColor(id),
	}
}

// AddMember adds a member to the council info.
func (ci *CouncilInfo) AddMember(specterKey [32]byte, x, y float64) {
	ci.Members = append(ci.Members, CouncilMemberInfo{
		SpecterKey: specterKey,
		X:          x,
		Y:          y,
	})
}

// lastUpdateTime is used for calculating dt when Update is called without a dt parameter.
var lastUpdateTime = time.Now()

// Tick advances animations by one frame (convenience method).
func (co *CouncilOverlay) Tick() {
	now := time.Now()
	dt := now.Sub(lastUpdateTime).Seconds()
	lastUpdateTime = now
	if dt > 0.1 { // Cap dt to prevent large jumps.
		dt = 0.1
	}
	co.Update(dt)
}
