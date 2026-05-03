// Package mechanics - Louvain community detection for Territory Drift.
// Per ANONYMOUS_GAME_MECHANICS.md, territories are defined by the Louvain
// community detection algorithm applied to the Anonymous Layer topology.
// Each detected cluster constitutes a territory.
package mechanics

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
)

// Louvain errors.
var (
	ErrNoNodes          = errors.New("graph has no nodes")
	ErrNoEdges          = errors.New("graph has no edges")
	ErrNegativeWeight   = errors.New("negative edge weight")
	ErrSelfLoop         = errors.New("self-loops not allowed")
	ErrInvalidCommunity = errors.New("invalid community assignment")
)

// LouvainNode represents a node in the network graph.
type LouvainNode struct {
	ID        string  // Unique node identifier (peer ID).
	Community int     // Current community assignment.
	X, Y      float64 // Position on Pulse Map.
}

// LouvainEdge represents a weighted edge between two nodes.
type LouvainEdge struct {
	Source string  // Source node ID.
	Target string  // Target node ID.
	Weight float64 // Edge weight (connection strength).
}

// LouvainGraph represents the network topology for community detection.
// Uses adjacency list representation for efficient iteration.
type LouvainGraph struct {
	mu          sync.RWMutex
	nodes       map[string]*LouvainNode  // Node ID -> node data.
	edges       map[string][]LouvainEdge // Node ID -> outgoing edges.
	totalWeight float64                  // Sum of all edge weights (2m in modularity).
	nodeOrder   []string                 // Deterministic iteration order.
}

// NewLouvainGraph creates an empty graph for community detection.
func NewLouvainGraph() *LouvainGraph {
	return &LouvainGraph{
		nodes: make(map[string]*LouvainNode),
		edges: make(map[string][]LouvainEdge),
	}
}

// AddNode adds a node to the graph.
func (g *LouvainGraph) AddNode(id string, x, y float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[id]; exists {
		return nil // Already exists.
	}

	g.nodes[id] = &LouvainNode{
		ID:        id,
		Community: -1, // Unassigned.
		X:         x,
		Y:         y,
	}
	g.nodeOrder = append(g.nodeOrder, id)
	return nil
}

