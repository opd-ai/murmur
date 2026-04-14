package waves

import (
	"bytes"
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
)

func TestValidateParentChain(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	// Create root wave.
	root := createTestSurfaceWave(t, kp, []byte("Root message"))
	provider.Add(root)

	// Create reply to root.
	reply := createTestReplyWave(t, kp, []byte("Reply message"), root.WaveId)
	provider.Add(reply)

	// Validate the chain.
	chain, err := validator.ValidateParentChain(reply)
	if err != nil {
		t.Fatalf("ValidateParentChain() error = %v", err)
	}

	// Chain should contain [root, reply].
	if len(chain) != 2 {
		t.Errorf("Chain length = %d, want 2", len(chain))
	}

	if !bytes.Equal(chain[0].WaveId, root.WaveId) {
		t.Error("Chain[0] should be root")
	}
	if !bytes.Equal(chain[1].WaveId, reply.WaveId) {
		t.Error("Chain[1] should be reply")
	}
}

func TestValidateParentChainDeep(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	// Create a chain of 5 waves.
	var lastWave *pb.Wave
	for i := 0; i < 5; i++ {
		var wave *pb.Wave
		if lastWave == nil {
			wave = createTestSurfaceWave(t, kp, []byte("Root"))
		} else {
			wave = createTestReplyWave(t, kp, []byte("Reply"), lastWave.WaveId)
		}
		provider.Add(wave)
		lastWave = wave
	}

	// Validate the chain from the last reply.
	chain, err := validator.ValidateParentChain(lastWave)
	if err != nil {
		t.Fatalf("ValidateParentChain() error = %v", err)
	}

	// Chain should contain all 5 waves.
	if len(chain) != 5 {
		t.Errorf("Chain length = %d, want 5", len(chain))
	}
}

func TestValidateParentChainCycle(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	kp := generateTestKeyPair(t)

	// Create root, then reply, then create a reply pointing back to root.
	// The cycle is: root -> reply1 -> reply2 (points back to root).
	root := createTestSurfaceWave(t, kp, []byte("Root"))
	provider.Add(root)

	reply1 := createTestReplyWave(t, kp, []byte("Reply1"), root.WaveId)
	provider.Add(reply1)

	// Create reply2 but manually modify it to point back to root's wave ID.
	reply2 := createTestReplyWave(t, kp, []byte("Reply2"), reply1.WaveId)
	provider.Add(reply2)

	// Create a fake wave that creates a cycle by having the root point to reply2.
	// We need to test the cycle detection logic. Let's create a direct self-cycle.
	selfCycleWave := createTestReplyWave(t, kp, []byte("Self"), []byte("dummy-parent"))
	// Manually set parent to itself to create cycle.
	selfCycleWave.ParentHash = selfCycleWave.WaveId
	provider.Add(selfCycleWave)

	// This should detect the self-cycle immediately.
	validator := NewParentChainValidator(provider, 1)
	_, err := validator.ValidateParentChain(selfCycleWave)
	if err != ErrParentChainCycle {
		t.Errorf("Expected ErrParentChainCycle for self-cycle, got %v", err)
	}
}

func TestValidateParentChainNotFound(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	// Create reply without adding parent.
	reply := createTestReplyWave(t, kp, []byte("Orphan reply"), []byte("missing-parent"))

	_, err := validator.ValidateParentChain(reply)
	if err != ErrParentNotFound {
		t.Errorf("Expected ErrParentNotFound, got %v", err)
	}
}

func TestValidateParentChainNotReply(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	surface := createTestSurfaceWave(t, kp, []byte("Surface message"))

	_, err := validator.ValidateParentChain(surface)
	if err != ErrNotReplyWave {
		t.Errorf("Expected ErrNotReplyWave, got %v", err)
	}
}

func TestValidateParentChainNil(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)

	_, err := validator.ValidateParentChain(nil)
	if err == nil {
		t.Error("Expected error for nil wave")
	}
}

func TestValidateParentChainWithResult(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	root := createTestSurfaceWave(t, kp, []byte("Root"))
	provider.Add(root)

	reply := createTestReplyWave(t, kp, []byte("Reply"), root.WaveId)
	provider.Add(reply)

	result, err := validator.ValidateParentChainWithResult(reply)
	if err != nil {
		t.Fatalf("ValidateParentChainWithResult() error = %v", err)
	}

	if !result.AllValid {
		t.Error("AllValid should be true")
	}
	if result.Depth != 1 {
		t.Errorf("Depth = %d, want 1", result.Depth)
	}
	if result.RootWave == nil {
		t.Error("RootWave should not be nil")
	}
	if !bytes.Equal(result.RootWave.WaveId, root.WaveId) {
		t.Error("RootWave should be the root")
	}
}

func TestValidateParentChainWithResultMissing(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	reply := createTestReplyWave(t, kp, []byte("Orphan"), []byte("missing"))

	result, err := validator.ValidateParentChainWithResult(reply)
	if err != ErrParentNotFound {
		t.Errorf("Expected ErrParentNotFound, got %v", err)
	}

	if result.AllValid {
		t.Error("AllValid should be false")
	}
	if len(result.MissingIDs) != 1 {
		t.Errorf("MissingIDs length = %d, want 1", len(result.MissingIDs))
	}
}

