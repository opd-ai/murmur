// Package overlays - Surface Spark Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: "Surface Sparks are lightweight, Surface-Layer-exclusive
// challenge mechanics that give Open-mode users a taste of gamified interaction."
// Per ROADMAP.md line 559: "Pulse Map visualization for active Sparks"
//
// Active Sparks appear on the Pulse Map as pulsing star icons near their initiator nodes.
// Wave Relay sparks show a creative constraint beacon. Echo Race sparks show a racing flag.
// Winners display a golden crown glyph for SparkCrownDuration (1 hour).
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// SparkType identifies the type of Surface Spark.
type SparkType uint8

const (
	SparkWaveRelay SparkType = iota + 1 // Wave Relay challenge.
	SparkEchoRace                       // Echo Race - first amplifier wins.
)

// SparkState represents the lifecycle state of a Spark.
type SparkState uint8

const (
	SparkActive    SparkState = iota + 1 // Spark is accepting responses.
	SparkCompleted                       // Spark has ended.
	SparkExpired                         // Spark timed out.
	SparkCancelled                       // Spark cancelled by initiator.
)

// SparkInfo contains information about a Spark for visualization.
type SparkInfo struct {
	ID          [32]byte   // Unique spark ID.
	Type        SparkType  // WaveRelay or EchoRace.
	State       SparkState // Current state.
	X, Y        float64    // Position on the Pulse Map (near initiator).
	Prompt      string     // Creative constraint for WaveRelay.
	CreatedAt   time.Time  // When the spark was initiated.
	ExpiresAt   time.Time  // When the spark challenge window closes.
	WinnerKey   [32]byte   // Public key of winner (for completed EchoRace).
	Responses   int        // Number of responses received.
	InitiatorID [32]byte   // Initiator's public key.
}

// CrownHolder represents a user holding a winner crown.
type CrownHolder struct {
	UserKey   [32]byte  // Public key of crown holder.
	X, Y      float64   // Position on the Pulse Map.
	ExpiresAt time.Time // When the crown expires.
}

// SparkOverlay renders Surface Spark events on the Pulse Map.
type SparkOverlay struct {
	mu sync.RWMutex

	visible      bool
	sparks       map[[32]byte]*SparkInfo
	crownHolders map[[32]byte]*CrownHolder
	time         float64 // Animation time.

	// Visual settings.
	waveRelayColor color.RGBA // Blue beacon for Wave Relay.
	echoRaceColor  color.RGBA // Green racing flag for Echo Race.
	activeGlow     color.RGBA // Pulsing glow for active sparks.
	expiredColor   color.RGBA // Dimmed color for expired sparks.
	crownColor     color.RGBA // Golden crown color.
	completedColor color.RGBA // Color for completed sparks.
	responseColor  color.RGBA // Color for response indicators.
}

// NewSparkOverlay creates a new Surface Spark overlay.
func NewSparkOverlay() *SparkOverlay {
	return &SparkOverlay{
		visible:      true,
		sparks:       make(map[[32]byte]*SparkInfo),
		crownHolders: make(map[[32]byte]*CrownHolder),
		waveRelayColor: color.RGBA{
			R: 100,
			G: 150,
			B: 255,
			A: 255,
		},
		echoRaceColor: color.RGBA{
			R: 80,
			G: 220,
			B: 120,
			A: 255,
		},
		activeGlow: color.RGBA{
			R: 255,
			G: 220,
			B: 100,
			A: 120,
		},
		expiredColor: color.RGBA{
			R: 120,
			G: 120,
			B: 130,
			A: 180,
		},
		crownColor: color.RGBA{
			R: 255,
			G: 215,
			B: 0,
			A: 255,
		},
		completedColor: color.RGBA{
			R: 180,
			G: 180,
			B: 200,
			A: 200,
		},
		responseColor: color.RGBA{
			R: 200,
			G: 200,
			B: 255,
			A: 180,
		},
	}
}

// SetVisible controls visibility.
func (o *SparkOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *SparkOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetSpark adds or updates a spark.
func (o *SparkOverlay) SetSpark(spark *SparkInfo) {
	if spark == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sparks[spark.ID] = spark
}

// RemoveSpark removes a spark by ID.
func (o *SparkOverlay) RemoveSpark(id [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.sparks, id)
}

// GetSpark returns a spark by ID.
func (o *SparkOverlay) GetSpark(id [32]byte) *SparkInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.sparks[id]
}

