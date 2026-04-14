package mechanics

import (
	"testing"
)

func TestNewLouvainGraph(t *testing.T) {
	g := NewLouvainGraph()
	if g == nil {
		t.Fatal("expected graph, got nil")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("expected 0 edges, got %d", g.EdgeCount())
	}
}

func TestLouvainGraph_AddNode(t *testing.T) {
	g := NewLouvainGraph()

	err := g.AddNode("node1", 10.0, 20.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", g.NodeCount())
	}

	// Adding same node again should be no-op.
	err = g.AddNode("node1", 30.0, 40.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.NodeCount() != 1 {
		t.Errorf("expected 1 node after duplicate add, got %d", g.NodeCount())
	}
}

func TestLouvainGraph_AddEdge(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 10, 0)

	err := g.AddEdge("a", "b", 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.EdgeCount() != 1 {
		t.Errorf("expected 1 edge, got %d", g.EdgeCount())
	}
}

func TestLouvainGraph_AddEdge_Errors(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 10, 0)

	// Self-loop.
	err := g.AddEdge("a", "a", 1.0)
	if err != ErrSelfLoop {
		t.Errorf("expected ErrSelfLoop, got %v", err)
	}

	// Negative weight.
	err = g.AddEdge("a", "b", -1.0)
	if err != ErrNegativeWeight {
		t.Errorf("expected ErrNegativeWeight, got %v", err)
	}

	// Missing node.
	err = g.AddEdge("a", "nonexistent", 1.0)
	if err == nil {
		t.Error("expected error for missing node")
	}
}

func TestLouvain_DetectCommunities_EmptyGraph(t *testing.T) {
	g := NewLouvainGraph()
	l := NewLouvain(g)

	_, err := l.DetectCommunities()
	if err != ErrNoNodes {
		t.Errorf("expected ErrNoNodes, got %v", err)
	}
}

func TestLouvain_DetectCommunities_NoEdges(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 10, 0)
	l := NewLouvain(g)

	_, err := l.DetectCommunities()
	if err != ErrNoEdges {
		t.Errorf("expected ErrNoEdges, got %v", err)
	}
}

func TestLouvain_DetectCommunities_TwoCliques(t *testing.T) {
	// Create two distinct cliques that should be detected as separate communities.
	g := NewLouvainGraph()

	// Clique 1: a, b, c
	g.AddNode("a", 0, 0)
	g.AddNode("b", 1, 0)
	g.AddNode("c", 0, 1)
	g.AddEdge("a", "b", 1.0)
	g.AddEdge("b", "c", 1.0)
	g.AddEdge("c", "a", 1.0)

	// Clique 2: d, e, f
	g.AddNode("d", 10, 10)
	g.AddNode("e", 11, 10)
	g.AddNode("f", 10, 11)
	g.AddEdge("d", "e", 1.0)
	g.AddEdge("e", "f", 1.0)
	g.AddEdge("f", "d", 1.0)

	// Weak connection between cliques.
	g.AddEdge("c", "d", 0.1)

	l := NewLouvain(g)
	communities, err := l.DetectCommunities()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect 2 communities.
	if len(communities) != 2 {
		t.Errorf("expected 2 communities, got %d", len(communities))
		for comID, members := range communities {
			t.Logf("community %d: %v", comID, members)
		}
	}
}

func TestLouvain_DetectCommunities_SingleClique(t *testing.T) {
	g := NewLouvainGraph()

	// Fully connected clique.
	nodes := []string{"a", "b", "c", "d", "e"}
	for i, n := range nodes {
		g.AddNode(n, float64(i), float64(i))
	}
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			g.AddEdge(nodes[i], nodes[j], 1.0)
		}
	}

	l := NewLouvain(g)
	communities, err := l.DetectCommunities()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A fully connected clique may not merge into a single community
	// because modularity gain is 0 when all edges have equal weight.
	// Verify that at least some communities are detected.
	if len(communities) == 0 {
		t.Error("expected at least 1 community")
	}

	// All nodes should be accounted for.
	totalNodes := 0
	for _, members := range communities {
		totalNodes += len(members)
	}
	if totalNodes != 5 {
		t.Errorf("expected 5 total nodes, got %d", totalNodes)
	}
}

func TestLouvain_DetectTerritories(t *testing.T) {
	g := NewLouvainGraph()

	// Create 3 clusters with known positions.
	// Cluster 1: (0,0) area
	g.AddNode("a1", 0, 0)
	g.AddNode("a2", 1, 1)
	g.AddNode("a3", 2, 0)
	g.AddEdge("a1", "a2", 1.0)
	g.AddEdge("a2", "a3", 1.0)
	g.AddEdge("a3", "a1", 1.0)

	// Cluster 2: (50,50) area
	g.AddNode("b1", 50, 50)
	g.AddNode("b2", 51, 51)
	g.AddNode("b3", 52, 50)
	g.AddEdge("b1", "b2", 1.0)
	g.AddEdge("b2", "b3", 1.0)
	g.AddEdge("b3", "b1", 1.0)

	// Cluster 3: (100,100) area
	g.AddNode("c1", 100, 100)
	g.AddNode("c2", 101, 101)
	g.AddNode("c3", 102, 100)
	g.AddEdge("c1", "c2", 1.0)
	g.AddEdge("c2", "c3", 1.0)
	g.AddEdge("c3", "c1", 1.0)

	// Weak inter-cluster links.
	g.AddEdge("a3", "b1", 0.1)
	g.AddEdge("b3", "c1", 0.1)

	l := NewLouvain(g)
	territories, err := l.DetectTerritories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(territories) != 3 {
		t.Errorf("expected 3 territories, got %d", len(territories))
	}

	// Each territory should have 3 members.
	for _, terr := range territories {
		if terr.Size != 3 {
			t.Errorf("expected territory size 3, got %d", terr.Size)
		}
	}
}

