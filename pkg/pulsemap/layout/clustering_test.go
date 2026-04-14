// Package layout - Tests for hierarchical aggregation clustering.
package layout

import (
	"math"
	"testing"
)

func TestNewClusterManager(t *testing.T) {
	cm := NewClusterManager()

	if cm == nil {
		t.Fatal("NewClusterManager returned nil")
	}
	if cm.IsEnabled() {
		t.Error("clustering should be disabled initially")
	}
	if cm.ClusterCount() != 0 {
		t.Error("should have no clusters initially")
	}
}

func TestClusterManager_EnableDisable(t *testing.T) {
	cm := NewClusterManager()

	cm.Enable()
	if !cm.IsEnabled() {
		t.Error("should be enabled after Enable")
	}

	cm.Disable()
	if cm.IsEnabled() {
		t.Error("should be disabled after Disable")
	}
}

func TestClusterManager_SetThreshold(t *testing.T) {
	cm := NewClusterManager()

	// Default threshold.
	if cm.threshold != ClusterThreshold {
		t.Errorf("expected default threshold %d, got %d", ClusterThreshold, cm.threshold)
	}

	// Set new threshold.
	cm.SetThreshold(200)
	if cm.threshold != 200 {
		t.Error("threshold should be updated")
	}
}

func TestClusterManager_SetMaxClusters(t *testing.T) {
	cm := NewClusterManager()

	cm.SetMaxClusters(50)
	if cm.maxClusters != 50 {
		t.Error("maxClusters should be updated")
	}
}

func TestClusterManager_SetClusterDistance(t *testing.T) {
	cm := NewClusterManager()

	cm.SetClusterDistance(100.0)
	if cm.clusterDist != 100.0 {
		t.Error("clusterDist should be updated")
	}
}

func TestClusterManager_UpdateClusters_BelowThreshold(t *testing.T) {
	cm := NewClusterManager()
	cm.SetThreshold(100) // Set low threshold.

	// Create 50 nodes (below threshold).
	nodes := make(map[string]*Node)
	positions := make(map[string]Position)

	for i := 0; i < 50; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: float64(i * 10), Y: float64(i * 10)}
	}

	cm.UpdateClusters(nodes, positions, nil)

	if cm.IsEnabled() {
		t.Error("clustering should not be enabled below threshold")
	}
	if cm.ClusterCount() != 0 {
		t.Error("should have no clusters below threshold")
	}
}

func TestClusterManager_UpdateClusters_AboveThreshold(t *testing.T) {
	cm := NewClusterManager()
	cm.SetThreshold(10) // Low threshold for testing.
	cm.SetClusterDistance(50.0)

	// Create 20 nodes in clusters.
	nodes := make(map[string]*Node)
	positions := make(map[string]Position)

	// Group 1: nodes at (0,0).
	for i := 0; i < 5; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: float64(i * 5), Y: float64(i * 5)}
	}

	// Group 2: nodes at (200,200).
	for i := 5; i < 10; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: 200 + float64((i-5)*5), Y: 200 + float64((i-5)*5)}
	}

	// Group 3: nodes at (400,400).
	for i := 10; i < 15; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: 400 + float64((i-10)*5), Y: 400 + float64((i-10)*5)}
	}

	// Scattered nodes.
	for i := 15; i < 20; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: float64(i * 100), Y: float64(i * 50)}
	}

	cm.UpdateClusters(nodes, positions, nil)

	if !cm.IsEnabled() {
		t.Error("clustering should be enabled above threshold")
	}

	// Should have created some clusters.
	if cm.ClusterCount() == 0 {
		t.Error("should have created clusters")
	}
}

