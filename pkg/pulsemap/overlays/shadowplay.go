// Package overlays - Shadow Play Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: "Shadow Play games appear on the Pulse Map
// as a dark shimmering dome centered on the initiator's node. Participant nodes
// inside the dome pulse with role-specific auras".
// Per ROADMAP.md line 491: "Pulse Map visualization — dark shimmering dome
// with lightning effects".
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ShadowPlayState represents the current state of a Shadow Play game.
type ShadowPlayState uint8

const (
	ShadowPlayWaiting   ShadowPlayState = iota // Waiting for players.
	ShadowPlayActive                           // Game in progress.
	ShadowPlayVoting                           // Voting phase.
	ShadowPlayEchoesWin                        // Echoes won.
	ShadowPlayShadesWin                        // Shades won.
	ShadowPlayExpired                          // Game expired.
)

// ShadowPlayerRole represents a player's role (only revealed to viewer).
type ShadowPlayerRole uint8

const (
	ShadowRoleUnknown ShadowPlayerRole = iota // Role not known to viewer.
	ShadowRoleEcho                            // Standard participant.
	ShadowRoleShade                           // Hidden disruptor.
)

// ShadowPlayer represents a participant in a Shadow Play game.
type ShadowPlayer struct {
	SpecterKey   [32]byte         // Specter identity.
	Role         ShadowPlayerRole // Role (may be unknown to viewer).
	IsEliminated bool             // True if eliminated.
	X, Y         float64          // Position on Pulse Map.
}

// ShadowPlayInfo contains information about a Shadow Play game.
type ShadowPlayInfo struct {
	GameID      [32]byte        // Unique game identifier.
	State       ShadowPlayState // Current game state.
	X, Y        float64         // Center position (initiator's node).
	StartTime   time.Time       // When the game started.
	EndTime     time.Time       // When the game ends.
	RoundNumber int             // Current voting round.
	Players     []ShadowPlayer  // Participants.
}

// Lightning represents an animated lightning bolt effect.
type Lightning struct {
	startAngle float64     // Starting angle on dome.
	endAngle   float64     // Ending angle on dome.
	segments   []lightning // Lightning path segments.
	intensity  float64     // Current intensity (0-1).
	startTime  time.Time   // When lightning started.
	duration   time.Duration
}

// lightning segment represents a single segment of a lightning bolt.
type lightning struct {
	x1, y1 float64
	x2, y2 float64
}

// ShadowPlayOverlay renders Shadow Play games on the Pulse Map.
type ShadowPlayOverlay struct {
	mu sync.RWMutex

	games      map[string]*ShadowPlayInfo
	lightnings map[string][]*Lightning // Lightning effects per game.
	domePhase  map[string]float64      // Animation phase per dome.

	rng *rand.Rand
}