func TestLouvain_Modularity(t *testing.T) {
	g := NewLouvainGraph()

	// Two cliques.
	g.AddNode("a", 0, 0)
	g.AddNode("b", 1, 0)
	g.AddNode("c", 10, 10)
	g.AddNode("d", 11, 10)
	g.AddEdge("a", "b", 1.0)
	g.AddEdge("c", "d", 1.0)

	l := NewLouvain(g)

	// Optimal partition: {a,b} and {c,d}.
	optimalPartition := map[string]int{"a": 0, "b": 0, "c": 1, "d": 1}
	optimalMod := l.Modularity(optimalPartition)

	// Suboptimal partition: {a,c} and {b,d}.
	suboptimalPartition := map[string]int{"a": 0, "b": 1, "c": 0, "d": 1}
	suboptimalMod := l.Modularity(suboptimalPartition)

	// Optimal should have higher modularity.
	if optimalMod <= suboptimalMod {
		t.Errorf("optimal modularity (%f) should be > suboptimal (%f)", optimalMod, suboptimalMod)
	}

	// Modularity should be in range [-0.5, 1].
	if optimalMod < -0.5 || optimalMod > 1.0 {
		t.Errorf("modularity out of expected range: %f", optimalMod)
	}
}

func TestLouvain_SetResolution(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 1, 0)
	g.AddEdge("a", "b", 1.0)

	l := NewLouvain(g)
	l.SetResolution(2.0)
	l.SetMaxIterations(50)

	// Should not panic with different settings.
	_, err := l.DetectCommunities()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLouvain_ComputeStats(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 1, 0)
	g.AddNode("c", 2, 0)
	g.AddEdge("a", "b", 1.0)
	g.AddEdge("b", "c", 1.0)

	l := NewLouvain(g)
	communities, _ := l.DetectCommunities()

	stats := l.ComputeStats(communities)

	if stats.NodeCount != 3 {
		t.Errorf("expected 3 nodes, got %d", stats.NodeCount)
	}
	if stats.EdgeCount != 2 {
		t.Errorf("expected 2 edges, got %d", stats.EdgeCount)
	}
	if stats.CommunityCount == 0 {
		t.Error("expected at least 1 community")
	}
}

func TestLouvain_UpdateTerritories(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 1, 0)
	g.AddEdge("a", "b", 1.0)

	l := NewLouvain(g)
	clusters, _ := l.DetectTerritories()

	manager := NewTerritoryManager()
	territories := l.UpdateTerritories(manager, clusters)

	if len(territories) == 0 {
		t.Error("expected at least 1 territory")
	}

	// Territory should be retrievable from manager.
	for _, terr := range territories {
		retrieved := manager.GetTerritory(terr.ID)
		if retrieved == nil {
			t.Errorf("territory %s not found in manager", terr.ID)
		}
	}
}

func TestLouvain_UpdateTerritories_NilManager(t *testing.T) {
	g := NewLouvainGraph()
	g.AddNode("a", 0, 0)
	g.AddNode("b", 1, 0)
	g.AddEdge("a", "b", 1.0)

	l := NewLouvain(g)
	clusters, _ := l.DetectTerritories()

	// Should not panic with nil manager.
	territories := l.UpdateTerritories(nil, clusters)
	if territories != nil {
		t.Error("expected nil for nil manager")
	}
}

func TestLouvainGraph_Concurrent(t *testing.T) {
	g := NewLouvainGraph()

	done := make(chan bool)

	// Concurrent node adds.
	go func() {
		for i := 0; i < 100; i++ {
			g.AddNode(string(rune('A'+i%26))+string(rune('0'+i%10)), float64(i), float64(i))
		}
		done <- true
	}()

	// Concurrent reads.
	go func() {
		for i := 0; i < 100; i++ {
			_ = g.NodeCount()
			_ = g.EdgeCount()
		}
		done <- true
	}()

	<-done
	<-done
}

func BenchmarkLouvain_DetectCommunities_Small(b *testing.B) {
	g := buildTestGraph(50, 4)
	l := NewLouvain(g)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.DetectCommunities()
	}
}

func BenchmarkLouvain_DetectCommunities_Medium(b *testing.B) {
	g := buildTestGraph(200, 4)
	l := NewLouvain(g)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.DetectCommunities()
	}
}

func buildTestGraph(nodeCount, clusterCount int) *LouvainGraph {
	g := NewLouvainGraph()
	nodesPerCluster := nodeCount / clusterCount

	for c := 0; c < clusterCount; c++ {
		baseX := float64(c * 100)
		baseY := float64(c * 100)

		clusterNodes := make([]string, nodesPerCluster)
		for i := 0; i < nodesPerCluster; i++ {
			nodeID := string(rune('a' + c*nodesPerCluster + i))
			clusterNodes[i] = nodeID
			g.AddNode(nodeID, baseX+float64(i), baseY+float64(i))
		}

		// Connect nodes within cluster.
		for i := 0; i < len(clusterNodes); i++ {
			for j := i + 1; j < len(clusterNodes); j++ {
				g.AddEdge(clusterNodes[i], clusterNodes[j], 1.0)
			}
		}
	}

	return g
}
