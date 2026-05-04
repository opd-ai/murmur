// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This file implements Echo Index visual color-coding on the Pulse Map.
// Per RESONANCE_SYSTEM.md §Echo Index Display:
// - High Echo Index (>0.7) = warm colors (amber to red) indicating insularity
// - Low Echo Index (<0.4) = cool colors (blue to green) indicating openness
// - Mid-range = neutral (gray to white)
//

package overlays

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/anonymous/resonance"
)

// EchoIndexOverlay renders Echo Index color-coding on cluster boundaries.
// Per RESONANCE_SYSTEM.md: displayed as color-coded badges on cluster boundaries.
type EchoIndexOverlay struct {
	// ClusterBoundaries maps cluster IDs to their boundary polygons.
	// Each boundary is a list of (x, y) vertex pairs.
	ClusterBoundaries map[string][]float32

	// ClusterCenters maps cluster IDs to their center points (x, y).
	ClusterCenters map[string][2]float32

	// Computer provides Echo Index values for clusters.
	Computer *resonance.EchoIndexComputer

	// IsAnonymousLayer indicates whether this overlay is for Echo Shadow.
	IsAnonymousLayer bool

	// BadgeRadius is the radius of the Echo Index badge circle.
	BadgeRadius float32

	// ShowBadges enables badge rendering at cluster centers.
	ShowBadges bool

	// ShowTint enables tinted boundary fill.
	ShowTint bool

	// TintAlpha controls the opacity of boundary tints (0-255).
	TintAlpha uint8
}

// NewEchoIndexOverlay creates a new Echo Index overlay.
func NewEchoIndexOverlay(computer *resonance.EchoIndexComputer) *EchoIndexOverlay {
	return &EchoIndexOverlay{
		ClusterBoundaries: make(map[string][]float32),
		ClusterCenters:    make(map[string][2]float32),
		Computer:          computer,
		IsAnonymousLayer:  false,
		BadgeRadius:       12.0,
		ShowBadges:        true,
		ShowTint:          true,
		TintAlpha:         40, // Subtle tint
	}
}

// NewEchoShadowOverlay creates an Echo Shadow overlay for the Anonymous Layer.
// Per RESONANCE_SYSTEM.md: Echo Shadow is displayed as a subtle color field
// around Specter clusters.
func NewEchoShadowOverlay(shadow *resonance.EchoShadow) *EchoIndexOverlay {
	return &EchoIndexOverlay{
		ClusterBoundaries: make(map[string][]float32),
		ClusterCenters:    make(map[string][2]float32),
		Computer:          shadow.EchoIndexComputer,
		IsAnonymousLayer:  true,
		BadgeRadius:       10.0, // Smaller for anonymous layer
		ShowBadges:        true,
		ShowTint:          true,
		TintAlpha:         30, // More subtle for anonymous layer
	}
}

// SetClusterBoundary sets the boundary polygon for a cluster.
// Vertices are specified as alternating x, y values: [x1, y1, x2, y2, ...].
func (o *EchoIndexOverlay) SetClusterBoundary(clusterID string, vertices []float32) {
	if len(vertices) >= 6 { // At least 3 vertices (triangle)
		o.ClusterBoundaries[clusterID] = vertices
	}
}

// SetClusterCenter sets the center point for a cluster.
func (o *EchoIndexOverlay) SetClusterCenter(clusterID string, x, y float32) {
	o.ClusterCenters[clusterID] = [2]float32{x, y}
}

// RemoveCluster removes a cluster from the overlay.
func (o *EchoIndexOverlay) RemoveCluster(clusterID string) {
	delete(o.ClusterBoundaries, clusterID)
	delete(o.ClusterCenters, clusterID)
}

// Clear removes all cluster data.
func (o *EchoIndexOverlay) Clear() {
	o.ClusterBoundaries = make(map[string][]float32)
	o.ClusterCenters = make(map[string][2]float32)
}

