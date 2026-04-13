package threads

import (
	"os"
	"testing"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
)

func createTestDB(t *testing.T) (*store.DB, func()) {
	t.Helper()

	f, err := os.CreateTemp("", "murmur-threads-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()

	db, err := store.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(f.Name())
	}

	return db, cleanup
}

func TestNewIndex(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, err := NewIndex(db)
	if err != nil {
		t.Fatalf("NewIndex failed: %v", err)
	}

	if idx == nil {
		t.Fatal("index is nil")
	}
}

func TestNewIndexNilDB(t *testing.T) {
	_, err := NewIndex(nil)
	if err != ErrNilStore {
		t.Errorf("expected ErrNilStore, got %v", err)
	}
}

func TestIndexAddReply(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	parent := &pb.Wave{WaveId: []byte("parent-1")}
	reply := &pb.Wave{
		WaveId:     []byte("reply-1"),
		ParentHash: []byte("parent-1"),
	}

	// Add parent first.
	if err := idx.Add(parent); err != nil {
		t.Fatalf("Add parent failed: %v", err)
	}

	// Add reply.
	if err := idx.Add(reply); err != nil {
		t.Fatalf("Add reply failed: %v", err)
	}

	// Check parent has reply.
	replies := idx.GetReplies(parent.WaveId)
	if len(replies) != 1 {
		t.Errorf("expected 1 reply, got %d", len(replies))
	}

	if string(replies[0]) != "reply-1" {
		t.Errorf("reply mismatch: got %s", replies[0])
	}
}

func TestIndexGetParent(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	reply := &pb.Wave{
		WaveId:     []byte("reply-1"),
		ParentHash: []byte("parent-1"),
	}

	idx.Add(reply)

	parent, err := idx.GetParent(reply.WaveId)
	if err != nil {
		t.Fatalf("GetParent failed: %v", err)
	}

	if string(parent) != "parent-1" {
		t.Errorf("parent mismatch: got %s", parent)
	}
}

func TestIndexGetParentNotFound(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	root := &pb.Wave{WaveId: []byte("root-1")}
	idx.Add(root)

	_, err := idx.GetParent(root.WaveId)
	if err != ErrNoParent {
		t.Errorf("expected ErrNoParent, got %v", err)
	}
}

func TestIndexGetThread(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	// Create a thread: root -> reply1 -> reply2
	root := &pb.Wave{WaveId: []byte("root")}
	reply1 := &pb.Wave{WaveId: []byte("reply1"), ParentHash: []byte("root")}
	reply2 := &pb.Wave{WaveId: []byte("reply2"), ParentHash: []byte("reply1")}

	idx.Add(root)
	idx.Add(reply1)
	idx.Add(reply2)

	// Get thread from reply2's perspective.
	thread, err := idx.GetThread(reply2.WaveId)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if len(thread) != 3 {
		t.Fatalf("expected thread length 3, got %d", len(thread))
	}

	if string(thread[0]) != "root" {
		t.Errorf("expected root at index 0, got %s", thread[0])
	}
	if string(thread[1]) != "reply1" {
		t.Errorf("expected reply1 at index 1, got %s", thread[1])
	}
	if string(thread[2]) != "reply2" {
		t.Errorf("expected reply2 at index 2, got %s", thread[2])
	}
}

func TestIndexGetDepth(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	root := &pb.Wave{WaveId: []byte("root")}
	reply1 := &pb.Wave{WaveId: []byte("reply1"), ParentHash: []byte("root")}
	reply2 := &pb.Wave{WaveId: []byte("reply2"), ParentHash: []byte("reply1")}

	idx.Add(root)
	idx.Add(reply1)
	idx.Add(reply2)

	tests := []struct {
		waveID []byte
		depth  int
	}{
		{[]byte("root"), 0},
		{[]byte("reply1"), 1},
		{[]byte("reply2"), 2},
	}

	for _, tc := range tests {
		depth, err := idx.GetDepth(tc.waveID)
		if err != nil {
			t.Errorf("GetDepth(%s) failed: %v", tc.waveID, err)
			continue
		}
		if depth != tc.depth {
			t.Errorf("GetDepth(%s) = %d, want %d", tc.waveID, depth, tc.depth)
		}
	}
}

func TestIndexGetRoot(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	root := &pb.Wave{WaveId: []byte("root")}
	reply1 := &pb.Wave{WaveId: []byte("reply1"), ParentHash: []byte("root")}
	reply2 := &pb.Wave{WaveId: []byte("reply2"), ParentHash: []byte("reply1")}

	idx.Add(root)
	idx.Add(reply1)
	idx.Add(reply2)

	// All should resolve to the same root.
	for _, waveID := range [][]byte{[]byte("root"), []byte("reply1"), []byte("reply2")} {
		rootID, err := idx.GetRoot(waveID)
		if err != nil {
			t.Errorf("GetRoot(%s) failed: %v", waveID, err)
			continue
		}
		if string(rootID) != "root" {
			t.Errorf("GetRoot(%s) = %s, want root", waveID, rootID)
		}
	}
}