func TestIsReply(t *testing.T) {
	kp := generateTestKeyPair(t)

	reply := createTestReplyWave(t, kp, []byte("Reply"), []byte("parent"))
	if !IsReply(reply) {
		t.Error("IsReply() = false, want true")
	}

	surface := createTestSurfaceWave(t, kp, []byte("Surface"))
	if IsReply(surface) {
		t.Error("IsReply() = true for Surface wave, want false")
	}

	if IsReply(nil) {
		t.Error("IsReply() = true for nil, want false")
	}
}

func TestValidateReply(t *testing.T) {
	kp := generateTestKeyPair(t)

	reply := createTestReplyWave(t, kp, []byte("Reply"), []byte("parent"))
	if err := ValidateReply(reply, 1); err != nil {
		t.Errorf("ValidateReply() error = %v", err)
	}
}

func TestValidateReplyNotReply(t *testing.T) {
	kp := generateTestKeyPair(t)

	surface := createTestSurfaceWave(t, kp, []byte("Surface"))
	if err := ValidateReply(surface, 1); err != ErrNotReplyWave {
		t.Errorf("Expected ErrNotReplyWave, got %v", err)
	}
}

func TestValidateReplyMissingParent(t *testing.T) {
	wave := &pb.Wave{
		WaveType:   pb.WaveType(TypeReply),
		ParentHash: nil, // Missing
	}

	if err := ValidateReply(wave, 1); err != ErrMissingParent {
		t.Errorf("Expected ErrMissingParent, got %v", err)
	}
}

func TestGetParentHash(t *testing.T) {
	kp := generateTestKeyPair(t)
	parentHash := []byte("parent-hash-123")

	reply := createTestReplyWave(t, kp, []byte("Reply"), parentHash)

	got := GetParentHash(reply)
	if !bytes.Equal(got, parentHash) {
		t.Error("ParentHash mismatch")
	}

	surface := createTestSurfaceWave(t, kp, []byte("Surface"))
	if GetParentHash(surface) != nil {
		t.Error("GetParentHash() should return nil for non-reply")
	}

	if GetParentHash(nil) != nil {
		t.Error("GetParentHash() should return nil for nil")
	}
}

func TestGetReplyDepth(t *testing.T) {
	provider := NewInMemoryWaveProvider()
	validator := NewParentChainValidator(provider, 1)
	kp := generateTestKeyPair(t)

	// Create chain: root -> reply1 -> reply2.
	root := createTestSurfaceWave(t, kp, []byte("Root"))
	provider.Add(root)

	reply1 := createTestReplyWave(t, kp, []byte("Reply1"), root.WaveId)
	provider.Add(reply1)

	reply2 := createTestReplyWave(t, kp, []byte("Reply2"), reply1.WaveId)
	provider.Add(reply2)

	// Root depth = 0 (not a reply).
	depth, err := GetReplyDepth(root, validator)
	if err != nil || depth != 0 {
		t.Errorf("Root depth = %d, want 0", depth)
	}

	// Reply1 depth = 1.
	depth, err = GetReplyDepth(reply1, validator)
	if err != nil {
		t.Fatalf("GetReplyDepth(reply1) error = %v", err)
	}
	if depth != 1 {
		t.Errorf("Reply1 depth = %d, want 1", depth)
	}

	// Reply2 depth = 2.
	depth, err = GetReplyDepth(reply2, validator)
	if err != nil {
		t.Fatalf("GetReplyDepth(reply2) error = %v", err)
	}
	if depth != 2 {
		t.Errorf("Reply2 depth = %d, want 2", depth)
	}
}

func TestInMemoryWaveProvider(t *testing.T) {
	provider := NewInMemoryWaveProvider()

	wave := &pb.Wave{
		WaveId: []byte("test-wave"),
	}

	// Add wave.
	provider.Add(wave)

	// Get wave.
	got, err := provider.GetWave([]byte("test-wave"))
	if err != nil {
		t.Fatalf("GetWave() error = %v", err)
	}
	if got == nil {
		t.Error("GetWave() returned nil")
	}

	// Get non-existent wave.
	got, err = provider.GetWave([]byte("non-existent"))
	if err != nil || got != nil {
		t.Error("GetWave() should return nil, nil for non-existent")
	}

	// Remove wave.
	provider.Remove([]byte("test-wave"))
	got, _ = provider.GetWave([]byte("test-wave"))
	if got != nil {
		t.Error("GetWave() should return nil after Remove()")
	}
}

func TestInMemoryWaveProviderNilWave(t *testing.T) {
	provider := NewInMemoryWaveProvider()

	// Add nil should not panic.
	provider.Add(nil)

	// Add wave without ID should not add.
	provider.Add(&pb.Wave{})
}

// Helper functions.

func generateTestKeyPair(t *testing.T) *keys.KeyPair {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	return kp
}

func createTestSurfaceWave(t *testing.T, kp *keys.KeyPair, content []byte) *pb.Wave {
	opts := DefaultCreateOptions()
	opts.Difficulty = 1 // Low difficulty for fast tests

	wave, err := Create(TypeSurface, content, kp, opts)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return wave
}

func createTestReplyWave(t *testing.T, kp *keys.KeyPair, content, parentHash []byte) *pb.Wave {
	opts := DefaultCreateOptions()
	opts.Difficulty = 1
	opts.ParentHash = parentHash

	wave, err := Create(TypeReply, content, kp, opts)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return wave
}