// NewShadowPlayOverlay creates a new Shadow Play overlay renderer.
func NewShadowPlayOverlay() *ShadowPlayOverlay {
	return &ShadowPlayOverlay{
		games:      make(map[string]*ShadowPlayInfo),
		lightnings: make(map[string][]*Lightning),
		domePhase:  make(map[string]float64),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// AddGame adds a Shadow Play game to the overlay.
func (so *ShadowPlayOverlay) AddGame(info *ShadowPlayInfo) {
	if info == nil {
		return
	}
	so.mu.Lock()
	defer so.mu.Unlock()

	key := gameKey(info.GameID)
	so.games[key] = info
	so.lightnings[key] = make([]*Lightning, 0)
	so.domePhase[key] = 0
}

// RemoveGame removes a Shadow Play game from the overlay.
func (so *ShadowPlayOverlay) RemoveGame(gameID [32]byte) {
	so.mu.Lock()
	defer so.mu.Unlock()

	key := gameKey(gameID)
	delete(so.games, key)
	delete(so.lightnings, key)
	delete(so.domePhase, key)
}

// UpdateGame updates game state.
func (so *ShadowPlayOverlay) UpdateGame(info *ShadowPlayInfo) {
	if info == nil {
		return
	}
	so.mu.Lock()
	defer so.mu.Unlock()

	key := gameKey(info.GameID)
	so.games[key] = info
}

// gameKey converts a game ID to a map key.
func gameKey(id [32]byte) string {
	return string(id[:])
}

// Update advances animations for all Shadow Play games.
func (so *ShadowPlayOverlay) Update() {
	so.mu.Lock()
	defer so.mu.Unlock()

	now := time.Now()

	for key, info := range so.games {
		so.updateDomePhase(key)
		so.updateLightning(key, info, now)
		so.maybeSpawnLightning(key, info, now)
	}
}

// updateDomePhase advances the dome shimmer animation.
func (so *ShadowPlayOverlay) updateDomePhase(key string) {
	phase := so.domePhase[key]
	phase += 0.02
	if phase > 2*math.Pi {
		phase -= 2 * math.Pi
	}
	so.domePhase[key] = phase
}

// updateLightning updates existing lightning effects.
func (so *ShadowPlayOverlay) updateLightning(key string, info *ShadowPlayInfo, now time.Time) {
	bolts := so.lightnings[key]
	active := bolts[:0]

	for _, bolt := range bolts {
		elapsed := now.Sub(bolt.startTime)
		if elapsed < bolt.duration {
			// Fade intensity over time.
			progress := float64(elapsed) / float64(bolt.duration)
			bolt.intensity = 1.0 - progress
			active = append(active, bolt)
		}
	}

	so.lightnings[key] = active
}

// maybeSpawnLightning randomly spawns new lightning effects.
func (so *ShadowPlayOverlay) maybeSpawnLightning(key string, info *ShadowPlayInfo, now time.Time) {
	// Lightning is more frequent during voting phase.
	spawnChance := 0.01
	if info.State == ShadowPlayVoting {
		spawnChance = 0.03
	}

	if so.rng.Float64() < spawnChance {
		bolt := so.createLightning(info)
		so.lightnings[key] = append(so.lightnings[key], bolt)
	}
}

// createLightning generates a new lightning bolt effect.
func (so *ShadowPlayOverlay) createLightning(info *ShadowPlayInfo) *Lightning {
	radius := so.computeDomeRadius(info)

	startAngle := so.rng.Float64() * 2 * math.Pi
	endAngle := startAngle + (so.rng.Float64()-0.5)*math.Pi

	segments := so.generateLightningPath(info.X, info.Y, radius, startAngle, endAngle)

	return &Lightning{
		startAngle: startAngle,
		endAngle:   endAngle,
		segments:   segments,
		intensity:  1.0,
		startTime:  time.Now(),
		duration:   time.Duration(200+so.rng.Intn(200)) * time.Millisecond,
	}
}

// generateLightningPath creates jagged segments for a lightning bolt.
func (so *ShadowPlayOverlay) generateLightningPath(
	cx, cy, radius, startAngle, endAngle float64,
) []lightning {
	segments := make([]lightning, 0, 8)

	// Start point on dome edge.
	x1 := cx + radius*math.Cos(startAngle)
	y1 := cy + radius*math.Sin(startAngle)

	// End point on dome edge.
	endX := cx + radius*math.Cos(endAngle)
	endY := cy + radius*math.Sin(endAngle)

	// Create jagged path with 4-6 segments.
	numSegments := 4 + so.rng.Intn(3)
	for i := 0; i < numSegments; i++ {
		progress := float64(i+1) / float64(numSegments)

		// Interpolate between start and end with jitter.
		baseX := x1 + (endX-x1)*progress
		baseY := y1 + (endY-y1)*progress

		jitter := 15.0 * (1.0 - progress) // More jitter near start.
		x2 := baseX + (so.rng.Float64()-0.5)*jitter
		y2 := baseY + (so.rng.Float64()-0.5)*jitter

		segments = append(segments, lightning{x1, y1, x2, y2})
		x1, y1 = x2, y2
	}

	return segments
}

// computeDomeRadius calculates dome radius based on player count.
func (so *ShadowPlayOverlay) computeDomeRadius(info *ShadowPlayInfo) float64 {
	base := 80.0
	perPlayer := 8.0
	return base + float64(len(info.Players))*perPlayer
}

// Draw renders Shadow Play games to the screen.
func (so *ShadowPlayOverlay) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	so.mu.RLock()
	defer so.mu.RUnlock()

	for key, info := range so.games {
		phase := so.domePhase[key]
		bolts := so.lightnings[key]
		so.drawGame(screen, info, phase, bolts, camX, camY, zoom)
	}
}

// drawGame renders a single Shadow Play game.
func (so *ShadowPlayOverlay) drawGame(
	screen *ebiten.Image,
	info *ShadowPlayInfo,
	phase float64,
	bolts []*Lightning,
	camX, camY, zoom float64,
) {
	// Transform world coordinates to screen coordinates.
	_, _, centerX, centerY := getCameraSetup(screen)
	screenX, screenY := worldToScreen(info.X, info.Y, camX, camY, centerX, centerY, zoom)

	radius := so.computeDomeRadius(info) * zoom

	// Draw the shimmering dome.
	so.drawShimmeringDome(screen, screenX, screenY, radius, phase, info.State)

	// Draw lightning effects.
	for _, bolt := range bolts {
		so.drawLightning(screen, bolt, info.X, info.Y, camX, camY, centerX, centerY, zoom)
	}

	// Draw player nodes inside dome.
	so.drawPlayers(screen, info, camX, camY, centerX, centerY, zoom)

	// Draw game status indicator.
	so.drawStatusIndicator(screen, screenX, screenY, radius, info)
}

// drawShimmeringDome renders the dark dome with shimmer effect.
func (so *ShadowPlayOverlay) drawShimmeringDome(
	screen *ebiten.Image,
	cx, cy, radius, phase float64,
	state ShadowPlayState,
) {
	// Draw multiple concentric rings with varying opacity.
	numRings := 12
	for i := 0; i < numRings; i++ {
		ringProgress := float64(i) / float64(numRings)
		ringRadius := radius * (0.3 + 0.7*ringProgress)

		// Shimmer effect: opacity varies with phase and ring position.
		shimmer := 0.5 + 0.3*math.Sin(phase+ringProgress*math.Pi*2)
		baseAlpha := uint8(40 * (1.0 - ringProgress) * shimmer)

		// Color varies by state.
		col := so.getDomeColor(state, baseAlpha)

		so.drawRing(screen, cx, cy, ringRadius, 2.0, col)
	}

	// Draw dome outline.
	outlineColor := so.getDomeOutlineColor(state, phase)
	so.drawRing(screen, cx, cy, radius, 3.0, outlineColor)
}

// getDomeColor returns the dome fill color based on state.
func (so *ShadowPlayOverlay) getDomeColor(state ShadowPlayState, alpha uint8) color.RGBA {
	switch state {
	case ShadowPlayVoting:
		return color.RGBA{80, 40, 120, alpha} // Purple during voting.
	case ShadowPlayEchoesWin:
		return color.RGBA{40, 120, 80, alpha} // Green for echoes win.
	case ShadowPlayShadesWin:
		return color.RGBA{120, 40, 40, alpha} // Red for shades win.
	default:
		return color.RGBA{40, 40, 60, alpha} // Dark blue-gray.
	}
}

// getDomeOutlineColor returns the outline color with shimmer.
func (so *ShadowPlayOverlay) getDomeOutlineColor(state ShadowPlayState, phase float64) color.RGBA {
	alpha := uint8(150 + 50*math.Sin(phase*3))

	switch state {
	case ShadowPlayVoting:
		return color.RGBA{160, 80, 200, alpha} // Bright purple.
	case ShadowPlayEchoesWin:
		return color.RGBA{80, 200, 120, alpha} // Bright green.
	case ShadowPlayShadesWin:
		return color.RGBA{200, 80, 80, alpha} // Bright red.
	default:
		return color.RGBA{100, 100, 140, alpha} // Blue-gray.
	}
}

// drawRing renders a circular ring.
func (so *ShadowPlayOverlay) drawRing(
	screen *ebiten.Image,
	cx, cy, radius, strokeWidth float64,
	col color.RGBA,
) {
	drawSegmentedCircle(screen, cx, cy, radius, strokeWidth, col)
}

// drawLightning renders a lightning bolt effect.
func (so *ShadowPlayOverlay) drawLightning(
	screen *ebiten.Image,
	bolt *Lightning,
	worldX, worldY, camX, camY, centerX, centerY, zoom float64,
) {
	alpha := uint8(255 * bolt.intensity)
	col := color.RGBA{200, 200, 255, alpha} // Bright white-blue.
	glowCol := color.RGBA{150, 150, 255, alpha / 2}

	for _, seg := range bolt.segments {
		// Transform to screen coordinates.
		x1, y1 := worldToScreen(seg.x1, seg.y1, camX, camY, centerX, centerY, zoom)
		x2, y2 := worldToScreen(seg.x2, seg.y2, camX, camY, centerX, centerY, zoom)

		// Draw glow (wider, dimmer).
		vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), 4.0, glowCol, false)

		// Draw core (thinner, brighter).
		vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), 2.0, col, false)
	}
}

