// Package layout - Hierarchical aggregation for large graphs.
// Per ROADMAP.md line 591: "Hierarchical aggregation — cluster representatives
// for >500 visible nodes".
// Per PULSE_MAP.md: When node count exceeds threshold, nearby nodes are
// clustered into representative supernodes to maintain rendering performance.
package layout

import (
	"math"
	"sort"
	"sync"
)

// ClusterThreshold is the visible node count that triggers clustering.
const ClusterThreshold = 500

// MaxClusters is the maximum number of clusters to display.
const MaxClusters = 100

// ClusterMinSize is the minimum nodes in a cluster to be displayed.
const ClusterMinSize = 3

// Cluster represents a group of nodes aggregated into a single representative.
type Cluster struct {
	ID          string   // Unique cluster identifier.
	CenterX     float64  // Centroid X position.
	CenterY     float64  // Centroid Y position.
	NodeCount   int      // Number of nodes in this cluster.
	TotalEdges  int      // Sum of edge counts from member nodes.
	NodeIDs     []string // IDs of nodes in this cluster.
	Radius      float64  // Visual radius based on node count.
	Connections []string // IDs of other clusters this cluster connects to.
	Activity    float64  // Average activity of member nodes.
	Expanded    bool     // Whether this cluster is expanded to show members.
}

// ClusterManager handles hierarchical aggregation of nodes.
type ClusterManager struct {
	mu          sync.RWMutex
	clusters    map[string]*Cluster
	nodeCluster map[string]string // Maps node ID to cluster ID.
	enabled     bool
	threshold   int
	maxClusters int
	clusterDist float64 // Distance threshold for clustering.
	expandedIDs map[string]bool
}

// NewClusterManager creates a new cluster manager.
func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		clusters:    make(map[string]*Cluster),
		nodeCluster: make(map[string]string),
		threshold:   ClusterThreshold,
		maxClusters: MaxClusters,
		clusterDist: 80.0,
		expandedIDs: make(map[string]bool),
	}
}

// SetThreshold sets the node count that triggers clustering.
func (cm *ClusterManager) SetThreshold(threshold int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.threshold = threshold
}

// SetMaxClusters sets the maximum number of clusters.
func (cm *ClusterManager) SetMaxClusters(max int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.maxClusters = max
}

// SetClusterDistance sets the distance threshold for grouping.
func (cm *ClusterManager) SetClusterDistance(dist float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clusterDist = dist
}

// IsEnabled returns whether clustering is currently active.
func (cm *ClusterManager) IsEnabled() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.enabled
}

// Enable enables clustering.
func (cm *ClusterManager) Enable() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.enabled = true
}

// Disable disables clustering.
func (cm *ClusterManager) Disable() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.enabled = false
}

// GetCluster returns the cluster for a node ID, if clustered.
func (cm *ClusterManager) GetCluster(nodeID string) *Cluster {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	clusterID, ok := cm.nodeCluster[nodeID]
	if !ok {
		return nil
	}
	return cm.clusters[clusterID]
}

// GetAllClusters returns all current clusters.
func (cm *ClusterManager) GetAllClusters() []*Cluster {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make([]*Cluster, 0, len(cm.clusters))
	for _, c := range cm.clusters {
		result = append(result, c)
	}
	return result
}

// ClusterCount returns the number of clusters.
func (cm *ClusterManager) ClusterCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.clusters)
}

// IsNodeClustered returns whether a node is part of a cluster.
func (cm *ClusterManager) IsNodeClustered(nodeID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, ok := cm.nodeCluster[nodeID]
	return ok
}

// GetClusterForNode returns the cluster ID for a node, empty if unclustered.
func (cm *ClusterManager) GetClusterForNode(nodeID string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.nodeCluster[nodeID]
}

// ExpandCluster marks a cluster as expanded to show member nodes.
func (cm *ClusterManager) ExpandCluster(clusterID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.expandedIDs[clusterID] = true
	if c, ok := cm.clusters[clusterID]; ok {
		c.Expanded = true
	}
}

// CollapseCluster marks a cluster as collapsed.
func (cm *ClusterManager) CollapseCluster(clusterID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.expandedIDs, clusterID)
	if c, ok := cm.clusters[clusterID]; ok {
		c.Expanded = false
	}
}

// IsClusterExpanded returns whether a cluster is expanded.
func (cm *ClusterManager) IsClusterExpanded(clusterID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.expandedIDs[clusterID]
}

