# Pulse Map Degradation Curve and Scalability Strategy

**Version:** 1.0  
**Date:** 2026-05-06  
**Status:** Complete Specification  
**Purpose:** Define performance and visual behavior across network scales

---

## Executive Summary

Per PLAN.md §1.5, this document specifies the Pulse Map's "degradation curve" — how rendering quality, update frequency, and visual fidelity gracefully degrade as visible node count increases from 50 to 50,000. The goal is **60fps at 500 nodes** (per TECHNICAL_IMPLEMENTATION.md §5.1) with acceptable degradation beyond that threshold.

**Core Philosophy:** Preserve user experience (smooth interaction, readable information) over visual perfection. Degrade algorithmic accuracy before frame rate.

---

## Performance Targets

| Node Count | Target FPS | Layout Algorithm | Visual Fidelity | Interaction Latency |
|------------|-----------|-----------------|----------------|-------------------|
| 1–50       | 60fps     | Fruchterman-Reingold (exact) | Full (glow, ripple, spectra) | < 16ms |
| 51–500     | 60fps     | Fruchterman-Reingold (exact) | Full | < 16ms |
| 501–2,000  | 60fps     | Barnes-Hut (θ=0.5) | Reduced (no ripple on distant nodes) | < 16ms |
| 2,001–10,000 | 45–60fps | Barnes-Hut (θ=0.8) | Minimal (sigils hidden >3 hops) | < 32ms |
| 10,001–50,000 | 30–45fps | Clustering (meta-nodes) | Aggregate (clusters shown as single node) | < 50ms |
| 50,000+    | 30fps (fallback) | Static layout (no force-directed) | Statistical (heatmap only) | < 100ms |

**Rationale:**
- 60fps is perceptually smooth; 45fps is acceptable; 30fps is minimum tolerable
- User interaction (pan, zoom, tap) must never lag (always < 100ms response)
- Visual fidelity can degrade without UX penalty — users adjust expectations at scale

---

## Scalability Thresholds

### Threshold 1: Small Network (1–50 nodes)
**Characteristics:**
- Every node fully rendered (sigil, glow, edges, labels)
- Force-directed layout runs every frame (Fruchterman-Reingold, exact)
- All visual effects active (ripple, glow, spectra, particle trails)
- Edge labels visible on hover (shows connection type, timestamp)
- Hover tooltips show full node details (display name, online status, Resonance rank)

**Performance:**
- CPU: ~5% (single core, ~200 MHz on 4 GHz CPU)
- GPU: ~10% (integrated GPU, minimal shader work)
- Memory: ~2 MB (node positions, edge list, metadata)
- Frame time: ~8ms (120fps headroom)

**Implementation:**
```go
// pkg/pulsemap/layout/engine.go
func (e *Engine) updateSmallNetwork() {
    // Full Fruchterman-Reingold, no optimizations.
    for i := 0; i < len(e.nodes); i++ {
        repulsion := e.calculateRepulsionForces(i)
        attraction := e.calculateAttractionForces(i)
        e.nodes[i].velocity = repulsion + attraction
    }
    e.applyVelocities()
}
```

### Threshold 2: Medium Network (51–500 nodes)
**Characteristics:**
- All nodes rendered, but distant nodes use simplified geometry (circles instead of sigils beyond 2 hops)
- Force-directed layout runs every frame (Fruchterman-Reingold, exact)
- Visual effects limited to viewport + 10% margin (ripples only on visible nodes)
- Edge labels visible on hover (no change from small network)
- Hover tooltips remain full-detail

**Performance:**
- CPU: ~15% (layout dominates, O(n²) repulsion calculation)
- GPU: ~25% (shader instancing for simplified nodes)
- Memory: ~20 MB (node positions, edge list, visual state)
- Frame time: ~14ms (70fps average)

**Optimization:**
- Viewport culling: Only render nodes within camera bounds + margin
- Shader instancing: Batch draw calls for simplified nodes (single draw for all distant nodes)
- Level of Detail (LOD): Distant nodes use 8×8 sigil texture instead of 64×64