// drawPlayers renders player nodes inside the dome.
func (so *ShadowPlayOverlay) drawPlayers(
	screen *ebiten.Image,
	info *ShadowPlayInfo,
	camX, camY, centerX, centerY, zoom float64,
) {
	for _, player := range info.Players {
		so.drawPlayer(screen, &player, info.State, camX, camY, centerX, centerY, zoom)
	}
}

// drawPlayer renders a single player node with role-specific effects.
func (so *ShadowPlayOverlay) drawPlayer(
	screen *ebiten.Image,
	player *ShadowPlayer,
	gameState ShadowPlayState,
	camX, camY, centerX, centerY, zoom float64,
) {
	sx, sy := worldToScreen(player.X, player.Y, camX, camY, centerX, centerY, zoom)
	screenX := float32(sx)
	screenY := float32(sy)

	radius := float32(8 * zoom)
	if player.IsEliminated {
		radius *= 0.6
	}

	// Get color based on role visibility and elimination status.
	col := so.getPlayerColor(player, gameState)

	// Draw player marker.
	if player.IsEliminated {
		// Eliminated players are hollow.
		so.drawHollowCircle(screen, screenX, screenY, radius, col)
	} else {
		// Active players are filled with aura.
		so.drawPlayerAura(screen, screenX, screenY, radius, player, gameState)
		vector.DrawFilledCircle(screen, screenX, screenY, radius, col, false)
	}
}