// UpdateClusters recalculates clusters based on current node positions.
// Uses a simple grid-based spatial hashing followed by merging nearby cells.
func (cm *ClusterManager) UpdateClusters(
	nodes map[string]*Node,
	positions map[string]Position,
	edges []Edge,
) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if clustering should be enabled.
	nodeCount := len(nodes)
	if nodeCount < cm.threshold {
		cm.enabled = false
		cm.clusters = make(map[string]*Cluster)
		cm.nodeCluster = make(map[string]string)
		return
	}
	cm.enabled = true

	// Build edge map for connectivity info.
	edgeMap := make(map[string]int)
	connections := make(map[string]map[string]bool)
	for _, e := range edges {
		edgeMap[e.SourceID]++
		edgeMap[e.TargetID]++
		if connections[e.SourceID] == nil {
			connections[e.SourceID] = make(map[string]bool)
		}
		connections[e.SourceID][e.TargetID] = true
	}

	// Perform hierarchical clustering.
	clusters := cm.performClustering(nodes, positions, edgeMap)

	// Limit cluster count if too many.
	if len(clusters) > cm.maxClusters {
		clusters = cm.mergeClusters(clusters, positions, cm.maxClusters)
	}

	// Build cluster connections.
	cm.buildClusterConnections(clusters, connections)

	// Update internal state.
	cm.clusters = make(map[string]*Cluster)
	cm.nodeCluster = make(map[string]string)

	for _, c := range clusters {
		cm.clusters[c.ID] = c
		for _, nodeID := range c.NodeIDs {
			cm.nodeCluster[nodeID] = c.ID
		}
		// Preserve expanded state.
		if cm.expandedIDs[c.ID] {
			c.Expanded = true
		}
	}
}

// performClustering uses agglomerative clustering to group nearby nodes.
func (cm *ClusterManager) performClustering(
	nodes map[string]*Node,
	positions map[string]Position,
	edgeMap map[string]int,
) []*Cluster {
	clusters, nodeToCluster := cm.initializeClusters(nodes, positions, edgeMap)
	cm.mergeNearClusters(clusters, nodeToCluster)
	return cm.filterAndFinalize(clusters)
}

func (cm *ClusterManager) initializeClusters(
	nodes map[string]*Node,
	positions map[string]Position,
	edgeMap map[string]int,
) (map[string]*Cluster, map[string]string) {
	clusters := make(map[string]*Cluster)
	nodeToCluster := make(map[string]string)

	i := 0
	for nodeID, node := range nodes {
		pos := positions[nodeID]
		clusterID := generateClusterID(i)
		c := &Cluster{
			ID:         clusterID,
			CenterX:    pos.X,
			CenterY:    pos.Y,
			NodeCount:  1,
			TotalEdges: edgeMap[nodeID],
			NodeIDs:    []string{nodeID},
			Activity:   node.Activity,
		}
		clusters[clusterID] = c
		nodeToCluster[nodeID] = clusterID
		i++
	}
	return clusters, nodeToCluster
}

func (cm *ClusterManager) mergeNearClusters(clusters map[string]*Cluster, nodeToCluster map[string]string) {
	for {
		merge1, merge2, minDist := cm.findClosestPair(clusters)

		if minDist > cm.clusterDist || merge1 == "" || merge2 == "" {
			break
		}

		cm.mergeTwoClusters(clusters, nodeToCluster, merge1, merge2)
	}
}

func (cm *ClusterManager) findClosestPair(clusters map[string]*Cluster) (string, string, float64) {
	minDist := math.MaxFloat64
	var merge1, merge2 string

	clusterIDs := make([]string, 0, len(clusters))
	for id := range clusters {
		clusterIDs = append(clusterIDs, id)
	}

	for i := 0; i < len(clusterIDs); i++ {
		c1 := clusters[clusterIDs[i]]
		for j := i + 1; j < len(clusterIDs); j++ {
			c2 := clusters[clusterIDs[j]]
			dist := distance(c1.CenterX, c1.CenterY, c2.CenterX, c2.CenterY)
			if dist < minDist {
				minDist = dist
				merge1 = c1.ID
				merge2 = c2.ID
			}
		}
	}
	return merge1, merge2, minDist
}

