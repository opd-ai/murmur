// Package territory - Louvain clustering integration for Territory Drift.
// Per ANONYMOUS_GAME_MECHANICS.md §Territory Drift and AUDIT.md MEDIUM remediation,
// territories are defined by Louvain community detection algorithm applied to
// the Anonymous Layer topology, replacing placeholder grid-based partitioning.
package territory

import (
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// UpdateTerritoriesFromClusters integrates Louvain clusters into the territory system.
// This method replaces the placeholder grid-based partitioning with proper Louvain
// modularity-based clustering per ANONYMOUS_GAME_MECHANICS.md §Territory Definition:
// "Territories are defined by the Louvain community detection algorithm applied to
// the Anonymous Layer topology. Each detected cluster constitutes a territory."
//
// Returns newly created or updated Territory objects. Existing territories are updated
// with new centroid and member lists; new clusters create new Territory objects.
func (m *TerritoryManager) UpdateTerritoriesFromClusters(clusters []mechanics.TerritoryCluster) []*Territory {
	if len(clusters) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	territories := make([]*Territory, 0, len(clusters))
	for _, cluster := range clusters {
		// Check if territory already exists.
		existing := m.territories[cluster.ID]
		if existing != nil {
			// Update centroid and members.
			existing.mu.Lock()
			existing.CentroidX = cluster.CentroidX
			existing.CentroidY = cluster.CentroidY
			existing.MemberKeys = make([][]byte, len(cluster.Members))
			for i, m := range cluster.Members {
				existing.MemberKeys[i] = []byte(m)
			}
			existing.mu.Unlock()
			territories = append(territories, existing)
		} else {
			// Create new territory from cluster.
			t := NewTerritory(cluster.ID, cluster.CentroidX, cluster.CentroidY)
			t.MemberKeys = make([][]byte, len(cluster.Members))
			for i, m := range cluster.Members {
				t.MemberKeys[i] = []byte(m)
			}
			m.territories[t.ID] = t
			territories = append(territories, t)
		}
	}

	return territories
}

// ComputeTerritoriesFromGraph runs Louvain clustering on the provided graph
// and updates the territory manager. This is the primary method for territory
// computation per AUDIT.md MEDIUM remediation.
//
// Input: Louvain graph constructed from Pulse Map nodes and edges.
// Output: List of territories created or updated.
//
// Validation per AUDIT.md: Create graph with 2 dense clusters + sparse bridge,
// verify territories align with clusters not grid.
func (m *TerritoryManager) ComputeTerritoriesFromGraph(graph *mechanics.LouvainGraph) ([]*Territory, error) {
	// Run Louvain community detection.
	louvain := mechanics.NewLouvain(graph)
	clusters, err := louvain.DetectTerritories()
	if err != nil {
		return nil, err
	}

	// Integrate clusters into territory system.
	return m.UpdateTerritoriesFromClusters(clusters), nil
}