// GetAllSparks returns all sparks.
func (o *SparkOverlay) GetAllSparks() []*SparkInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	sparks := make([]*SparkInfo, 0, len(o.sparks))
	for _, s := range o.sparks {
		sparks = append(sparks, s)
	}
	return sparks
}

// GetActiveSparks returns only active sparks.
func (o *SparkOverlay) GetActiveSparks() []*SparkInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var active []*SparkInfo
	for _, s := range o.sparks {
		if s.State == SparkActive {
			active = append(active, s)
		}
	}
	return active
}

// SetCrownHolder adds or updates a crown holder.
func (o *SparkOverlay) SetCrownHolder(holder *CrownHolder) {
	if holder == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.crownHolders[holder.UserKey] = holder
}

// RemoveCrownHolder removes a crown holder.
func (o *SparkOverlay) RemoveCrownHolder(userKey [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.crownHolders, userKey)
}

// GetCrownHolder returns a crown holder by key.
func (o *SparkOverlay) GetCrownHolder(userKey [32]byte) *CrownHolder {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.crownHolders[userKey]
}

// HasCrown checks if a user has an active crown.
func (o *SparkOverlay) HasCrown(userKey [32]byte) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	holder, ok := o.crownHolders[userKey]
	if !ok {
		return false
	}
	return time.Now().Before(holder.ExpiresAt)
}

// Update advances animation state.
func (o *SparkOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.time += dt

	// Purge expired crowns.
	now := time.Now()
	for key, holder := range o.crownHolders {
		if now.After(holder.ExpiresAt) {
			delete(o.crownHolders, key)
		}
	}
}

// Draw renders the spark events and crowns.
func (o *SparkOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible {
		return
	}

	screenW, screenH, centerX, centerY := getCameraSetup(screen)

	// Draw sparks.
	for _, spark := range o.sparks {
		sx, sy := worldToScreen(spark.X, spark.Y, cameraX, cameraY, centerX, centerY, zoom)

		// Skip if off-screen.
		if isOffScreen(sx, sy, screenW, screenH, 100) {
			continue
		}

		o.drawSpark(screen, float32(sx), float32(sy), float32(zoom), spark)
	}

	// Draw crowns.
	now := time.Now()
	for _, holder := range o.crownHolders {
		if now.After(holder.ExpiresAt) {
			continue
		}

		sx := centerX + (holder.X-cameraX)*zoom
		sy := centerY + (holder.Y-cameraY)*zoom

		// Skip if off-screen.
		if sx < -100 || sx > screenW+100 || sy < -100 || sy > screenH+100 {
			continue
		}

		o.drawCrown(screen, float32(sx), float32(sy), float32(zoom))
	}
}

// drawSpark draws a single spark icon based on type and state.
func (o *SparkOverlay) drawSpark(screen *ebiten.Image, x, y, zoom float32, spark *SparkInfo) {
	scale := zoom * 0.5
	if scale < 0.3 {
		scale = 0.3
	}
	if scale > 1.5 {
		scale = 1.5
	}

	// Draw glow for active sparks.
	if spark.State == SparkActive {
		o.drawActiveGlow(screen, x, y, scale)
	}

	// Draw icon based on type.
	switch spark.Type {
	case SparkWaveRelay:
		o.drawWaveRelayIcon(screen, x, y, scale, spark)
	case SparkEchoRace:
		o.drawEchoRaceIcon(screen, x, y, scale, spark)
	}

	// Draw response indicators.
	if spark.Responses > 0 {
		o.drawResponseIndicators(screen, x, y, scale, spark.Responses)
	}

	// Draw time remaining indicator for active sparks.
	if spark.State == SparkActive {
		o.drawTimeRemaining(screen, x, y, scale, spark)
	}
}

