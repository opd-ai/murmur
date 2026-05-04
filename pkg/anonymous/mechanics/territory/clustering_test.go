// Package territory - Louvain clustering integration tests.
// Per AUDIT.md MEDIUM remediation: "Create graph with 2 dense clusters + sparse bridge,
// verify territories align with clusters not grid."
package territory

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// TestComputeTerritoriesFromGraph_TwoClusters validates that Louvain clustering
// correctly identifies two dense clusters connected by a sparse bridge.
// Per AUDIT.md: territories should align with graph topology, not grid partitioning.
func TestComputeTerritoriesFromGraph_TwoClusters(t *testing.T) {
	// Create graph with two dense clusters.
	graph := mechanics.NewLouvainGraph()

	// Cluster 1: 5 nodes in top-left (0-100, 0-100) with dense connections.
	c1Nodes := []string{"node1", "node2", "node3", "node4", "node5"}
	for i, id := range c1Nodes {
		err := graph.AddNode(id, float64(i*20), float64(i*20))
		if err != nil {
			t.Fatalf("failed to add node %s: %v", id, err)
		}
	}

	// Add dense intra-cluster edges in cluster 1 (each node connected to all others).
	for i := 0; i < len(c1Nodes); i++ {
		for j := i + 1; j < len(c1Nodes); j++ {
			err := graph.AddEdge(c1Nodes[i], c1Nodes[j], 1.0)
			if err != nil {
				t.Fatalf("failed to add edge %s-%s: %v", c1Nodes[i], c1Nodes[j], err)
			}
		}
	}

	// Cluster 2: 5 nodes in bottom-right (500-600, 500-600) with dense connections.
	c2Nodes := []string{"node6", "node7", "node8", "node9", "node10"}
	for i, id := range c2Nodes {
		err := graph.AddNode(id, 500+float64(i*20), 500+float64(i*20))
		if err != nil {
			t.Fatalf("failed to add node %s: %v", id, err)
		}
	}

	// Add dense intra-cluster edges in cluster 2.
	for i := 0; i < len(c2Nodes); i++ {
		for j := i + 1; j < len(c2Nodes); j++ {
			err := graph.AddEdge(c2Nodes[i], c2Nodes[j], 1.0)
			if err != nil {
				t.Fatalf("failed to add edge %s-%s: %v", c2Nodes[i], c2Nodes[j], err)
			}
		}
	}

	// Add sparse bridge: single edge connecting the two clusters.
	err := graph.AddEdge("node5", "node6", 0.1) // Weak bridge.
	if err != nil {
		t.Fatalf("failed to add bridge edge: %v", err)
	}

	// Compute territories using Louvain.
	manager := NewTerritoryManager()
	territories, err := manager.ComputeTerritoriesFromGraph(graph)
	if err != nil {
		t.Fatalf("failed to compute territories: %v", err)
	}

	// Expect 2 territories (one per cluster).
	if len(territories) != 2 {
		t.Fatalf("expected 2 territories, got %d", len(territories))
	}

	// Verify each territory has 5 members.
	for i, territory := range territories {
		if len(territory.MemberKeys) != 5 {
			t.Errorf("territory %d has %d members, expected 5", i, len(territory.MemberKeys))
		}
	}

	// Verify centroids are in expected regions (cluster 1 around 40,40; cluster 2 around 540,540).
	// Sort territories by centroid X to identify them.
	if territories[0].CentroidX > territories[1].CentroidX {
		territories[0], territories[1] = territories[1], territories[0]
	}

	// Territory 0 should be cluster 1 (top-left).
	if territories[0].CentroidX < 0 || territories[0].CentroidX > 100 {
		t.Errorf("territory 0 centroid X=%f, expected in range [0, 100]", territories[0].CentroidX)
	}
	if territories[0].CentroidY < 0 || territories[0].CentroidY > 100 {
		t.Errorf("territory 0 centroid Y=%f, expected in range [0, 100]", territories[0].CentroidY)
	}

	// Territory 1 should be cluster 2 (bottom-right).
	if territories[1].CentroidX < 500 || territories[1].CentroidX > 600 {
		t.Errorf("territory 1 centroid X=%f, expected in range [500, 600]", territories[1].CentroidX)
	}
	if territories[1].CentroidY < 500 || territories[1].CentroidY > 600 {
		t.Errorf("territory 1 centroid Y=%f, expected in range [500, 600]", territories[1].CentroidY)
	}

	// Verify territories are distinct (different IDs).
	if territories[0].ID == territories[1].ID {
		t.Errorf("territories have duplicate ID: %s", territories[0].ID)
	}
}