**Implementation:**
```go
// pkg/pulsemap/rendering/renderer.go
func (r *Renderer) drawMediumNetwork(screen *ebiten.Image) {
    visible := r.layout.GetVisibleNodes(r.camera.Bounds())
    
    // Draw distant nodes as simple circles (instanced).
    distantNodes := r.filterDistantNodes(visible, 2) // >2 hops from center
    r.drawSimplifiedNodes(screen, distantNodes)
    
    // Draw nearby nodes with full fidelity.
    nearbyNodes := r.filterNearbyNodes(visible, 2)
    for _, node := range nearbyNodes {
        r.drawFullNode(screen, node) // Sigil, glow, label
    }
    
    // Draw edges (all visible, but thin lines for distant edges).
    r.drawEdges(screen, visible)
}
```

### Threshold 3: Large Network (501–2,000 nodes)
**Characteristics:**
- Barnes-Hut approximation for force calculation (θ=0.5, 10% error vs exact)
- Distant nodes (>3 hops) hidden entirely (declutters viewport)
- Ripple effects disabled (too expensive at scale)
- Edge labels on hover only for edges between nearby nodes
- Hover tooltips load async (fetch from storage on demand, not preloaded)

**Performance:**
- CPU: ~30% (Barnes-Hut reduces O(n²) → O(n log n))
- GPU: ~40% (still rendering 500–1,000 visible nodes)
- Memory: ~100 MB (quadtree for Barnes-Hut, node cache)
- Frame time: ~16ms (60fps maintained)

**Optimization:**
- Barnes-Hut quadtree: Group distant nodes for approximate repulsion
- Aggressive viewport culling: Only render nodes within strict camera bounds (no margin)
- Edge bundling: Group parallel edges (multiple connections between same two nodes) into single thick edge

**Implementation:**
```go
// pkg/pulsemap/layout/engine.go
func (e *Engine) updateLargeNetwork() {
    // Build Barnes-Hut quadtree.
    tree := e.buildQuadtree(e.nodes)
    
    // Calculate forces using quadtree approximation.
    for i := 0; i < len(e.nodes); i++ {
        repulsion := tree.approximateRepulsion(e.nodes[i], 0.5) // θ=0.5
        attraction := e.calculateAttractionForces(i) // Exact (only connected nodes)
        e.nodes[i].velocity = repulsion + attraction
    }
    e.applyVelocities()
}

// pkg/pulsemap/layout/viewport_culling.go
func (c *Culler) GetVisibleNodes(cameraBounds image.Rectangle) []NodeID {
    // Strict culling: no margin, only nodes inside camera viewport.
    visible := []NodeID{}
    for _, node := range c.allNodes {
        if cameraBounds.Contains(node.Position) {
            visible = append(visible, node.ID)
        }
    }
    return visible
}
```

### Threshold 4: Very Large Network (2,001–10,000 nodes)
**Characteristics:**
- Barnes-Hut with looser approximation (θ=0.8, 20% error vs exact — acceptable trade-off)
- Sigils hidden beyond 1 hop (only direct connections show sigils)
- Labels hidden entirely (only shown in node detail panel, not on map)
- No glow effects (solid circles with flat colors)
- Frame rate drops to 45fps (still acceptable)

**Performance:**
- CPU: ~50% (layout + culling overhead)
- GPU: ~60% (many draw calls despite instancing)
- Memory: ~400 MB (large quadtree, edge cache)
- Frame time: ~22ms (45fps average)

**Optimization:**
- Throttled layout: Run force-directed algorithm every 2 frames (30 updates/sec instead of 60)
- Batch culling: Cull nodes in chunks (spatial hash grid, only test grid cells intersecting viewport)
- Simplified edges: All edges rendered as thin lines (no gradients, no glow)

**Implementation:**
```go
// pkg/pulsemap/layout/throttle.go
type ThrottledEngine struct {
    inner      *Engine
    frameCount int
}

func (t *ThrottledEngine) Update() {
    t.frameCount++
    if t.frameCount%2 == 0 {
        t.inner.Update() // Run layout every 2 frames
    }
}

// pkg/pulsemap/rendering/renderer.go
func (r *Renderer) drawVeryLargeNetwork(screen *ebiten.Image) {
    visible := r.layout.GetVisibleNodes(r.camera.Bounds())
    
    // Draw all nodes as simple colored circles (no sigils).
    for _, node := range visible {
        r.drawCircle(screen, node.Position, node.Color, 8) // 8px radius
    }
    
    // Draw edges as thin lines (no shader effects).
    for _, edge := range r.getVisibleEdges(visible) {
        r.drawLine(screen, edge.From, edge.To, edge.Color, 1) // 1px width
    }
}
```