// drawActiveGlow draws a pulsing glow for active sparks.
func (o *SparkOverlay) drawActiveGlow(screen *ebiten.Image, x, y, scale float32) {
	pulsePhase := float32(math.Sin(o.time * 3))
	baseRadius := 35 * scale
	pulseRadius := baseRadius + 8*scale*pulsePhase

	glowAlpha := uint8(80 + 40*pulsePhase)
	glowColor := color.RGBA{
		R: o.activeGlow.R,
		G: o.activeGlow.G,
		B: o.activeGlow.B,
		A: glowAlpha,
	}

	// Draw multiple glow rings.
	for i := 3; i >= 0; i-- {
		radius := pulseRadius * float32(1+i) / 3
		alpha := uint8(float32(glowAlpha) / float32(i+2))
		ringColor := color.RGBA{glowColor.R, glowColor.G, glowColor.B, alpha}
		vector.DrawFilledCircle(screen, x, y, radius, ringColor, true)
	}
}

// drawWaveRelayIcon draws the Wave Relay beacon icon.
func (o *SparkOverlay) drawWaveRelayIcon(screen *ebiten.Image, x, y, scale float32, spark *SparkInfo) {
	baseColor := o.waveRelayColor
	if spark.State == SparkExpired || spark.State == SparkCancelled {
		baseColor = o.expiredColor
	} else if spark.State == SparkCompleted {
		baseColor = o.completedColor
	}

	// Draw beacon base (circle with radiating lines).
	baseRadius := 12 * scale
	vector.DrawFilledCircle(screen, x, y, baseRadius, baseColor, true)

	// Draw inner highlight.
	innerColor := color.RGBA{
		R: uint8(min(int(baseColor.R)+50, 255)),
		G: uint8(min(int(baseColor.G)+50, 255)),
		B: uint8(min(int(baseColor.B)+50, 255)),
		A: 255,
	}
	vector.DrawFilledCircle(screen, x, y-2*scale, baseRadius*0.6, innerColor, true)

	// Draw radiating lines (beacon rays) for active sparks.
	if spark.State == SparkActive {
		numRays := 6
		rayLength := 20 * scale
		rayWidth := float32(2.0)

		for i := 0; i < numRays; i++ {
			angle := o.time*0.5 + float64(i)*math.Pi/float64(numRays/2)
			// Pulsing ray length.
			pulse := float32(math.Sin(o.time*2+float64(i))*0.3 + 0.7)
			actualLength := rayLength * pulse

			endX := x + actualLength*float32(math.Cos(angle))
			endY := y + actualLength*float32(math.Sin(angle))

			rayAlpha := uint8(150 * pulse)
			rayColor := color.RGBA{baseColor.R, baseColor.G, baseColor.B, rayAlpha}

			vector.StrokeLine(screen, x, y, endX, endY, rayWidth, rayColor, true)
		}
	}

	// Draw "W" symbol in center.
	o.drawLetter(screen, x, y, scale*0.8, "W", color.RGBA{255, 255, 255, 220})
}

// drawEchoRaceIcon draws the Echo Race racing flag icon.
func (o *SparkOverlay) drawEchoRaceIcon(screen *ebiten.Image, x, y, scale float32, spark *SparkInfo) {
	baseColor := o.getSparkBaseColor(spark)

	poleHeight := 25 * scale
	poleWidth := 2 * scale
	o.drawFlagPole(screen, x, y, poleWidth, poleHeight)

	flagWidth := 18 * scale
	flagHeight := 12 * scale
	flagX := x
	flagY := y - poleHeight/2 + flagHeight/2

	o.drawCheckeredFlag(screen, flagX, flagY, flagWidth, flagHeight, baseColor)

	if spark.State == SparkActive {
		o.drawFlagMotionLines(screen, flagX, flagY, flagWidth, scale, baseColor)
	}

	o.drawLetter(screen, x+flagWidth/2, flagY, scale*0.6, "E", color.RGBA{255, 255, 255, 200})
}

// getSparkBaseColor determines the base color for a spark icon based on its state.
func (o *SparkOverlay) getSparkBaseColor(spark *SparkInfo) color.RGBA {
	if spark.State == SparkExpired || spark.State == SparkCancelled {
		return o.expiredColor
	}
	if spark.State == SparkCompleted {
		return o.completedColor
	}
	return o.echoRaceColor
}

// drawFlagPole draws the vertical pole of a flag icon.
func (o *SparkOverlay) drawFlagPole(screen *ebiten.Image, x, y, poleWidth, poleHeight float32) {
	poleColor := color.RGBA{80, 80, 90, 255}
	vector.DrawFilledRect(screen, x-poleWidth/2, y-poleHeight/2, poleWidth, poleHeight, poleColor, true)
}