// Render draws the Echo Index overlay on the destination image.
// Camera parameters transform world coordinates to screen coordinates.
func (o *EchoIndexOverlay) Render(dst *ebiten.Image, cameraX, cameraY, scale float32, time float64) {
	if o.Computer == nil {
		return
	}

	screenW := float32(dst.Bounds().Dx())
	screenH := float32(dst.Bounds().Dy())
	halfW := screenW / 2
	halfH := screenH / 2

	// Draw each cluster's boundary and badge.
	for clusterID, boundary := range o.ClusterBoundaries {
		echoIndex := o.getClusterEchoIndex(clusterID)
		clusterColor := colorForEchoIndex(echoIndex)

		if o.ShowTint && len(boundary) >= 6 {
			o.renderClusterTint(dst, boundary, clusterColor, cameraX, cameraY, scale, halfW, halfH)
		}
	}

	// Draw badges on top of tints.
	if o.ShowBadges {
		for clusterID, center := range o.ClusterCenters {
			echoIndex := o.getClusterEchoIndex(clusterID)
			clusterColor := colorForEchoIndex(echoIndex)

			screenX := (center[0]-cameraX)*scale + halfW
			screenY := (center[1]-cameraY)*scale + halfH

			// Skip if off-screen.
			if screenX < -50 || screenX > screenW+50 || screenY < -50 || screenY > screenH+50 {
				continue
			}

			o.renderBadge(dst, screenX, screenY, clusterColor, echoIndex, time)
		}
	}
}

// getClusterEchoIndex retrieves the Echo Index for a cluster.
func (o *EchoIndexOverlay) getClusterEchoIndex(clusterID string) float64 {
	if idx := o.Computer.GetClusterIndex(clusterID); idx != nil {
		return idx.EchoIndex
	}
	return 0.5 // Default to neutral
}

// renderClusterTint draws a semi-transparent fill over the cluster boundary.
func (o *EchoIndexOverlay) renderClusterTint(
	dst *ebiten.Image,
	boundary []float32,
	baseColor color.RGBA,
	cameraX, cameraY, scale, halfW, halfH float32,
) {
	// Transform boundary vertices to screen coordinates.
	screenVerts := make([]float32, len(boundary))
	for i := 0; i < len(boundary); i += 2 {
		screenVerts[i] = (boundary[i]-cameraX)*scale + halfW
		screenVerts[i+1] = (boundary[i+1]-cameraY)*scale + halfH
	}

	// Create tint color with configured alpha.
	tintColor := color.RGBA{
		R: baseColor.R,
		G: baseColor.G,
		B: baseColor.B,
		A: o.TintAlpha,
	}

	// Draw filled polygon using triangle fan from centroid.
	// First, compute centroid.
	var cx, cy float32
	numVerts := len(screenVerts) / 2
	for i := 0; i < numVerts; i++ {
		cx += screenVerts[i*2]
		cy += screenVerts[i*2+1]
	}
	cx /= float32(numVerts)
	cy /= float32(numVerts)

	// Draw triangles from centroid to each edge.
	for i := 0; i < numVerts; i++ {
		x1 := screenVerts[i*2]
		y1 := screenVerts[i*2+1]
		j := (i + 1) % numVerts
		x2 := screenVerts[j*2]
		y2 := screenVerts[j*2+1]

		// Draw triangle (centroid, vertex i, vertex i+1).
		drawFilledTriangle(dst, cx, cy, x1, y1, x2, y2, tintColor)
	}

	// Draw boundary outline.
	outlineColor := color.RGBA{
		R: baseColor.R,
		G: baseColor.G,
		B: baseColor.B,
		A: o.TintAlpha + 30,
	}
	for i := 0; i < numVerts; i++ {
		x1 := screenVerts[i*2]
		y1 := screenVerts[i*2+1]
		j := (i + 1) % numVerts
		x2 := screenVerts[j*2]
		y2 := screenVerts[j*2+1]
		vector.StrokeLine(dst, x1, y1, x2, y2, 1.0, outlineColor, true)
	}
}