// TestComputeTerritoriesFromGraph_DenseGraph validates territory computation
// on a dense graph. Louvain behavior depends on modularity optimization and may
// produce varying cluster counts. This test validates the method runs successfully
// and produces reasonable output.
func TestComputeTerritoriesFromGraph_DenseGraph(t *testing.T) {
	graph := mechanics.NewLouvainGraph()

	// Add 6 nodes in a dense cluster.
	nodes := []string{"nodeA", "nodeB", "nodeC", "nodeD", "nodeE", "nodeF"}
	for i, id := range nodes {
		err := graph.AddNode(id, float64(i*10), float64(i*10))
		if err != nil {
			t.Fatalf("failed to add node %s: %v", id, err)
		}
	}

	// Add dense connections (each node connected to all others).
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			err := graph.AddEdge(nodes[i], nodes[j], 2.0)
			if err != nil {
				t.Fatalf("failed to add edge: %v", err)
			}
		}
	}

	manager := NewTerritoryManager()
	territories, err := manager.ComputeTerritoriesFromGraph(graph)
	if err != nil {
		t.Fatalf("failed to compute territories: %v", err)
	}

	// Validate at least one territory created.
	if len(territories) < 1 {
		t.Fatalf("expected at least 1 territory, got %d", len(territories))
	}

	// Verify total members across all territories equals 6.
	totalMembers := 0
	for _, territory := range territories {
		totalMembers += len(territory.MemberKeys)
	}
	if totalMembers != 6 {
		t.Errorf("expected 6 total members, got %d", totalMembers)
	}

	// Verify all territories have valid centroids.
	for i, territory := range territories {
		if territory.CentroidX < 0 || territory.CentroidY < 0 {
			t.Errorf("territory %d has invalid centroid: (%.0f, %.0f)",
				i, territory.CentroidX, territory.CentroidY)
		}
	}
}

// TestUpdateTerritoriesFromClusters_ExistingTerritory validates update of existing territory.
func TestUpdateTerritoriesFromClusters_ExistingTerritory(t *testing.T) {
	manager := NewTerritoryManager()

	// Create initial territory.
	t1 := NewTerritory("cluster-1", 100, 100)
	t1.MemberKeys = [][]byte{[]byte("node1"), []byte("node2")}
	manager.AddTerritory(t1)

	// Update with new cluster data (same ID, different centroid and members).
	clusters := []mechanics.TerritoryCluster{
		{
			ID:        "cluster-1",
			CentroidX: 150,
			CentroidY: 150,
			Members:   []string{"node1", "node3"}, // node2 removed, node3 added.
			Size:      2,
		},
	}

	territories := manager.UpdateTerritoriesFromClusters(clusters)
	if len(territories) != 1 {
		t.Fatalf("expected 1 territory, got %d", len(territories))
	}

	// Verify centroid updated.
	if territories[0].CentroidX != 150 || territories[0].CentroidY != 150 {
		t.Errorf("centroid not updated: got (%.0f, %.0f), expected (150, 150)",
			territories[0].CentroidX, territories[0].CentroidY)
	}

	// Verify members updated.
	if len(territories[0].MemberKeys) != 2 {
		t.Fatalf("expected 2 members, got %d", len(territories[0].MemberKeys))
	}
	if string(territories[0].MemberKeys[0]) != "node1" {
		t.Errorf("expected member 0 to be 'node1', got %s", string(territories[0].MemberKeys[0]))
	}
	if string(territories[0].MemberKeys[1]) != "node3" {
		t.Errorf("expected member 1 to be 'node3', got %s", string(territories[0].MemberKeys[1]))
	}
}

// TestUpdateTerritoriesFromClusters_NewTerritory validates creation of new territory.
func TestUpdateTerritoriesFromClusters_NewTerritory(t *testing.T) {
	manager := NewTerritoryManager()

	clusters := []mechanics.TerritoryCluster{
		{
			ID:        "cluster-new",
			CentroidX: 200,
			CentroidY: 300,
			Members:   []string{"nodeX", "nodeY"},
			Size:      2,
		},
	}

	territories := manager.UpdateTerritoriesFromClusters(clusters)
	if len(territories) != 1 {
		t.Fatalf("expected 1 territory, got %d", len(territories))
	}

	// Verify territory added to manager.
	retrieved := manager.GetTerritory("cluster-new")
	if retrieved == nil {
		t.Fatalf("territory not added to manager")
	}

	// Verify properties.
	if retrieved.CentroidX != 200 || retrieved.CentroidY != 300 {
		t.Errorf("centroid incorrect: got (%.0f, %.0f), expected (200, 300)",
			retrieved.CentroidX, retrieved.CentroidY)
	}
	if len(retrieved.MemberKeys) != 2 {
		t.Fatalf("expected 2 members, got %d", len(retrieved.MemberKeys))
	}
}

// TestComputeTerritoriesFromGraph_EmptyGraph validates error handling.
func TestComputeTerritoriesFromGraph_EmptyGraph(t *testing.T) {
	graph := mechanics.NewLouvainGraph()
	manager := NewTerritoryManager()

	_, err := manager.ComputeTerritoriesFromGraph(graph)
	if err == nil {
		t.Fatalf("expected error for empty graph, got nil")
	}
}