func (cm *ClusterManager) mergeTwoClusters(clusters map[string]*Cluster, nodeToCluster map[string]string, id1, id2 string) {
	c1, c2 := clusters[id1], clusters[id2]
	totalNodes := c1.NodeCount + c2.NodeCount

	merged := &Cluster{
		ID:         c1.ID,
		CenterX:    (c1.CenterX*float64(c1.NodeCount) + c2.CenterX*float64(c2.NodeCount)) / float64(totalNodes),
		CenterY:    (c1.CenterY*float64(c1.NodeCount) + c2.CenterY*float64(c2.NodeCount)) / float64(totalNodes),
		NodeCount:  totalNodes,
		TotalEdges: c1.TotalEdges + c2.TotalEdges,
		NodeIDs:    append(c1.NodeIDs, c2.NodeIDs...),
		Activity:   (c1.Activity*float64(c1.NodeCount) + c2.Activity*float64(c2.NodeCount)) / float64(totalNodes),
	}

	clusters[c1.ID] = merged
	delete(clusters, c2.ID)

	for _, nodeID := range c2.NodeIDs {
		nodeToCluster[nodeID] = c1.ID
	}
}

func (cm *ClusterManager) filterAndFinalize(clusters map[string]*Cluster) []*Cluster {
	result := make([]*Cluster, 0, len(clusters))
	for _, c := range clusters {
		if c.NodeCount >= ClusterMinSize {
			c.Radius = calculateClusterRadius(c.NodeCount)
			result = append(result, c)
		}
	}
	return result
}

// mergeClusters reduces cluster count by merging closest pairs.
func (cm *ClusterManager) mergeClusters(
	clusters []*Cluster,
	positions map[string]Position,
	targetCount int,
) []*Cluster {
	clusterMap := make(map[string]*Cluster)
	for _, c := range clusters {
		clusterMap[c.ID] = c
	}

	for len(clusterMap) > targetCount {
		merge1, merge2 := cm.findClosestClusterPair(clusterMap)
		if merge1 == "" || merge2 == "" {
			break
		}
		cm.mergeClusterPair(clusterMap, merge1, merge2)
	}

	return cm.clusterMapToSlice(clusterMap)
}

// findClosestClusterPair identifies the two closest clusters.
func (cm *ClusterManager) findClosestClusterPair(clusterMap map[string]*Cluster) (string, string) {
	minDist := math.MaxFloat64
	var merge1, merge2 string

	clusterIDs := make([]string, 0, len(clusterMap))
	for id := range clusterMap {
		clusterIDs = append(clusterIDs, id)
	}

	for i := 0; i < len(clusterIDs); i++ {
		c1 := clusterMap[clusterIDs[i]]
		for j := i + 1; j < len(clusterIDs); j++ {
			c2 := clusterMap[clusterIDs[j]]
			dist := distance(c1.CenterX, c1.CenterY, c2.CenterX, c2.CenterY)
			if dist < minDist {
				minDist = dist
				merge1 = c1.ID
				merge2 = c2.ID
			}
		}
	}

	return merge1, merge2
}

// mergeClusterPair combines two clusters into one.
func (cm *ClusterManager) mergeClusterPair(clusterMap map[string]*Cluster, id1, id2 string) {
	c1 := clusterMap[id1]
	c2 := clusterMap[id2]

	totalNodes := c1.NodeCount + c2.NodeCount
	newX := (c1.CenterX*float64(c1.NodeCount) + c2.CenterX*float64(c2.NodeCount)) / float64(totalNodes)
	newY := (c1.CenterY*float64(c1.NodeCount) + c2.CenterY*float64(c2.NodeCount)) / float64(totalNodes)

	merged := &Cluster{
		ID:         c1.ID,
		CenterX:    newX,
		CenterY:    newY,
		NodeCount:  totalNodes,
		TotalEdges: c1.TotalEdges + c2.TotalEdges,
		NodeIDs:    append(c1.NodeIDs, c2.NodeIDs...),
		Activity:   (c1.Activity*float64(c1.NodeCount) + c2.Activity*float64(c2.NodeCount)) / float64(totalNodes),
		Radius:     calculateClusterRadius(totalNodes),
	}

	clusterMap[c1.ID] = merged
	delete(clusterMap, c2.ID)
}

// clusterMapToSlice converts cluster map to slice.
func (cm *ClusterManager) clusterMapToSlice(clusterMap map[string]*Cluster) []*Cluster {
	result := make([]*Cluster, 0, len(clusterMap))
	for _, c := range clusterMap {
		result = append(result, c)
	}
	return result
}

func (cm *ClusterManager) buildClusterConnections(
	clusters []*Cluster,
	nodeConnections map[string]map[string]bool,
) {
	nodeToCluster := cm.mapNodesToClusters(clusters)

	for _, c := range clusters {
		connectedClusters := cm.findConnectedClusters(c, nodeToCluster, nodeConnections)
		c.Connections = cm.sortedClusterIDs(connectedClusters)
	}
}