// renderBadge draws the Echo Index badge at a screen position.
func (o *EchoIndexOverlay) renderBadge(
	dst *ebiten.Image,
	x, y float32,
	badgeColor color.RGBA,
	echoIndex float64,
	time float64,
) {
	radius := o.BadgeRadius

	// Animate badge with subtle pulse for insular/open clusters.
	category := resonance.CategoryFromEchoIndex(echoIndex)
	if category == resonance.EchoCategoryInsular || category == resonance.EchoCategoryOpen {
		pulse := float32(1.0 + 0.1*math.Sin(time*2.0))
		radius *= pulse
	}

	// Draw badge background.
	bgColor := color.RGBA{
		R: badgeColor.R,
		G: badgeColor.G,
		B: badgeColor.B,
		A: 180,
	}
	vector.DrawFilledCircle(dst, x, y, radius, bgColor, true)

	// Draw badge border.
	borderColor := color.RGBA{
		R: uint8(min32(float32(badgeColor.R)+50, 255)),
		G: uint8(min32(float32(badgeColor.G)+50, 255)),
		B: uint8(min32(float32(badgeColor.B)+50, 255)),
		A: 220,
	}
	vector.StrokeCircle(dst, x, y, radius, 1.5, borderColor, true)

	// Draw category indicator.
	// Insular: flame icon (simplified as upward triangle)
	// Open: wave icon (simplified as horizontal lines)
	// Neutral: no icon
	iconColor := color.RGBA{255, 255, 255, 200}
	iconSize := radius * 0.5

	switch category {
	case resonance.EchoCategoryInsular:
		// Draw flame (upward triangle).
		drawFilledTriangle(dst,
			x, y-iconSize, // Top
			x-iconSize*0.6, y+iconSize*0.4, // Bottom left
			x+iconSize*0.6, y+iconSize*0.4, // Bottom right
			iconColor)
	case resonance.EchoCategoryOpen:
		// Draw wave (three horizontal lines).
		for i := -1; i <= 1; i++ {
			yOff := float32(i) * iconSize * 0.5
			vector.StrokeLine(dst, x-iconSize*0.6, y+yOff, x+iconSize*0.6, y+yOff, 1.0, iconColor, true)
		}
	}
}

// colorForEchoIndex returns the display color for an Echo Index value.
// Uses the same logic as resonance.ColorForEchoIndex but returns color.RGBA.
func colorForEchoIndex(index float64) color.RGBA {
	r, g, b := resonance.ColorForEchoIndex(index)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// drawFilledTriangle draws a filled triangle using vector graphics.
func drawFilledTriangle(dst *ebiten.Image, x1, y1, x2, y2, x3, y3 float32, c color.RGBA) {
	// Create path for triangle.
	var path vector.Path
	path.MoveTo(x1, y1)
	path.LineTo(x2, y2)
	path.LineTo(x3, y3)
	path.Close()

	// Fill the triangle.
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(c.R) / 255
		vs[i].ColorG = float32(c.G) / 255
		vs[i].ColorB = float32(c.B) / 255
		vs[i].ColorA = float32(c.A) / 255
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = true
	dst.DrawTriangles(vs, is, whitePixel(), op)
}

// whitePixel returns a 1x1 white image for triangle drawing.
// This is cached to avoid repeated allocations.
var whitePixelImage *ebiten.Image

func whitePixel() *ebiten.Image {
	if whitePixelImage == nil {
		whitePixelImage = ebiten.NewImage(3, 3)
		whitePixelImage.Fill(color.White)
	}
	return whitePixelImage
}

// min32 returns the smaller of two float32 values.
func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// ClusterData represents cluster information for rendering.
type ClusterData struct {
	ID        string
	EchoIndex float64
	Category  resonance.EchoCategory
	CenterX   float32
	CenterY   float32
	NodeCount int
	TotalAmps int
}

// GetClusterData returns rendering data for all tracked clusters.
func (o *EchoIndexOverlay) GetClusterData() []ClusterData {
	if o.Computer == nil {
		return nil
	}

	indices := o.Computer.GetAllIndices()
	data := make([]ClusterData, 0, len(indices))

	for clusterID, idx := range indices {
		center := o.ClusterCenters[clusterID]
		data = append(data, ClusterData{
			ID:        clusterID,
			EchoIndex: idx.EchoIndex,
			Category:  idx.Category,
			CenterX:   center[0],
			CenterY:   center[1],
			TotalAmps: idx.TotalAmps,
		})
	}

	return data
}

// RenderLegend draws a legend explaining Echo Index colors.
func (o *EchoIndexOverlay) RenderLegend(dst *ebiten.Image, x, y float32) {
	// Legend background.
	bgColor := color.RGBA{30, 30, 30, 200}
	vector.DrawFilledRect(dst, x, y, 150, 80, bgColor, true)

	// Legend items.
	items := []struct {
		label   string
		r, g, b uint8
		yOff    float32
	}{
		{"Insular (>0.7)", 255, 100, 0, 15},
		{"Neutral", 128, 128, 128, 35},
		{"Open (<0.4)", 0, 200, 200, 55},
	}

	for _, item := range items {
		itemColor := color.RGBA{item.r, item.g, item.b, 255}
		vector.DrawFilledCircle(dst, x+15, y+item.yOff, 6, itemColor, true)
		// Note: Text rendering requires text/v2 package, omitted here.
	}
}