// drawCheckeredFlag draws a checkered flag pattern.
func (o *SparkOverlay) drawCheckeredFlag(screen *ebiten.Image, flagX, flagY, flagWidth, flagHeight float32, baseColor color.RGBA) {
	vector.DrawFilledRect(screen, flagX, flagY-flagHeight/2, flagWidth, flagHeight, baseColor, true)

	checkSize := flagWidth / 4
	darkColor := color.RGBA{30, 30, 40, 255}
	for row := 0; row < 2; row++ {
		for col := 0; col < 4; col++ {
			if (row+col)%2 == 0 {
				cx := flagX + float32(col)*checkSize
				cy := flagY - flagHeight/2 + float32(row)*checkSize*1.5
				vector.DrawFilledRect(screen, cx, cy, checkSize, flagHeight/2, darkColor, true)
			}
		}
	}
}

// drawFlagMotionLines draws animated motion lines for active flag waving.
func (o *SparkOverlay) drawFlagMotionLines(screen *ebiten.Image, flagX, flagY, flagWidth, scale float32, baseColor color.RGBA) {
	wavePhase := float32(math.Sin(o.time * 4))
	for i := 0; i < 3; i++ {
		lineX := flagX + flagWidth + 5*scale + float32(i)*4*scale
		lineY := flagY + wavePhase*2*scale + float32(i)*2*scale
		lineLen := 8*scale - float32(i)*2*scale
		lineAlpha := uint8(150 - i*40)
		lineColor := color.RGBA{baseColor.R, baseColor.G, baseColor.B, lineAlpha}
		vector.StrokeLine(screen, lineX, lineY-lineLen/2, lineX, lineY+lineLen/2, 2, lineColor, true)
	}
}

// drawLetter draws a simplified letter at position.
func (o *SparkOverlay) drawLetter(screen *ebiten.Image, x, y, scale float32, letter string, c color.RGBA) {
	strokeWidth := 2 * scale
	size := 8 * scale

	switch letter {
	case "W":
		// Draw W shape.
		vector.StrokeLine(screen, x-size/2, y-size/2, x-size/4, y+size/2, strokeWidth, c, true)
		vector.StrokeLine(screen, x-size/4, y+size/2, x, y, strokeWidth, c, true)
		vector.StrokeLine(screen, x, y, x+size/4, y+size/2, strokeWidth, c, true)
		vector.StrokeLine(screen, x+size/4, y+size/2, x+size/2, y-size/2, strokeWidth, c, true)
	case "E":
		// Draw E shape.
		vector.StrokeLine(screen, x-size/2, y-size/2, x-size/2, y+size/2, strokeWidth, c, true)
		vector.StrokeLine(screen, x-size/2, y-size/2, x+size/2, y-size/2, strokeWidth, c, true)
		vector.StrokeLine(screen, x-size/2, y, x+size/4, y, strokeWidth, c, true)
		vector.StrokeLine(screen, x-size/2, y+size/2, x+size/2, y+size/2, strokeWidth, c, true)
	}
}

// drawResponseIndicators draws small dots showing response count.
func (o *SparkOverlay) drawResponseIndicators(screen *ebiten.Image, x, y, scale float32, count int) {
	if count == 0 {
		return
	}

	// Limit displayed dots.
	displayCount := count
	if displayCount > 8 {
		displayCount = 8
	}

	// Arrange in arc below the spark.
	arcRadius := 25 * scale
	arcStart := math.Pi * 0.6
	arcEnd := math.Pi * 0.9
	arcSpan := arcEnd - arcStart

	for i := 0; i < displayCount; i++ {
		var angle float64
		if displayCount == 1 {
			angle = (arcStart + arcEnd) / 2
		} else {
			angle = arcStart + arcSpan*float64(i)/float64(displayCount-1)
		}

		dotX := x + arcRadius*float32(math.Cos(angle))
		dotY := y + arcRadius*float32(math.Sin(angle))

		// Pulsing dots.
		pulse := float32(math.Sin(o.time*2+float64(i)*0.3)*0.3 + 0.7)
		dotRadius := 3 * scale * pulse

		dotAlpha := uint8(180 * pulse)
		dotColor := color.RGBA{o.responseColor.R, o.responseColor.G, o.responseColor.B, dotAlpha}

		vector.DrawFilledCircle(screen, dotX, dotY, dotRadius, dotColor, true)
	}

	// Show "+" indicator if more than displayed.
	if count > displayCount {
		plusX := x + arcRadius*float32(math.Cos(arcEnd+0.15))
		plusY := y + arcRadius*float32(math.Sin(arcEnd+0.15))
		plusColor := color.RGBA{200, 200, 200, 160}
		vector.DrawFilledCircle(screen, plusX, plusY, 4*scale, plusColor, true)
	}
}