### Threshold 5: Massive Network (10,001–50,000 nodes)
**Characteristics:**
- Clustering: Distant nodes grouped into "meta-nodes" (aggregate representation)
- Static layout: Force-directed disabled, use precomputed layout (e.g., community detection)
- Heatmap overlay: Replace node-by-node rendering with density heatmap
- Frame rate drops to 30fps (minimum acceptable)
- Interaction latency increases to ~50ms (still feels responsive)

**Performance:**
- CPU: ~70% (clustering algorithm + heatmap computation)
- GPU: ~80% (heatmap shader, cluster rendering)
- Memory: ~1 GB (full node list + cluster metadata + heatmap texture)
- Frame time: ~33ms (30fps target)

**Optimization:**
- Clustering algorithm: Louvain method (community detection) groups nodes into 100–200 clusters
- Cluster rendering: Each cluster is a single node with size proportional to member count
- Precomputed layout: Run force-directed once offline, store positions in Bbolt (no runtime layout)
- Heatmap shader: GPU-accelerated 2D density estimation (Gaussian blur over node positions)

**Implementation:**
```go
// pkg/pulsemap/layout/clustering.go
type ClusterEngine struct {
    clusters []Cluster
}

type Cluster struct {
    ID       int
    Members  []NodeID
    Position image.Point
    Radius   int // Size proportional to sqrt(member count)
}

func (c *ClusterEngine) BuildClusters(nodes []Node) {
    // Run Louvain community detection.
    communities := louvain.Detect(nodes) // External library
    
    for _, community := range communities {
        cluster := Cluster{
            ID:      community.ID,
            Members: community.NodeIDs,
        }
        
        // Calculate cluster center (average position of members).
        cluster.Position = c.calculateCentroid(community.NodeIDs)
        
        // Calculate cluster radius.
        cluster.Radius = int(math.Sqrt(float64(len(community.NodeIDs)))) * 10
        
        c.clusters = append(c.clusters, cluster)
    }
}

// pkg/pulsemap/rendering/renderer.go
func (r *Renderer) drawMassiveNetwork(screen *ebiten.Image) {
    // Draw heatmap overlay (density estimation).
    r.drawHeatmap(screen)
    
    // Draw clusters as large circles with member count labels.
    for _, cluster := range r.layout.GetClusters() {
        if r.camera.Bounds().Overlaps(cluster.BoundingBox()) {
            r.drawCluster(screen, cluster)
        }
    }
}

func (r *Renderer) drawCluster(screen *ebiten.Image, cluster Cluster) {
    // Draw circle.
    r.drawCircle(screen, cluster.Position, color.RGBA{100, 150, 255, 200}, cluster.Radius)
    
    // Draw member count label.
    label := fmt.Sprintf("%d nodes", len(cluster.Members))
    r.drawText(screen, label, cluster.Position, 16)
}
```

### Threshold 6: Extreme Network (50,000+ nodes)
**Characteristics:**
- Statistical view only: Heatmap + aggregate metrics, no individual nodes visible
- Static visualization: No interactivity (pan/zoom disabled or heavily throttled)
- Fallback to list view: Offer alternative UI (searchable node list, not spatial graph)
- Frame rate 30fps (barely maintained)
- User warned: "Your network is very large. Consider filtering or using search."

**Performance:**
- CPU: ~90% (heatmap computation dominates)
- GPU: ~90% (large texture upload for heatmap)
- Memory: ~2 GB (entire node list in memory)
- Frame time: ~40ms (25fps actual, stutters likely)

**Optimization:**
- Heatmap-only mode: Disable node rendering, show density heatmap exclusively
- Spatial indexing: R-tree for fast node lookup (supports search/filter)
- Pagination: Only load visible region of network (streaming model, not all-in-memory)