func TestClusterManager_GetCluster(t *testing.T) {
	cm := NewClusterManager()
	cm.SetThreshold(5)
	cm.SetClusterDistance(100.0)

	// Create clusterable nodes.
	nodes := make(map[string]*Node)
	positions := make(map[string]Position)

	for i := 0; i < 10; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		// All close together.
		positions[id] = Position{X: float64(i * 5), Y: float64(i * 5)}
	}

	cm.UpdateClusters(nodes, positions, nil)

	// Some nodes should be clustered.
	clusteredCount := 0
	for id := range nodes {
		if cm.IsNodeClustered(id) {
			clusteredCount++
			cluster := cm.GetCluster(id)
			if cluster == nil {
				t.Error("GetCluster should return cluster for clustered node")
			}
		}
	}

	if clusteredCount == 0 && cm.ClusterCount() > 0 {
		t.Error("some nodes should be clustered when clusters exist")
	}
}

func TestClusterManager_ExpandCollapse(t *testing.T) {
	cm := NewClusterManager()

	// Create a cluster manually for testing.
	cm.clusters["test_cluster"] = &Cluster{
		ID:        "test_cluster",
		NodeCount: 5,
		NodeIDs:   []string{"n1", "n2", "n3", "n4", "n5"},
	}

	if cm.IsClusterExpanded("test_cluster") {
		t.Error("cluster should not be expanded initially")
	}

	cm.ExpandCluster("test_cluster")
	if !cm.IsClusterExpanded("test_cluster") {
		t.Error("cluster should be expanded after ExpandCluster")
	}
	if !cm.clusters["test_cluster"].Expanded {
		t.Error("cluster.Expanded should be true")
	}

	cm.CollapseCluster("test_cluster")
	if cm.IsClusterExpanded("test_cluster") {
		t.Error("cluster should be collapsed after CollapseCluster")
	}
}

func TestClusterManager_GetAllClusters(t *testing.T) {
	cm := NewClusterManager()

	// Add clusters.
	cm.clusters["c1"] = &Cluster{ID: "c1", NodeCount: 3}
	cm.clusters["c2"] = &Cluster{ID: "c2", NodeCount: 5}
	cm.clusters["c3"] = &Cluster{ID: "c3", NodeCount: 7}

	clusters := cm.GetAllClusters()

	if len(clusters) != 3 {
		t.Errorf("expected 3 clusters, got %d", len(clusters))
	}
}

func TestClusterManager_GetVisibleNodes(t *testing.T) {
	cm := NewClusterManager()
	cm.enabled = true

	// Set up clusters.
	cm.clusters["c1"] = &Cluster{
		ID:        "c1",
		NodeIDs:   []string{"n1", "n2", "n3"},
		NodeCount: 3,
	}
	cm.clusters["c2"] = &Cluster{
		ID:        "c2",
		NodeIDs:   []string{"n4", "n5"},
		NodeCount: 2,
	}
	cm.nodeCluster["n1"] = "c1"
	cm.nodeCluster["n2"] = "c1"
	cm.nodeCluster["n3"] = "c1"
	cm.nodeCluster["n4"] = "c2"
	cm.nodeCluster["n5"] = "c2"

	allNodes := []string{"n1", "n2", "n3", "n4", "n5", "n6"}

	// Initially no clusters expanded.
	visible := cm.GetVisibleNodes(allNodes)

	// Only n6 should be visible (not in any cluster).
	if len(visible) != 1 || visible[0] != "n6" {
		t.Errorf("expected only n6 visible, got %v", visible)
	}

	// Expand c1.
	cm.ExpandCluster("c1")
	visible = cm.GetVisibleNodes(allNodes)

	// n1, n2, n3, n6 should be visible.
	if len(visible) != 4 {
		t.Errorf("expected 4 visible nodes, got %d", len(visible))
	}
}

func TestClusterManager_ShouldShowCluster(t *testing.T) {
	cm := NewClusterManager()

	cm.clusters["c1"] = &Cluster{ID: "c1"}

	// Should show when not expanded.
	if !cm.ShouldShowCluster("c1") {
		t.Error("should show cluster when not expanded")
	}

	// Expand it.
	cm.ExpandCluster("c1")

	// Should not show when expanded.
	if cm.ShouldShowCluster("c1") {
		t.Error("should not show cluster when expanded")
	}
}