// drawTimeRemaining draws a countdown arc for active sparks.
func (o *SparkOverlay) drawTimeRemaining(screen *ebiten.Image, x, y, scale float32, spark *SparkInfo) {
	now := time.Now()
	if now.After(spark.ExpiresAt) {
		return
	}

	fraction := calculateTimeFraction(spark, now)
	if fraction <= 0 {
		return
	}

	arcRadius := 18 * scale
	arcWidth := float32(3.0)
	arcColor := selectArcColor(fraction)

	drawTimerArc(screen, x, y, arcRadius, arcWidth, fraction, arcColor)
}

// calculateTimeFraction computes remaining time as fraction of total.
func calculateTimeFraction(spark *SparkInfo, now time.Time) float64 {
	total := spark.ExpiresAt.Sub(spark.CreatedAt).Seconds()
	if total <= 0 {
		return 0
	}
	remaining := spark.ExpiresAt.Sub(now).Seconds()
	fraction := remaining / total
	if fraction > 1 {
		return 1
	}
	if fraction < 0 {
		return 0
	}
	return fraction
}

// selectArcColor returns color based on remaining time fraction.
func selectArcColor(fraction float64) color.RGBA {
	if fraction > 0.5 {
		return color.RGBA{80, 200, 80, 200}
	}
	if fraction > 0.2 {
		return color.RGBA{220, 200, 50, 200}
	}
	return color.RGBA{220, 80, 80, 200}
}

// drawTimerArc renders segmented arc indicating time remaining.
func drawTimerArc(screen *ebiten.Image, x, y, radius, width float32, fraction float64, arcColor color.RGBA) {
	startAngle := -math.Pi / 2
	endAngle := startAngle + 2*math.Pi*fraction
	segments := int(fraction * 20)
	if segments < 2 {
		segments = 2
	}

	for i := 0; i < segments; i++ {
		t0 := float64(i) / float64(segments)
		t1 := float64(i+1) / float64(segments)
		a0 := startAngle + (endAngle-startAngle)*t0
		a1 := startAngle + (endAngle-startAngle)*t1

		x0 := x + radius*float32(math.Cos(a0))
		y0 := y + radius*float32(math.Sin(a0))
		x1 := x + radius*float32(math.Cos(a1))
		y1 := y + radius*float32(math.Sin(a1))

		vector.StrokeLine(screen, x0, y0, x1, y1, width, arcColor, true)
	}
}

// drawCrown draws a golden crown above a node.
func (o *SparkOverlay) drawCrown(screen *ebiten.Image, x, y, zoom float32) {
	scale := zoom * 0.5
	if scale < 0.3 {
		scale = 0.3
	}
	if scale > 1.5 {
		scale = 1.5
	}

	// Position crown above the node.
	crownY := y - 30*scale

	// Crown base.
	baseWidth := 24 * scale
	baseHeight := 6 * scale
	vector.DrawFilledRect(screen, x-baseWidth/2, crownY, baseWidth, baseHeight, o.crownColor, true)

	// Crown points (3 points).
	pointHeight := 12 * scale
	pointWidth := 6 * scale

	// Left point.
	o.drawCrownPoint(screen, x-baseWidth/3, crownY, pointWidth, pointHeight, scale)
	// Center point (taller).
	o.drawCrownPoint(screen, x, crownY, pointWidth, pointHeight*1.3, scale)
	// Right point.
	o.drawCrownPoint(screen, x+baseWidth/3, crownY, pointWidth, pointHeight, scale)

	// Draw jewels on points.
	jewelColor := color.RGBA{255, 50, 50, 255}
	jewelRadius := 2 * scale

	vector.DrawFilledCircle(screen, x-baseWidth/3, crownY-pointHeight+jewelRadius*2, jewelRadius, jewelColor, true)
	vector.DrawFilledCircle(screen, x, crownY-pointHeight*1.3+jewelRadius*2, jewelRadius, jewelColor, true)
	vector.DrawFilledCircle(screen, x+baseWidth/3, crownY-pointHeight+jewelRadius*2, jewelRadius, jewelColor, true)

	// Sparkle effect.
	sparklePhase := float32(math.Sin(o.time * 4))
	if sparklePhase > 0.7 {
		sparkleX := x + 8*scale*float32(math.Cos(o.time*2))
		sparkleY := crownY - 6*scale
		sparkleColor := color.RGBA{255, 255, 255, uint8(200 * sparklePhase)}
		vector.DrawFilledCircle(screen, sparkleX, sparkleY, 2*scale, sparkleColor, true)
	}
}