**Implementation:**
```go
// pkg/pulsemap/rendering/renderer.go
func (r *Renderer) drawExtremeNetwork(screen *ebiten.Image) {
    // Show warning overlay.
    r.drawWarningOverlay(screen, "Network too large for visualization. Use search or filters.")
    
    // Render heatmap only.
    r.drawHeatmap(screen)
    
    // Offer alternative: "Switch to List View" button.
    r.drawAlternativeViewButton(screen)
}

// Alternative view: pkg/ui/node_list.go
type NodeListView struct {
    nodes       []Node
    searchQuery string
    filterHops  int // Show only nodes within N hops
}

func (v *NodeListView) Draw(screen *ebiten.Image) {
    // Render scrollable list of nodes (similar to contact list UI).
    // Each row: sigil (small), display name, online status, Resonance rank.
    // Tap row → open node detail panel.
}
```

---

## Level of Detail (LOD) System

### LOD 0: Full Fidelity (0–1 hops from center node)
- **Sigil:** 64×64 full-resolution texture
- **Glow:** Full shader effect (gaussian blur, 5-pass)
- **Label:** Display name (full UTF-8, up to 64 characters)
- **Edge:** Gradient shader (warm → cool), animated pulse
- **Effects:** Ripple on Wave receipt, spectra on Specter interaction

### LOD 1: High Detail (2–3 hops)
- **Sigil:** 32×32 reduced texture (still recognizable)
- **Glow:** Simplified shader (gaussian blur, 3-pass)
- **Label:** Display name (truncated to 32 characters)
- **Edge:** Solid color (no gradient), no animation
- **Effects:** No ripple, no spectra (glow only)

### LOD 2: Medium Detail (4–5 hops)
- **Sigil:** 16×16 thumbnail (recognizable only when zoomed)
- **Glow:** Single-pass glow (circle blur, not gaussian)
- **Label:** Hidden (only shown on hover)
- **Edge:** Thin line (1px, solid color)
- **Effects:** None (static rendering)

### LOD 3: Low Detail (6+ hops)
- **Sigil:** Hidden (colored circle only, derived from public key hash)
- **Glow:** None (flat circle)
- **Label:** Hidden (even on hover — use node detail panel)
- **Edge:** Hidden (only shown if both endpoints are < 6 hops)
- **Effects:** None

### LOD 4: Clustering (distant nodes, >2,000 total nodes)
- **Node:** Meta-node (aggregate of 10–100 nodes)
- **Visual:** Large circle, size proportional to member count
- **Label:** "N nodes" (member count)
- **Interaction:** Tap to expand (zoom in, reveal individual nodes)
- **Edge:** Cluster-to-cluster edges (aggregate connection strength)

**Implementation:**
```go
// pkg/pulsemap/rendering/lod.go
type LODLevel int

const (
    LODFull LODLevel = iota
    LODHigh
    LODMedium
    LODLow
    LODClustered
)

func (r *Renderer) calculateLOD(node Node, centerNode Node) LODLevel {
    hops := r.layout.GetShortestPathLength(node.ID, centerNode.ID)
    totalNodes := len(r.layout.GetAllNodes())
    
    switch {
    case totalNodes > 10000:
        return LODClustered
    case hops <= 1:
        return LODFull
    case hops <= 3:
        return LODHigh
    case hops <= 5:
        return LODMedium
    default:
        return LODLow
    }
}

func (r *Renderer) drawNodeWithLOD(screen *ebiten.Image, node Node, lod LODLevel) {
    switch lod {
    case LODFull:
        r.drawFullNode(screen, node)
    case LODHigh:
        r.drawHighDetailNode(screen, node)
    case LODMedium:
        r.drawMediumDetailNode(screen, node)
    case LODLow:
        r.drawLowDetailNode(screen, node)
    case LODClustered:
        // Handled by clustering engine, not per-node rendering.
    }
}
```

---

## Culling Strategies

### 1. Frustum Culling (Viewport)
**Purpose:** Eliminate nodes outside camera bounds (not visible)  
**Threshold:** All scales (always active)  
**Method:** Axis-aligned bounding box (AABB) test  
**Savings:** 50–90% of nodes culled at typical zoom levels

```go
func (c *Culler) frustumCull(nodes []Node, cameraBounds image.Rectangle) []Node {
    visible := []Node{}
    for _, node := range nodes {
        if cameraBounds.Contains(node.Position) {
            visible = append(visible, node)
        }
    }
    return visible
}
```