// getPlayerColor returns the color for a player based on role and state.
func (so *ShadowPlayOverlay) getPlayerColor(player *ShadowPlayer, gameState ShadowPlayState) color.RGBA {
	if player.IsEliminated {
		return color.RGBA{80, 80, 80, 180} // Gray for eliminated.
	}

	// During game, roles may not be visible.
	switch player.Role {
	case ShadowRoleEcho:
		// Known echoes are blue-green.
		if gameState == ShadowPlayEchoesWin || gameState == ShadowPlayShadesWin {
			return color.RGBA{60, 180, 120, 230}
		}
		return color.RGBA{100, 150, 180, 220}

	case ShadowRoleShade:
		// Known shades are red-purple.
		if gameState == ShadowPlayEchoesWin || gameState == ShadowPlayShadesWin {
			return color.RGBA{180, 60, 80, 230}
		}
		return color.RGBA{140, 80, 120, 220}

	default:
		// Unknown role is neutral gray-blue.
		return color.RGBA{120, 120, 150, 220}
	}
}

// drawPlayerAura renders a pulsing aura around active players.
func (so *ShadowPlayOverlay) drawPlayerAura(
	screen *ebiten.Image,
	cx, cy, radius float32,
	player *ShadowPlayer,
	gameState ShadowPlayState,
) {
	if player.IsEliminated {
		return
	}

	// Aura pulses based on time.
	t := float64(time.Now().UnixMilli()) / 1000.0
	pulse := float32(0.3 + 0.2*math.Sin(t*2+float64(player.SpecterKey[0])))

	auraRadius := radius * (1.5 + pulse)
	auraColor := so.getPlayerColor(player, gameState)
	auraColor.A = uint8(float32(auraColor.A) * 0.3)

	vector.DrawFilledCircle(screen, cx, cy, auraRadius, auraColor, false)
}

// drawHollowCircle renders a ring for eliminated players.
func (so *ShadowPlayOverlay) drawHollowCircle(
	screen *ebiten.Image,
	cx, cy, radius float32,
	col color.RGBA,
) {
	numSegments := 24
	angleStep := 2 * math.Pi / float64(numSegments)

	for i := 0; i < numSegments; i++ {
		angle1 := float64(i) * angleStep
		angle2 := float64(i+1) * angleStep

		x1 := cx + radius*float32(math.Cos(angle1))
		y1 := cy + radius*float32(math.Sin(angle1))
		x2 := cx + radius*float32(math.Cos(angle2))
		y2 := cy + radius*float32(math.Sin(angle2))

		vector.StrokeLine(screen, x1, y1, x2, y2, 2.0, col, false)
	}
}

// drawStatusIndicator renders the game status at dome top.
func (so *ShadowPlayOverlay) drawStatusIndicator(
	screen *ebiten.Image,
	cx, cy, radius float64,
	info *ShadowPlayInfo,
) {
	// Draw round indicator at top of dome.
	indicatorY := cy - radius - 20

	// Draw small status icons.
	iconSize := float32(6)
	iconY := float32(indicatorY)

	switch info.State {
	case ShadowPlayWaiting:
		// Hourglass icon (two triangles).
		so.drawHourglass(screen, float32(cx), iconY, iconSize)

	case ShadowPlayVoting:
		// Ballot icon (rectangle with lines).
		so.drawBallot(screen, float32(cx), iconY, iconSize)

	case ShadowPlayEchoesWin:
		// Checkmark.
		so.drawCheckmark(screen, float32(cx), iconY, iconSize, color.RGBA{80, 200, 120, 255})

	case ShadowPlayShadesWin:
		// X mark.
		so.drawXMark(screen, float32(cx), iconY, iconSize, color.RGBA{200, 80, 80, 255})

	default:
		// Active game - show round number indicator.
		so.drawRoundDots(screen, float32(cx), iconY, info.RoundNumber)
	}
}

