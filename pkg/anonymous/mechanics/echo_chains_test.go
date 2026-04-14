package mechanics

import (
	"crypto/rand"
	"math"
	"testing"
	"time"
)

func TestNewEchoChainStore(t *testing.T) {
	store := NewEchoChainStore()
	if store == nil {
		t.Fatal("NewEchoChainStore returned nil")
	}
	if store.CountActiveChains() != 0 {
		t.Error("new store should have no active chains")
	}
	if store.CountPendingChains() != 0 {
		t.Error("new store should have no pending chains")
	}
}

func TestRecordAmplification_FormChain(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	// Add 3 amplifications to form a chain.
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])

		chain, err := store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
		if err != nil {
			t.Fatalf("amplification %d failed: %v", i, err)
		}

		if i < 2 {
			// Chain not yet formed.
			if store.CountActiveChains() != 0 {
				t.Errorf("chain formed too early at amplification %d", i)
			}
			if store.CountPendingChains() != 1 {
				t.Errorf("expected 1 pending chain, got %d", store.CountPendingChains())
			}
		} else {
			// Chain should now be formed.
			if chain.Length() != 3 {
				t.Errorf("expected chain length 3, got %d", chain.Length())
			}
			if store.CountActiveChains() != 1 {
				t.Errorf("expected 1 active chain, got %d", store.CountActiveChains())
			}
			if store.CountPendingChains() != 0 {
				t.Errorf("expected 0 pending chains, got %d", store.CountPendingChains())
			}
			if chain.Layer != EchoChainSurface {
				t.Errorf("expected Surface layer, got %v", chain.Layer)
			}
			if chain.HasShimmer {
				t.Error("chain with 3 nodes should not have shimmer")
			}
		}
	}
}

func TestRecordAmplification_ExtendChain(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	// Form initial chain.
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}

	// Extend to 5 nodes (shimmer threshold).
	for i := 0; i < 2; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		chain, _ := store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)

		if i == 1 {
			// Should have shimmer now (5 nodes).
			if !chain.HasShimmer {
				t.Error("chain with 5 nodes should have shimmer")
			}
		}
	}

	chain, _ := store.GetChainByOriginal(originalWaveID)
	if chain.Length() != 5 {
		t.Errorf("expected chain length 5, got %d", chain.Length())
	}
}

func TestRecordAmplification_DuplicateNode(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	nodeID := make([]byte, 32)
	rand.Read(nodeID)
	var waveID1, waveID2 [32]byte
	rand.Read(waveID1[:])
	rand.Read(waveID2[:])

	// First amplification.
	store.RecordAmplification(originalWaveID, nodeID, waveID1, EchoChainSurface)

	// Duplicate should fail.
	_, err := store.RecordAmplification(originalWaveID, nodeID, waveID2, EchoChainSurface)
	if err != ErrChainDuplicate {
		t.Errorf("expected ErrChainDuplicate, got %v", err)
	}
}

func TestRecordAmplification_InvalidLayer(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])
	nodeID := make([]byte, 32)
	rand.Read(nodeID)
	var waveID [32]byte
	rand.Read(waveID[:])

	_, err := store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainLayer(99))
	if err != ErrInvalidChainLayer {
		t.Errorf("expected ErrInvalidChainLayer, got %v", err)
	}
}

func TestCalculateEchoChainBonus(t *testing.T) {
	// Too short.
	if CalculateEchoChainBonus(2) != 0.0 {
		t.Error("bonus should be 0 for chains < 3")
	}

	// Minimum length.
	bonus3 := CalculateEchoChainBonus(3)
	expected3 := math.Log(3.0)
	if math.Abs(bonus3-expected3) > 0.0001 {
		t.Errorf("expected bonus %f, got %f", expected3, bonus3)
	}

	// Longer chain gives more bonus.
	bonus5 := CalculateEchoChainBonus(5)
	if bonus5 <= bonus3 {
		t.Error("longer chain should give more bonus")
	}
}