### 2. Distance Culling (Hop Limit)
**Purpose:** Hide nodes beyond N hops from center (reduce clutter)  
**Threshold:** 501+ nodes → cull beyond 5 hops, 2,001+ → cull beyond 3 hops  
**Method:** Breadth-first search (BFS) from center node, mark reachable nodes  
**Savings:** 70–90% of nodes culled in large networks

```go
func (c *Culler) distanceCull(nodes []Node, centerNode Node, maxHops int) []Node {
    reachable := c.bfsReachable(centerNode, maxHops) // BFS, early termination at maxHops
    visible := []Node{}
    for _, node := range nodes {
        if reachable[node.ID] {
            visible = append(visible, node)
        }
    }
    return visible
}
```

### 3. Occlusion Culling (Overlapping Nodes)
**Purpose:** Hide nodes obscured by other nodes (layering)  
**Threshold:** 2,001+ nodes (expensive, only use at scale)  
**Method:** Z-buffer (sort nodes by depth, skip rendering if fully obscured)  
**Savings:** 20–40% of nodes culled in dense clusters

```go
func (c *Culler) occlusionCull(nodes []Node) []Node {
    // Sort nodes by Z-order (depth, closer nodes first).
    sort.Slice(nodes, func(i, j int) bool {
        return nodes[i].Depth < nodes[j].Depth
    })
    
    visible := []Node{}
    occupied := make(map[image.Point]bool) // Spatial hash of occupied pixels
    
    for _, node := range nodes {
        if !c.isOccluded(node, occupied) {
            visible = append(visible, node)
            c.markOccupied(node, occupied)
        }
    }
    return visible
}
```

### 4. Temporal Coherence (Frame-to-Frame Reuse)
**Purpose:** Reuse culling results from previous frame (avoid redundant computation)  
**Threshold:** All scales (always active)  
**Method:** Cache visible node list, only recompute on camera move or node update  
**Savings:** 50–90% reduction in culling CPU time

```go
type CoherentCuller struct {
    lastCameraPos   image.Point
    lastVisibleNodes []Node
    dirty           bool
}

func (c *CoherentCuller) cull(nodes []Node, camera Camera) []Node {
    // Check if camera moved significantly (>10px).
    if camera.Position.Sub(c.lastCameraPos).Abs() < 10 && !c.dirty {
        return c.lastVisibleNodes // Reuse cached result
    }
    
    // Camera moved — recompute.
    c.lastVisibleNodes = c.recomputeCulling(nodes, camera)
    c.lastCameraPos = camera.Position
    c.dirty = false
    return c.lastVisibleNodes
}
```

---

## Edge Bundling

**Purpose:** Reduce visual clutter when many edges connect same two regions  
**Threshold:** 501+ nodes  
**Method:** Group parallel edges (same source/destination or nearby endpoints) into single thick edge

**Visual Effect:**
- Before bundling: 100 edges between two clusters → spaghetti mess
- After bundling: 1 thick edge (width proportional to edge count) → clean, readable

**Implementation:**
```go
// pkg/pulsemap/rendering/edge_bundling.go
type EdgeBundle struct {
    From     image.Point
    To       image.Point
    Count    int     // Number of edges in bundle
    Width    int     // Render width (proportional to count)
}

func (b *Bundler) bundleEdges(edges []Edge) []EdgeBundle {
    bundles := make(map[string]*EdgeBundle)
    
    for _, edge := range edges {
        // Create bundle key (from → to, rounded to 50px grid).
        key := b.bundleKey(edge.From, edge.To, 50)
        
        if bundle, exists := bundles[key]; exists {
            bundle.Count++
        } else {
            bundles[key] = &EdgeBundle{
                From:  edge.From,
                To:    edge.To,
                Count: 1,
            }
        }
    }
    
    // Convert map to slice, calculate widths.
    result := []EdgeBundle{}
    for _, bundle := range bundles {
        bundle.Width = int(math.Log2(float64(bundle.Count)) * 2) // Logarithmic scaling
        result = append(result, *bundle)
    }
    return result
}
```

---

## Heatmap Rendering

**Purpose:** Visualize network density when individual nodes are too numerous to render  
**Threshold:** 10,001+ nodes  
**Method:** Gaussian kernel density estimation (KDE) on GPU

**Visual Effect:**
- Bright regions: High node density (many connections, active area)
- Dark regions: Low density (sparse, peripheral nodes)
- Color gradient: Cool (blue) → Warm (orange) → Hot (red) based on density