func (cm *ClusterManager) mapNodesToClusters(clusters []*Cluster) map[string]*Cluster {
	nodeToCluster := make(map[string]*Cluster)
	for _, c := range clusters {
		for _, nodeID := range c.NodeIDs {
			nodeToCluster[nodeID] = c
		}
	}
	return nodeToCluster
}

func (cm *ClusterManager) findConnectedClusters(
	cluster *Cluster,
	nodeToCluster map[string]*Cluster,
	nodeConnections map[string]map[string]bool,
) map[string]bool {
	connected := make(map[string]bool)

	for _, nodeID := range cluster.NodeIDs {
		conns, ok := nodeConnections[nodeID]
		if !ok {
			continue
		}

		for targetNode := range conns {
			targetCluster, ok := nodeToCluster[targetNode]
			if !ok || targetCluster.ID == cluster.ID {
				continue
			}
			connected[targetCluster.ID] = true
		}
	}
	return connected
}

func (cm *ClusterManager) sortedClusterIDs(clusterMap map[string]bool) []string {
	ids := make([]string, 0, len(clusterMap))
	for id := range clusterMap {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetVisibleNodes returns the list of node IDs that should be rendered.
// When clustering is enabled, this returns only nodes from expanded clusters
// plus the cluster representatives.
func (cm *ClusterManager) GetVisibleNodes(allNodeIDs []string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.enabled {
		return allNodeIDs
	}

	// Return nodes from expanded clusters.
	visible := make([]string, 0)
	for _, nodeID := range allNodeIDs {
		clusterID, ok := cm.nodeCluster[nodeID]
		if !ok {
			// Node not in any cluster - show it.
			visible = append(visible, nodeID)
			continue
		}
		// Show if cluster is expanded.
		if cm.expandedIDs[clusterID] {
			visible = append(visible, nodeID)
		}
	}

	return visible
}

// ShouldShowCluster returns whether a cluster should be rendered as a supernode.
func (cm *ClusterManager) ShouldShowCluster(clusterID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	// Show cluster if not expanded.
	return !cm.expandedIDs[clusterID]
}

// GetClusterPosition returns the position of a cluster centroid.
func (cm *ClusterManager) GetClusterPosition(clusterID string) (x, y float64, ok bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	c, exists := cm.clusters[clusterID]
	if !exists {
		return 0, 0, false
	}
	return c.CenterX, c.CenterY, true
}

// Clear removes all clusters.
func (cm *ClusterManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clusters = make(map[string]*Cluster)
	cm.nodeCluster = make(map[string]string)
	cm.expandedIDs = make(map[string]bool)
	cm.enabled = false
}

// generateClusterID creates a unique cluster identifier.
func generateClusterID(index int) string {
	return "cluster_" + string(rune('A'+index%26)) + string(rune('0'+index/26%10))
}

// distance calculates Euclidean distance between two points.
func distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

// calculateClusterRadius determines visual radius based on node count.
// Uses logarithmic scaling to prevent huge clusters.
func calculateClusterRadius(nodeCount int) float64 {
	baseRadius := 20.0
	return baseRadius + 10.0*math.Log(float64(nodeCount)+1)
}

// ClusterStats contains statistics about clustering state.
type ClusterStats struct {
	TotalNodes      int
	TotalClusters   int
	ExpandedCount   int
	LargestCluster  int
	SmallestCluster int
	AvgClusterSize  float64
}

// GetStats returns statistics about current clustering state.
func (cm *ClusterManager) GetStats() ClusterStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := ClusterStats{
		TotalClusters: len(cm.clusters),
	}

	if len(cm.clusters) == 0 {
		return stats
	}

	stats.LargestCluster = 0
	stats.SmallestCluster = math.MaxInt32
	total := 0

	for _, c := range cm.clusters {
		stats.TotalNodes += c.NodeCount
		total += c.NodeCount
		if c.NodeCount > stats.LargestCluster {
			stats.LargestCluster = c.NodeCount
		}
		if c.NodeCount < stats.SmallestCluster {
			stats.SmallestCluster = c.NodeCount
		}
		if c.Expanded {
			stats.ExpandedCount++
		}
	}

	if len(cm.clusters) > 0 {
		stats.AvgClusterSize = float64(total) / float64(len(cm.clusters))
	}

	if stats.SmallestCluster == math.MaxInt32 {
		stats.SmallestCluster = 0
	}

	return stats
}