func TestGetChain(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	// Form chain.
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}

	chain, err := store.GetChain(originalWaveID)
	if err != nil {
		t.Errorf("GetChain failed: %v", err)
	}
	if chain == nil {
		t.Fatal("chain is nil")
	}

	// Non-existent chain.
	var badID [32]byte
	rand.Read(badID[:])
	_, err = store.GetChain(badID)
	if err != ErrChainNotFound {
		t.Errorf("expected ErrChainNotFound, got %v", err)
	}
}

func TestGetNodeBonus(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	nodeIDs := make([][]byte, 3)
	for i := 0; i < 3; i++ {
		nodeIDs[i] = make([]byte, 32)
		rand.Read(nodeIDs[i])
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeIDs[i], waveID, EchoChainSurface)
	}

	// Check each participant got bonus.
	expectedBonus := math.Log(3.0)
	for i, nodeID := range nodeIDs {
		bonus := store.GetNodeBonus(nodeID)
		if math.Abs(bonus-expectedBonus) > 0.0001 {
			t.Errorf("node %d: expected bonus %f, got %f", i, expectedBonus, bonus)
		}
	}
}

func TestGetActiveChains(t *testing.T) {
	store := NewEchoChainStore()

	// Create 2 chains.
	for j := 0; j < 2; j++ {
		var originalWaveID [32]byte
		rand.Read(originalWaveID[:])
		for i := 0; i < 3; i++ {
			nodeID := make([]byte, 32)
			rand.Read(nodeID)
			var waveID [32]byte
			rand.Read(waveID[:])
			store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
		}
	}

	active := store.GetActiveChains()
	if len(active) != 2 {
		t.Errorf("expected 2 active chains, got %d", len(active))
	}
}

func TestGetChainsByLayer(t *testing.T) {
	store := NewEchoChainStore()

	// Create Surface chain.
	var surfaceWaveID [32]byte
	rand.Read(surfaceWaveID[:])
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(surfaceWaveID, nodeID, waveID, EchoChainSurface)
	}

	// Create Anonymous chain.
	var anonWaveID [32]byte
	rand.Read(anonWaveID[:])
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(anonWaveID, nodeID, waveID, EchoChainAnonymous)
	}

	surfaceChains := store.GetChainsByLayer(EchoChainSurface)
	if len(surfaceChains) != 1 {
		t.Errorf("expected 1 Surface chain, got %d", len(surfaceChains))
	}

	anonChains := store.GetChainsByLayer(EchoChainAnonymous)
	if len(anonChains) != 1 {
		t.Errorf("expected 1 Anonymous chain, got %d", len(anonChains))
	}
}

func TestGetShimmeringChains(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	// Create chain with 5 nodes (shimmer threshold).
	for i := 0; i < 5; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}

	shimmer := store.GetShimmeringChains()
	if len(shimmer) != 1 {
		t.Errorf("expected 1 shimmering chain, got %d", len(shimmer))
	}
}

func TestIsNodeInChain(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	nodeID := make([]byte, 32)
	rand.Read(nodeID)

	// Form chain including nodeID.
	for i := 0; i < 3; i++ {
		nid := nodeID
		if i > 0 {
			nid = make([]byte, 32)
			rand.Read(nid)
		}
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nid, waveID, EchoChainSurface)
	}

	if !store.IsNodeInChain(originalWaveID, nodeID) {
		t.Error("node should be in chain")
	}

	otherNode := make([]byte, 32)
	rand.Read(otherNode)
	if store.IsNodeInChain(originalWaveID, otherNode) {
		t.Error("other node should not be in chain")
	}
}

func TestExpireChains(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	// Form chain.
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}

	// Manually expire.
	chain, _ := store.GetChain(originalWaveID)
	chain.ExpiresAt = time.Now().Add(-time.Hour)

	expired := store.ExpireChains()
	if expired != 1 {
		t.Errorf("expected 1 expired, got %d", expired)
	}
}