**Implementation:**
```go
// pkg/pulsemap/rendering/heatmap.go
type Heatmap struct {
    densityTexture *ebiten.Image // 512×512 texture
    shader         *ebiten.Shader // Gaussian KDE shader
}

func (h *Heatmap) Update(nodes []Node) {
    // Clear density texture.
    h.densityTexture.Clear()
    
    // Render each node as a splat (Gaussian kernel).
    for _, node := range nodes {
        h.renderKernel(node.Position, 50.0) // 50px kernel radius
    }
    
    // Apply Gaussian blur (multiple passes).
    h.applyGaussianBlur(3) // 3 passes
}

func (h *Heatmap) Draw(screen *ebiten.Image) {
    // Render density texture with color gradient.
    op := &ebiten.DrawImageOptions{}
    op.ColorScale.Scale(1.0, 0.5, 0.2, 0.8) // Orange-red gradient
    screen.DrawImage(h.densityTexture, op)
}

// Kage shader: pkg/pulsemap/shaders/heatmap.kage
package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
    // Sample density texture.
    density := texture2D(Texture0, texCoord).r
    
    // Map density to color gradient (blue → orange → red).
    var c vec3
    if density < 0.5 {
        c = mix(vec3(0.0, 0.2, 0.5), vec3(1.0, 0.5, 0.0), density * 2.0)
    } else {
        c = mix(vec3(1.0, 0.5, 0.0), vec3(1.0, 0.0, 0.0), (density - 0.5) * 2.0)
    }
    
    return vec4(c, density * 0.8) // Alpha based on density
}
```

---

## User Communication

### Threshold Warnings

When network size crosses a threshold, show non-blocking notification:

| Threshold | Message | Suggested Action |
|-----------|---------|------------------|
| 501 nodes | "Your network is large. Performance optimizations active." | "Consider filtering connections in Settings" |
| 2,001 nodes | "Your network is very large. Detailed view limited to nearby nodes." | "Use Search to find specific nodes" |
| 10,001 nodes | "Your network is massive. Visualization simplified." | "Switch to List View or apply filters" |
| 50,000 nodes | "Your network exceeds visualization limits. Using statistical view." | "Use Search exclusively" |

### Performance Settings

Expose user controls in Settings → Advanced → Pulse Map:

- **Node Limit:** Max visible nodes (default auto, manual override: 100, 500, 1000, 2000, unlimited)
- **Hop Limit:** Max hops from center node (default auto, manual: 2, 3, 5, 10, unlimited)
- **Visual Quality:** High (all effects), Medium (no ripple), Low (no effects), Minimal (circles only)
- **Layout Frequency:** Real-time (60fps), Throttled (30fps), Static (precomputed, no updates)

---

## Testing Strategy

### Performance Tests

1. **Benchmark Suite:** `pkg/pulsemap/layout/benchmark_test.go`
   - Test cases: 50, 100, 500, 1000, 2000, 5000, 10000 nodes
   - Metrics: Frame time, CPU%, GPU%, memory
   - Pass criteria: 60fps @ 500 nodes, 45fps @ 2000 nodes, 30fps @ 10000 nodes

2. **Stress Test:** `pkg/pulsemap/layout/stress_test.go`
   - Generate synthetic network (50,000 nodes, random connections)
   - Run Pulse Map for 60 seconds, log frame time distribution
   - Pass criteria: No crashes, no frame times >100ms (10fps minimum)

3. **Regression Tests:** `pkg/pulsemap/rendering/visual_regression_test.go`
   - Render Pulse Map at each threshold, compare screenshots vs baseline
   - Pass criteria: <5% pixel difference (allows anti-aliasing variance)

### User Testing

1. **Scenario 1:** User with 50 connections
   - Expected: Smooth experience, full visual fidelity
   - Test: Ask user to pan, zoom, tap nodes — subjective rating (1-5) "smoothness"

2. **Scenario 2:** User with 500 connections
   - Expected: Smooth experience, slight visual degradation (no ripple on distant nodes)
   - Test: Measure time-to-tap (latency), confirm <16ms

3. **Scenario 3:** User with 2,000 connections
   - Expected: Acceptable experience (45fps), noticeable degradation (no sigils on distant nodes)
   - Test: Show side-by-side comparison vs 500-node network, ask "Is this usable?" (Y/N)