// AddEdge adds a weighted edge between two nodes.
func (g *LouvainGraph) AddEdge(source, target string, weight float64) error {
	if source == target {
		return ErrSelfLoop
	}
	if weight < 0 {
		return ErrNegativeWeight
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Ensure both nodes exist.
	if _, ok := g.nodes[source]; !ok {
		return fmt.Errorf("source node %s not found", source)
	}
	if _, ok := g.nodes[target]; !ok {
		return fmt.Errorf("target node %s not found", target)
	}

	// Add bidirectional edges (undirected graph).
	g.edges[source] = append(g.edges[source], LouvainEdge{Source: source, Target: target, Weight: weight})
	g.edges[target] = append(g.edges[target], LouvainEdge{Source: target, Target: source, Weight: weight})
	g.totalWeight += weight // Each edge counted once (stored twice but weight counted once).

	return nil
}

// NodeCount returns the number of nodes.
func (g *LouvainGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// EdgeCount returns the number of edges (counting each once).
func (g *LouvainGraph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	count := 0
	for _, edges := range g.edges {
		count += len(edges)
	}
	return count / 2 // Each edge stored twice.
}

// Louvain performs community detection using the Louvain algorithm.
// Returns a map from community ID to list of node IDs.
// Per ANONYMOUS_GAME_MECHANICS.md, detected clusters become territories.
type Louvain struct {
	graph         *LouvainGraph
	maxIterations int
	minModularity float64
	resolution    float64 // Resolution parameter (gamma). Default 1.0.
}

// NewLouvain creates a new Louvain algorithm instance.
func NewLouvain(graph *LouvainGraph) *Louvain {
	return &Louvain{
		graph:         graph,
		maxIterations: 100,
		minModularity: 0.0001, // Minimum modularity improvement to continue.
		resolution:    1.0,
	}
}

// SetResolution sets the resolution parameter (gamma).
// Higher values produce smaller communities.
func (l *Louvain) SetResolution(gamma float64) {
	l.resolution = gamma
}

// SetMaxIterations sets the maximum number of iterations.
func (l *Louvain) SetMaxIterations(max int) {
	l.maxIterations = max
}

// DetectCommunities runs the Louvain algorithm and returns communities.
// Returns a map from community ID (0-indexed) to list of node IDs.
func (l *Louvain) DetectCommunities() (map[int][]string, error) {
	l.graph.mu.Lock()
	defer l.graph.mu.Unlock()

	if err := l.validateGraph(); err != nil {
		return nil, err
	}

	community := l.initializeCommunities()
	degree := l.computeNodeDegrees()

	community = l.optimizeCommunities(community, degree)

	return l.renumberCommunitiesAndUpdate(community), nil
}

// validateGraph checks graph prerequisites for community detection.
func (l *Louvain) validateGraph() error {
	if len(l.graph.nodes) == 0 {
		return ErrNoNodes
	}
	if l.graph.totalWeight == 0 {
		return ErrNoEdges
	}
	return nil
}

// initializeCommunities assigns each node to its own community.
func (l *Louvain) initializeCommunities() map[string]int {
	community := make(map[string]int)
	for i, nodeID := range l.graph.nodeOrder {
		community[nodeID] = i
	}
	return community
}

// computeNodeDegrees precomputes weighted degree for each node.
func (l *Louvain) computeNodeDegrees() map[string]float64 {
	degree := make(map[string]float64)
	for nodeID, edges := range l.graph.edges {
		for _, e := range edges {
			degree[nodeID] += e.Weight
		}
	}
	return degree
}

// optimizeCommunities iteratively moves nodes to improve modularity.
func (l *Louvain) optimizeCommunities(community map[string]int, degree map[string]float64) map[string]int {
	m := l.graph.totalWeight
	m2 := 2 * m

	improved := true
	iterations := 0
	for improved && iterations < l.maxIterations {
		improved = false
		iterations++

		for _, nodeID := range l.graph.nodeOrder {
			if l.tryMoveNodeToBestCommunity(nodeID, community, degree, m, m2) {
				improved = true
			}
		}
	}
	return community
}

// tryMoveNodeToBestCommunity attempts to move a node to improve modularity.
func (l *Louvain) tryMoveNodeToBestCommunity(nodeID string, community map[string]int, degree map[string]float64, m, m2 float64) bool {
	currentCom := community[nodeID]
	bestCom := currentCom
	bestDelta := 0.0

	ki := degree[nodeID]
	kiIn := l.computeNodeCommunityWeights(nodeID, community)
	sigmaTot := l.computeCommunityTotalDegrees(community, degree)

	sigmaTotCurrent := sigmaTot[currentCom] - ki
	kiInCurrent := kiIn[currentCom]

	for neighborCom := range kiIn {
		if neighborCom == currentCom {
			continue
		}

		delta := l.computeModularityDelta(ki, kiInCurrent, kiIn[neighborCom], sigmaTotCurrent, sigmaTot[neighborCom], m, m2)

		if delta > bestDelta {
			bestDelta = delta
			bestCom = neighborCom
		}
	}

	if bestCom != currentCom && bestDelta > l.minModularity {
		community[nodeID] = bestCom
		return true
	}
	return false
}

// computeNodeCommunityWeights sums edge weights from a node to each community.
func (l *Louvain) computeNodeCommunityWeights(nodeID string, community map[string]int) map[int]float64 {
	kiIn := make(map[int]float64)
	for _, e := range l.graph.edges[nodeID] {
		neighborCom := community[e.Target]
		kiIn[neighborCom] += e.Weight
	}
	return kiIn
}

// computeCommunityTotalDegrees sums node degrees for each community.
func (l *Louvain) computeCommunityTotalDegrees(community map[string]int, degree map[string]float64) map[int]float64 {
	sigmaTot := make(map[int]float64)
	for nodeID, com := range community {
		sigmaTot[com] += degree[nodeID]
	}
	return sigmaTot
}

// computeModularityDelta calculates modularity change for moving a node between communities.
func (l *Louvain) computeModularityDelta(ki, kiInCurrent, kiInNeighbor, sigmaTotCurrent, sigmaTotNeighbor, m, m2 float64) float64 {
	removeFromCurrent := -kiInCurrent/m + l.resolution*ki*sigmaTotCurrent/(m2*m)
	addToNeighbor := kiInNeighbor/m - l.resolution*ki*(sigmaTotNeighbor+ki)/(m2*m)
	return removeFromCurrent + addToNeighbor
}

// renumberCommunitiesAndUpdate renumbers communities to be contiguous 0, 1, 2, ... and updates nodes.
func (l *Louvain) renumberCommunitiesAndUpdate(community map[string]int) map[int][]string {
	result := l.collectCommunities(community)

	comIndex := make(map[int]int)
	idx := 0
	for oldCom := range result {
		comIndex[oldCom] = idx
		idx++
	}

	finalResult := make(map[int][]string)
	for oldCom, nodes := range result {
		newCom := comIndex[oldCom]
		finalResult[newCom] = nodes
		for _, nodeID := range nodes {
			l.graph.nodes[nodeID].Community = newCom
		}
	}

	return finalResult
}

// communityTotalWeight computes the total weight of edges incident to a community.
func (l *Louvain) communityTotalWeight(community map[string]int, degree map[string]float64, com int) float64 {
	total := 0.0
	for nodeID, nodeCom := range community {
		if nodeCom == com {
			total += degree[nodeID]
		}
	}
	return total
}

// modularityGain computes the modularity gain from moving a node.
// Formula: ΔQ = [k_i,in / m] - gamma * [Σ_tot * k_i / (2m²)]
// where:
//   - k_i,in: sum of weights from node i to community
//   - k_i: degree of node i
//   - Σ_tot: sum of weights of edges incident to community
//   - m: total weight of edges
//   - gamma: resolution parameter
func (l *Louvain) modularityGain(kiIn, ki, sigmaTotal, m float64) float64 {
	if m == 0 {
		return 0
	}
	term1 := kiIn / m
	term2 := l.resolution * sigmaTotal * ki / (2 * m * m)
	return term1 - term2
}

// collectCommunities groups nodes by their community assignment.
func (l *Louvain) collectCommunities(community map[string]int) map[int][]string {
	result := make(map[int][]string)
	for nodeID, com := range community {
		result[com] = append(result[com], nodeID)
	}

	// Sort nodes within each community for determinism.
	for com := range result {
		sort.Strings(result[com])
	}

	return result
}

// Modularity computes the modularity of the current partition.
// Q = (1/2m) * Σ_ij [A_ij - gamma * k_i * k_j / 2m] * δ(c_i, c_j)
func (l *Louvain) Modularity(community map[string]int) float64 {
	l.graph.mu.RLock()
	defer l.graph.mu.RUnlock()

	if l.graph.totalWeight == 0 {
		return 0
	}

	m := l.graph.totalWeight
	q := 0.0

	// Precompute degree.
	degree := make(map[string]float64)
	for nodeID, edges := range l.graph.edges {
		for _, e := range edges {
			degree[nodeID] += e.Weight
		}
	}

	// Sum over all edges (counting each once).
	counted := make(map[string]bool)
	for nodeID, edges := range l.graph.edges {
		for _, e := range edges {
			edgeKey := edgeKeySort(nodeID, e.Target)
			if counted[edgeKey] {
				continue
			}
			counted[edgeKey] = true

			if community[nodeID] == community[e.Target] {
				// Same community: add to modularity.
				aij := e.Weight
				expected := l.resolution * degree[nodeID] * degree[e.Target] / (2 * m)
				q += aij - expected
			}
		}
	}

	return q / (2 * m)
}

// edgeKeySort returns a canonical key for an edge (undirected).
func edgeKeySort(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// TerritoryCluster represents a detected territory cluster.
type TerritoryCluster struct {
	ID        string   // Unique cluster ID (hash-based).
	CentroidX float64  // X coordinate of centroid.
	CentroidY float64  // Y coordinate of centroid.
	Members   []string // Member node IDs.
	Size      int      // Number of members.
}

// DetectTerritories runs Louvain and converts clusters to territories.
// Per ANONYMOUS_GAME_MECHANICS.md, each detected cluster becomes a territory.
func (l *Louvain) DetectTerritories() ([]TerritoryCluster, error) {
	communities, err := l.DetectCommunities()
	if err != nil {
		return nil, err
	}

	territories := make([]TerritoryCluster, 0, len(communities))
	for comID, members := range communities {
		if len(members) == 0 {
			continue
		}

		// Compute centroid.
		cx, cy := l.computeCentroid(members)

		// Generate cluster ID from member hash.
		clusterID := l.generateClusterID(comID, members)

		territories = append(territories, TerritoryCluster{
			ID:        clusterID,
			CentroidX: cx,
			CentroidY: cy,
			Members:   members,
			Size:      len(members),
		})
	}

	// Sort by size descending for consistent ordering.
	sort.Slice(territories, func(i, j int) bool {
		return territories[i].Size > territories[j].Size
	})

	return territories, nil
}

// computeCentroid calculates the geometric center of cluster members.
func (l *Louvain) computeCentroid(members []string) (float64, float64) {
	if len(members) == 0 {
		return 0, 0
	}

	var sumX, sumY float64
	for _, nodeID := range members {
		if node, ok := l.graph.nodes[nodeID]; ok {
			sumX += node.X
			sumY += node.Y
		}
	}

	n := float64(len(members))
	return sumX / n, sumY / n
}

// generateClusterID creates a stable ID for a cluster.
func (l *Louvain) generateClusterID(comID int, members []string) string {
	// Sort members for determinism.
	sorted := make([]string, len(members))
	copy(sorted, members)
	sort.Strings(sorted)

	// Hash the sorted member list.
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("cluster:%d:", comID)))
	for _, m := range sorted {
		h.Write([]byte(m))
	}

	return hex.EncodeToString(h.Sum(nil)[:16]) // 32-char hex.
}

// // UpdateTerritories integrates Louvain clusters into the territory system.
// // Returns newly created or updated Territory objects.
// func (l *Louvain) UpdateTerritories(manager *territory.TerritoryManager, clusters []TerritoryCluster) []*territory.Territory {
// 	if manager == nil {
// 		return nil
// 	}
//
// 	territories := make([]*territory.Territory, 0, len(clusters))
// 	for _, cluster := range clusters {
// 		// Check if territory already exists.
// 		existing := manager.GetTerritory(cluster.ID)
// 		if existing != nil {
// 			// Update centroid and members.
// 			existing.CentroidX = cluster.CentroidX
// 			existing.CentroidY = cluster.CentroidY
// 			existing.MemberKeys = make([][]byte, len(cluster.Members))
// 			for i, m := range cluster.Members {
// 				existing.MemberKeys[i] = []byte(m)
// 			}
// 			territories = append(territories, existing)
// 		} else {
// 			// Create new territory.
// 			t := NewTerritory(cluster.ID, cluster.CentroidX, cluster.CentroidY)
// 			t.MemberKeys = make([][]byte, len(cluster.Members))
// 			for i, m := range cluster.Members {
// 				t.MemberKeys[i] = []byte(m)
// 			}
// 			manager.AddTerritory(t)
// 			territories = append(territories, t)
// 		}
// 	}
//
// 	return territories
// }

// LouvainStats contains statistics about the community detection run.
type LouvainStats struct {
	NodeCount       int     // Number of nodes.
	EdgeCount       int     // Number of edges.
	CommunityCount  int     // Number of detected communities.
	Modularity      float64 // Final modularity score.
	LargestCluster  int     // Size of largest community.
	SmallestCluster int     // Size of smallest community.
}

// ComputeStats returns statistics about the detected communities.
func (l *Louvain) ComputeStats(communities map[int][]string) LouvainStats {
	stats := LouvainStats{
		NodeCount:       l.graph.NodeCount(),
		EdgeCount:       l.graph.EdgeCount(),
		CommunityCount:  len(communities),
		SmallestCluster: math.MaxInt,
	}

	// Build community map for modularity.
	comMap := make(map[string]int)
	for comID, members := range communities {
		for _, nodeID := range members {
			comMap[nodeID] = comID
		}
		if len(members) > stats.LargestCluster {
			stats.LargestCluster = len(members)
		}
		if len(members) < stats.SmallestCluster {
			stats.SmallestCluster = len(members)
		}
	}

	if len(communities) == 0 {
		stats.SmallestCluster = 0
	}

	stats.Modularity = l.Modularity(comMap)
	return stats
}