// drawHourglass renders a simple hourglass icon.
func (so *ShadowPlayOverlay) drawHourglass(screen *ebiten.Image, cx, cy, size float32) {
	col := color.RGBA{180, 180, 100, 220}

	// Top triangle.
	vector.StrokeLine(screen, cx-size, cy-size, cx+size, cy-size, 2, col, false)
	vector.StrokeLine(screen, cx-size, cy-size, cx, cy, 2, col, false)
	vector.StrokeLine(screen, cx+size, cy-size, cx, cy, 2, col, false)

	// Bottom triangle.
	vector.StrokeLine(screen, cx-size, cy+size, cx+size, cy+size, 2, col, false)
	vector.StrokeLine(screen, cx-size, cy+size, cx, cy, 2, col, false)
	vector.StrokeLine(screen, cx+size, cy+size, cx, cy, 2, col, false)
}

// drawBallot renders a ballot box icon.
func (so *ShadowPlayOverlay) drawBallot(screen *ebiten.Image, cx, cy, size float32) {
	col := color.RGBA{160, 80, 200, 220}

	// Box outline.
	x1, y1 := cx-size, cy-size
	x2, y2 := cx+size, cy+size

	vector.StrokeLine(screen, x1, y1, x2, y1, 2, col, false)
	vector.StrokeLine(screen, x2, y1, x2, y2, 2, col, false)
	vector.StrokeLine(screen, x2, y2, x1, y2, 2, col, false)
	vector.StrokeLine(screen, x1, y2, x1, y1, 2, col, false)

	// Slot at top.
	vector.StrokeLine(screen, cx-size/2, y1, cx+size/2, y1, 3, col, false)
}

// drawCheckmark renders a checkmark icon.
func (so *ShadowPlayOverlay) drawCheckmark(screen *ebiten.Image, cx, cy, size float32, col color.RGBA) {
	x1 := cx - size
	y1 := cy
	x2 := cx - size/3
	y2 := cy + size
	x3 := cx + size
	y3 := cy - size

	vector.StrokeLine(screen, x1, y1, x2, y2, 3, col, false)
	vector.StrokeLine(screen, x2, y2, x3, y3, 3, col, false)
}

// drawXMark renders an X icon.
func (so *ShadowPlayOverlay) drawXMark(screen *ebiten.Image, cx, cy, size float32, col color.RGBA) {
	vector.StrokeLine(screen, cx-size, cy-size, cx+size, cy+size, 3, col, false)
	vector.StrokeLine(screen, cx+size, cy-size, cx-size, cy+size, 3, col, false)
}

// drawRoundDots renders dots indicating the current round number.
func (so *ShadowPlayOverlay) drawRoundDots(screen *ebiten.Image, cx, cy float32, round int) {
	if round <= 0 {
		round = 1
	}
	if round > 5 {
		round = 5 // Cap at 5 dots.
	}

	dotRadius := float32(3)
	spacing := float32(8)
	totalWidth := float32(round-1) * spacing
	startX := cx - totalWidth/2

	col := color.RGBA{150, 150, 180, 220}

	for i := 0; i < round; i++ {
		dotX := startX + float32(i)*spacing
		vector.DrawFilledCircle(screen, dotX, cy, dotRadius, col, false)
	}
}

// GetGame retrieves game info by ID.
func (so *ShadowPlayOverlay) GetGame(gameID [32]byte) *ShadowPlayInfo {
	so.mu.RLock()
	defer so.mu.RUnlock()
	return so.games[gameKey(gameID)]
}

// GameCount returns the number of active games.
func (so *ShadowPlayOverlay) GameCount() int {
	so.mu.RLock()
	defer so.mu.RUnlock()
	return len(so.games)
}

// Clear removes all games from the overlay.
func (so *ShadowPlayOverlay) Clear() {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.games = make(map[string]*ShadowPlayInfo)
	so.lightnings = make(map[string][]*Lightning)
	so.domePhase = make(map[string]float64)
}