4. **Scenario 4:** User with 10,000 connections
   - Expected: Degraded experience (30fps), clustering visible
   - Test: Offer List View alternative, ask "Do you prefer List or Map?" (choice)

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Frame Rate @ 500 nodes | ≥60fps | `ebiten.TPS()` |
| Frame Rate @ 2,000 nodes | ≥45fps | `ebiten.TPS()` |
| Frame Rate @ 10,000 nodes | ≥30fps | `ebiten.TPS()` |
| Interaction Latency | <16ms @ all scales | Time from tap to node detail panel open |
| Memory Usage @ 10,000 nodes | <1 GB | `runtime.MemStats.Alloc` |
| User Satisfaction (500 nodes) | ≥90% "smooth" rating | Post-session survey |
| User Satisfaction (2,000 nodes) | ≥70% "acceptable" rating | Post-session survey |
| Alternative View Adoption (10,000+ nodes) | ≥50% prefer List View | A/B testing |

---

## Open Questions

1. **Clustering Algorithm:** Louvain method is expensive (O(n log n)) — use faster heuristic (e.g., grid-based)?
   - **Recommendation:** Prototype both, benchmark, choose based on data

2. **Heatmap Color Scheme:** Cool-to-warm gradient vs monochrome?
   - **Recommendation:** A/B test with 20 users, select based on preference

3. **Static Layout Precomputation:** How often to regenerate (daily, weekly, on-demand)?
   - **Recommendation:** Daily for active users, on-demand for <1 session/week

4. **50,000+ Node Fallback:** Should we block Pulse Map entirely and force List View?
   - **Recommendation:** No — offer degraded experience, let user choose

---

## Implementation Checklist

### Phase 1: Foundation (Week 1–2)
- [ ] Implement `pkg/pulsemap/layout/throttle.go` (throttled layout engine)
- [ ] Implement `pkg/pulsemap/rendering/lod.go` (LOD level calculation)
- [ ] Implement `pkg/pulsemap/layout/viewport_culling.go` (frustum + distance culling)
- [ ] Add performance metrics logging (`frame_time`, `visible_nodes`, `total_nodes`)

### Phase 2: Optimization (Week 3–4)
- [ ] Implement Barnes-Hut quadtree in `pkg/pulsemap/layout/engine.go`
- [ ] Implement edge bundling in `pkg/pulsemap/rendering/edge_bundling.go`
- [ ] Add occlusion culling (optional, if needed for 2,000+ nodes)
- [ ] Add temporal coherence caching to reduce redundant culling

### Phase 3: Clustering (Week 5–6)
- [ ] Implement Louvain community detection (or faster heuristic)
- [ ] Implement cluster rendering in `pkg/pulsemap/rendering/renderer.go`
- [ ] Add cluster expansion interaction (tap cluster → zoom in, reveal members)
- [ ] Test clustering with 10,000-node synthetic network

### Phase 4: Heatmap (Week 7–8)
- [ ] Implement heatmap shader in `pkg/pulsemap/shaders/heatmap.kage`
- [ ] Implement GPU-based Gaussian KDE
- [ ] Add heatmap toggle in UI (Settings → Pulse Map → "Show Heatmap")
- [ ] Test heatmap with 50,000-node synthetic network

### Phase 5: User Warnings & Settings (Week 9)
- [ ] Implement threshold warning notifications
- [ ] Add performance settings in Settings → Advanced → Pulse Map
- [ ] Add "Switch to List View" button in extreme network state
- [ ] Document user controls in help/tutorial

### Phase 6: Testing & Tuning (Week 10)
- [ ] Run benchmark suite, record baseline metrics
- [ ] Conduct user testing (4 scenarios, N=20 users)
- [ ] Analyze telemetry, identify bottlenecks
- [ ] Tune thresholds (LOD distances, culling radii, cluster sizes)
- [ ] Document final performance characteristics in `docs/PULSE_MAP_PERFORMANCE.md`

---

**Document Status:** Complete specification, ready for implementation  
**Dependencies:** PLAN.md §1.5, TECHNICAL_IMPLEMENTATION.md §5.1, PULSE_MAP.md  
**Next Steps:** Implement throttling and LOD system (Week 1–2, estimate 2 weeks, 1 engineer)