func TestIndexRemove(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	root := &pb.Wave{WaveId: []byte("root")}
	reply := &pb.Wave{WaveId: []byte("reply"), ParentHash: []byte("root")}

	idx.Add(root)
	idx.Add(reply)

	// Remove reply.
	if err := idx.Remove(reply.WaveId); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Root should have no replies.
	replies := idx.GetReplies(root.WaveId)
	if len(replies) != 0 {
		t.Errorf("expected 0 replies after remove, got %d", len(replies))
	}

	// Reply should have no parent.
	_, err := idx.GetParent(reply.WaveId)
	if err != ErrNoParent {
		t.Errorf("expected ErrNoParent after remove, got %v", err)
	}
}

func TestIndexReplyCount(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	root := &pb.Wave{WaveId: []byte("root")}
	idx.Add(root)

	if count := idx.ReplyCount(root.WaveId); count != 0 {
		t.Errorf("initial ReplyCount = %d, want 0", count)
	}

	// Add 3 replies.
	for i := 0; i < 3; i++ {
		reply := &pb.Wave{
			WaveId:     []byte("reply-" + string(rune('0'+i))),
			ParentHash: []byte("root"),
		}
		idx.Add(reply)
	}

	if count := idx.ReplyCount(root.WaveId); count != 3 {
		t.Errorf("ReplyCount = %d, want 3", count)
	}
}

func TestIndexTotalReplies(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	// Build tree:
	// root
	// ├── reply1
	// │   ├── reply1a
	// │   └── reply1b
	// └── reply2

	root := &pb.Wave{WaveId: []byte("root")}
	reply1 := &pb.Wave{WaveId: []byte("reply1"), ParentHash: []byte("root")}
	reply2 := &pb.Wave{WaveId: []byte("reply2"), ParentHash: []byte("root")}
	reply1a := &pb.Wave{WaveId: []byte("reply1a"), ParentHash: []byte("reply1")}
	reply1b := &pb.Wave{WaveId: []byte("reply1b"), ParentHash: []byte("reply1")}

	for _, w := range []*pb.Wave{root, reply1, reply2, reply1a, reply1b} {
		idx.Add(w)
	}

	total := idx.TotalReplies(root.WaveId)
	if total != 4 {
		t.Errorf("TotalReplies = %d, want 4", total)
	}

	// reply1 has 2 replies.
	if total := idx.TotalReplies(reply1.WaveId); total != 2 {
		t.Errorf("TotalReplies(reply1) = %d, want 2", total)
	}

	// reply2 has no replies.
	if total := idx.TotalReplies(reply2.WaveId); total != 0 {
		t.Errorf("TotalReplies(reply2) = %d, want 0", total)
	}
}

func TestIndexCyclicThread(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	// Try to add self-referencing wave.
	selfRef := &pb.Wave{
		WaveId:     []byte("self"),
		ParentHash: []byte("self"),
	}

	err := idx.Add(selfRef)
	if err != ErrCyclicThread {
		t.Errorf("expected ErrCyclicThread, got %v", err)
	}
}

func TestIndexInvalidWave(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	// Nil wave.
	if err := idx.Add(nil); err != ErrInvalidWave {
		t.Errorf("expected ErrInvalidWave for nil, got %v", err)
	}

	// Empty wave ID.
	if err := idx.Add(&pb.Wave{}); err != ErrInvalidWave {
		t.Errorf("expected ErrInvalidWave for empty ID, got %v", err)
	}
}

func TestLoadThread(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	idx, _ := NewIndex(db)

	// Build tree.
	waves := map[string]*pb.Wave{
		"root":   {WaveId: []byte("root"), Content: []byte("root content")},
		"reply1": {WaveId: []byte("reply1"), ParentHash: []byte("root"), Content: []byte("reply1 content")},
		"reply2": {WaveId: []byte("reply2"), ParentHash: []byte("root"), Content: []byte("reply2 content")},
	}

	for _, w := range waves {
		idx.Add(w)
	}

	loader := func(id []byte) (*pb.Wave, error) {
		if w, ok := waves[string(id)]; ok {
			return w, nil
		}
		return nil, ErrNotFound
	}

	thread, err := idx.LoadThread([]byte("root"), loader)
	if err != nil {
		t.Fatalf("LoadThread failed: %v", err)
	}

	if thread.Root == nil {
		t.Fatal("thread root is nil")
	}

	if string(thread.Root.Content) != "root content" {
		t.Errorf("root content mismatch")
	}

	if len(thread.Replies) != 2 {
		t.Errorf("expected 2 replies, got %d", len(thread.Replies))
	}
}