// drawCrownPoint draws a single crown point (triangle).
func (o *SparkOverlay) drawCrownPoint(screen *ebiten.Image, x, baseY, width, height, scale float32) {
	// Approximate triangle with filled rect tapering upward.
	// Draw multiple rectangles decreasing in width.
	segments := 5
	segHeight := height / float32(segments)

	for i := 0; i < segments; i++ {
		t := float32(i) / float32(segments)
		segWidth := width * (1 - t*0.7)
		segY := baseY - float32(i+1)*segHeight

		// Gradient from gold to lighter gold.
		r := uint8(min(int(o.crownColor.R)+int(t*30), 255))
		g := uint8(min(int(o.crownColor.G)+int(t*20), 255))
		b := uint8(int(float32(o.crownColor.B) * (1 + t*0.5)))
		segColor := color.RGBA{r, g, b, 255}

		vector.DrawFilledRect(screen, x-segWidth/2, segY, segWidth, segHeight+1, segColor, true)
	}
}

// SparkCount returns the total number of sparks.
func (o *SparkOverlay) SparkCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.sparks)
}

// ActiveSparkCount returns the number of active sparks.
func (o *SparkOverlay) ActiveSparkCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	count := 0
	for _, s := range o.sparks {
		if s.State == SparkActive {
			count++
		}
	}
	return count
}

// GetExpiresAt implements Expires interface for CrownHolder.
func (c *CrownHolder) GetExpiresAt() time.Time {
	return c.ExpiresAt
}

// CrownCount returns the number of active crown holders.
func (o *SparkOverlay) CrownCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return CountNonExpiredInMap(o.crownHolders)
}

// ClearExpired removes expired sparks older than a threshold.
func (o *SparkOverlay) ClearExpired(maxAge time.Duration) int {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, spark := range o.sparks {
		if spark.State != SparkActive {
			age := now.Sub(spark.ExpiresAt)
			if age > maxAge {
				delete(o.sparks, id)
				removed++
			}
		}
	}

	return removed
}

// SparkTypeString returns a human-readable name for a spark type.
func SparkTypeString(st SparkType) string {
	switch st {
	case SparkWaveRelay:
		return "Wave Relay"
	case SparkEchoRace:
		return "Echo Race"
	default:
		return "Unknown"
	}
}

// SparkStateString returns a human-readable name for a spark state.
func SparkStateString(ss SparkState) string {
	switch ss {
	case SparkActive:
		return "Active"
	case SparkCompleted:
		return "Completed"
	case SparkExpired:
		return "Expired"
	case SparkCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// UpdateCrownPosition updates the position of a crown holder.
func (o *SparkOverlay) UpdateCrownPosition(userKey [32]byte, x, y float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if holder, ok := o.crownHolders[userKey]; ok {
		holder.X = x
		holder.Y = y
	}
}

// UpdateSparkPosition updates the position of a spark.
func (o *SparkOverlay) UpdateSparkPosition(id [32]byte, x, y float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if spark, ok := o.sparks[id]; ok {
		spark.X = x
		spark.Y = y
	}
}

// SetSparkResponses updates the response count for a spark.
func (o *SparkOverlay) SetSparkResponses(id [32]byte, count int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if spark, ok := o.sparks[id]; ok {
		spark.Responses = count
	}
}

// SetSparkState updates the state of a spark.
func (o *SparkOverlay) SetSparkState(id [32]byte, state SparkState) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if spark, ok := o.sparks[id]; ok {
		spark.State = state
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