func TestPurgePendingChains(t *testing.T) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	// Add partial chain (only 2 nodes).
	for i := 0; i < 2; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}

	// Manually age the first amplification.
	chain, _ := store.GetChainByOriginal(originalWaveID)
	chain.Nodes[0].AmplifiedAt = time.Now().Add(-2 * time.Hour)

	purged := store.PurgePendingChains(1 * time.Hour)
	if purged != 1 {
		t.Errorf("expected 1 purged, got %d", purged)
	}
}

func TestGetLongestChain(t *testing.T) {
	store := NewEchoChainStore()

	// Create short chain (3 nodes).
	var shortWaveID [32]byte
	rand.Read(shortWaveID[:])
	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(shortWaveID, nodeID, waveID, EchoChainSurface)
	}

	// Create long chain (5 nodes).
	var longWaveID [32]byte
	rand.Read(longWaveID[:])
	for i := 0; i < 5; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(longWaveID, nodeID, waveID, EchoChainSurface)
	}

	longest := store.GetLongestChain()
	if longest == nil {
		t.Fatal("longest chain is nil")
	}
	if longest.Length() != 5 {
		t.Errorf("expected longest chain length 5, got %d", longest.Length())
	}
}

func TestGetChainsContainingNode(t *testing.T) {
	store := NewEchoChainStore()

	sharedNode := make([]byte, 32)
	rand.Read(sharedNode)

	// Create 2 chains both containing sharedNode.
	for j := 0; j < 2; j++ {
		var originalWaveID [32]byte
		rand.Read(originalWaveID[:])
		for i := 0; i < 3; i++ {
			nid := sharedNode
			if i > 0 {
				nid = make([]byte, 32)
				rand.Read(nid)
			}
			var waveID [32]byte
			rand.Read(waveID[:])
			store.RecordAmplification(originalWaveID, nid, waveID, EchoChainSurface)
		}
	}

	chains := store.GetChainsContainingNode(sharedNode)
	if len(chains) != 2 {
		t.Errorf("expected 2 chains containing node, got %d", len(chains))
	}
}

func TestGetTotalBonusAwarded(t *testing.T) {
	store := NewEchoChainStore()

	if store.GetTotalBonusAwarded() != 0 {
		t.Error("initial total bonus should be 0")
	}

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	for i := 0; i < 3; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}

	total := store.GetTotalBonusAwarded()
	expected := 3 * math.Log(3.0) // 3 participants, each gets ln(3).
	if math.Abs(total-expected) > 0.0001 {
		t.Errorf("expected total bonus %f, got %f", expected, total)
	}
}

func TestEchoChainLayerString(t *testing.T) {
	tests := []struct {
		layer EchoChainLayer
		want  string
	}{
		{EchoChainSurface, "Surface"},
		{EchoChainAnonymous, "Anonymous"},
		{EchoChainLayer(99), "Unknown"},
	}

	for _, tt := range tests {
		got := EchoChainLayerString(tt.layer)
		if got != tt.want {
			t.Errorf("EchoChainLayerString(%v) = %q, want %q", tt.layer, got, tt.want)
		}
	}
}

func TestEchoChainIsExpired(t *testing.T) {
	chain := &EchoChain{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if chain.IsExpired() {
		t.Error("future chain should not be expired")
	}

	chain.ExpiresAt = time.Now().Add(-time.Hour)
	if !chain.IsExpired() {
		t.Error("past chain should be expired")
	}
}

func BenchmarkRecordAmplification(b *testing.B) {
	store := NewEchoChainStore()

	var originalWaveID [32]byte
	rand.Read(originalWaveID[:])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := make([]byte, 32)
		rand.Read(nodeID)
		var waveID [32]byte
		rand.Read(waveID[:])
		store.RecordAmplification(originalWaveID, nodeID, waveID, EchoChainSurface)
	}
}

func BenchmarkCalculateEchoChainBonus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculateEchoChainBonus(10)
	}
}