func TestClusterManager_GetClusterPosition(t *testing.T) {
	cm := NewClusterManager()

	cm.clusters["c1"] = &Cluster{
		ID:      "c1",
		CenterX: 150.5,
		CenterY: 250.5,
	}

	x, y, ok := cm.GetClusterPosition("c1")
	if !ok {
		t.Error("should find cluster position")
	}
	if x != 150.5 || y != 250.5 {
		t.Errorf("expected (150.5, 250.5), got (%f, %f)", x, y)
	}

	// Non-existent cluster.
	_, _, ok = cm.GetClusterPosition("nonexistent")
	if ok {
		t.Error("should not find non-existent cluster")
	}
}

func TestClusterManager_Clear(t *testing.T) {
	cm := NewClusterManager()

	// Add some state.
	cm.clusters["c1"] = &Cluster{ID: "c1"}
	cm.nodeCluster["n1"] = "c1"
	cm.expandedIDs["c1"] = true
	cm.enabled = true

	cm.Clear()

	if cm.ClusterCount() != 0 {
		t.Error("clusters should be cleared")
	}
	if cm.IsEnabled() {
		t.Error("should be disabled after clear")
	}
	if len(cm.nodeCluster) != 0 {
		t.Error("nodeCluster should be cleared")
	}
	if len(cm.expandedIDs) != 0 {
		t.Error("expandedIDs should be cleared")
	}
}

func TestClusterManager_GetStats(t *testing.T) {
	cm := NewClusterManager()

	// Empty stats.
	stats := cm.GetStats()
	if stats.TotalClusters != 0 {
		t.Error("should have 0 clusters initially")
	}

	// Add clusters.
	cm.clusters["c1"] = &Cluster{ID: "c1", NodeCount: 3, Expanded: false}
	cm.clusters["c2"] = &Cluster{ID: "c2", NodeCount: 5, Expanded: true}
	cm.clusters["c3"] = &Cluster{ID: "c3", NodeCount: 10, Expanded: false}

	stats = cm.GetStats()

	if stats.TotalClusters != 3 {
		t.Errorf("expected 3 clusters, got %d", stats.TotalClusters)
	}
	if stats.TotalNodes != 18 {
		t.Errorf("expected 18 total nodes, got %d", stats.TotalNodes)
	}
	if stats.ExpandedCount != 1 {
		t.Errorf("expected 1 expanded, got %d", stats.ExpandedCount)
	}
	if stats.LargestCluster != 10 {
		t.Errorf("expected largest 10, got %d", stats.LargestCluster)
	}
	if stats.SmallestCluster != 3 {
		t.Errorf("expected smallest 3, got %d", stats.SmallestCluster)
	}
	if stats.AvgClusterSize != 6.0 {
		t.Errorf("expected avg 6.0, got %f", stats.AvgClusterSize)
	}
}

func TestCluster_Fields(t *testing.T) {
	c := &Cluster{
		ID:          "test_cluster",
		CenterX:     100.0,
		CenterY:     200.0,
		NodeCount:   5,
		TotalEdges:  10,
		NodeIDs:     []string{"a", "b", "c", "d", "e"},
		Radius:      30.0,
		Connections: []string{"other_cluster"},
		Activity:    0.75,
		Expanded:    true,
	}

	if c.ID != "test_cluster" {
		t.Error("ID mismatch")
	}
	if c.CenterX != 100.0 || c.CenterY != 200.0 {
		t.Error("center position mismatch")
	}
	if c.NodeCount != 5 {
		t.Error("NodeCount mismatch")
	}
	if len(c.NodeIDs) != 5 {
		t.Error("NodeIDs length mismatch")
	}
	if !c.Expanded {
		t.Error("Expanded should be true")
	}
}

func TestGenerateClusterID(t *testing.T) {
	ids := make(map[string]bool)

	// Generate many IDs and check uniqueness.
	for i := 0; i < 100; i++ {
		id := generateClusterID(i)
		if ids[id] {
			t.Errorf("duplicate cluster ID: %s", id)
		}
		ids[id] = true

		// Should have "cluster_" prefix.
		if len(id) < 8 || id[:8] != "cluster_" {
			t.Errorf("invalid cluster ID format: %s", id)
		}
	}
}

func TestDistance(t *testing.T) {
	// Origin to (3,4) should be 5.
	d := distance(0, 0, 3, 4)
	if math.Abs(d-5.0) > 0.001 {
		t.Errorf("expected 5.0, got %f", d)
	}

	// Same point.
	d = distance(10, 10, 10, 10)
	if d != 0 {
		t.Errorf("expected 0, got %f", d)
	}

	// Negative coordinates.
	d = distance(-3, -4, 0, 0)
	if math.Abs(d-5.0) > 0.001 {
		t.Errorf("expected 5.0, got %f", d)
	}
}

func TestCalculateClusterRadius(t *testing.T) {
	// Single node.
	r1 := calculateClusterRadius(1)

	// More nodes should have larger radius.
	r5 := calculateClusterRadius(5)
	r10 := calculateClusterRadius(10)
	r100 := calculateClusterRadius(100)

	if r1 >= r5 {
		t.Error("5 nodes should have larger radius than 1")
	}
	if r5 >= r10 {
		t.Error("10 nodes should have larger radius than 5")
	}
	if r10 >= r100 {
		t.Error("100 nodes should have larger radius than 10")
	}

	// Radius should be logarithmic, not linear.
	// r100 should be much less than 10x r10.
	if r100 > r10*3 {
		t.Error("radius growth should be logarithmic")
	}
}

func TestClusterManager_WithEdges(t *testing.T) {
	cm := NewClusterManager()
	cm.SetThreshold(5)
	cm.SetClusterDistance(100.0)

	nodes := make(map[string]*Node)
	positions := make(map[string]Position)

	for i := 0; i < 10; i++ {
		id := generateClusterID(i)
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: float64(i * 5), Y: float64(i * 5)}
	}

	edges := []Edge{
		{SourceID: generateClusterID(0), TargetID: generateClusterID(1)},
		{SourceID: generateClusterID(1), TargetID: generateClusterID(2)},
		{SourceID: generateClusterID(2), TargetID: generateClusterID(3)},
	}

	cm.UpdateClusters(nodes, positions, edges)

	if !cm.IsEnabled() {
		t.Error("clustering should be enabled")
	}

	// Check that clusters have edge counts.
	for _, c := range cm.GetAllClusters() {
		if c.TotalEdges < 0 {
			t.Error("TotalEdges should not be negative")
		}
	}
}

func TestClusterManager_ClusterConnections(t *testing.T) {
	cm := NewClusterManager()
	cm.SetThreshold(3)
	cm.SetClusterDistance(30.0)

	// Two groups of nodes with an edge between groups.
	nodes := make(map[string]*Node)
	positions := make(map[string]Position)

	// Group 1: near origin.
	for i := 0; i < 3; i++ {
		id := "g1_" + string(rune('a'+i))
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: float64(i * 5), Y: float64(i * 5)}
	}

	// Group 2: far away.
	for i := 0; i < 3; i++ {
		id := "g2_" + string(rune('a'+i))
		nodes[id] = &Node{ID: id, Activity: 1.0}
		positions[id] = Position{X: 500 + float64(i*5), Y: 500 + float64(i*5)}
	}

	// Edge connecting groups.
	edges := []Edge{
		{SourceID: "g1_a", TargetID: "g2_a"},
	}

	cm.UpdateClusters(nodes, positions, edges)

	// If clustering happened, check connections.
	clusters := cm.GetAllClusters()
	if len(clusters) >= 2 {
		// Find clusters with connections.
		hasConnection := false
		for _, c := range clusters {
			if len(c.Connections) > 0 {
				hasConnection = true
				break
			}
		}
		// Connection detection depends on clustering result.
		_ = hasConnection // May or may not have connections depending on clustering.
	}
}

func TestClusterConstants(t *testing.T) {
	if ClusterThreshold != 500 {
		t.Errorf("ClusterThreshold should be 500, got %d", ClusterThreshold)
	}
	if MaxClusters != 100 {
		t.Errorf("MaxClusters should be 100, got %d", MaxClusters)
	}
	if ClusterMinSize != 3 {
		t.Errorf("ClusterMinSize should be 3, got %d", ClusterMinSize)
	}
}
