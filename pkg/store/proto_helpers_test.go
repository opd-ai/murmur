package store

import (
	"path/filepath"
	"testing"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

func TestMarshalPut(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	wave := &pb.Wave{
		WaveId:   []byte("test-wave-123"),
		WaveType: pb.WaveType_WAVE_TYPE_SURFACE,
	}

	t.Run("marshal and put", func(t *testing.T) {
		if err := db.MarshalPut(BucketWaves, wave.WaveId, wave); err != nil {
			t.Fatalf("MarshalPut() error: %v", err)
		}

		// Verify retrieval
		got, err := db.GetWave(wave.WaveId)
		if err != nil {
			t.Fatalf("GetWave() error: %v", err)
		}
		if !proto.Equal(got, wave) {
			t.Errorf("Retrieved wave doesn't match original")
		}
	})

	t.Run("nil message", func(t *testing.T) {
		if err := db.MarshalPut(BucketWaves, []byte("key"), nil); err == nil {
			t.Error("MarshalPut(nil) expected error")
		}
	})
}

func TestUnmarshalGet(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	wave := &pb.Wave{
		WaveId:   []byte("unmarshal-test"),
		WaveType: pb.WaveType_WAVE_TYPE_REPLY,
	}

	// Store the wave
	if err := db.PutWave(wave); err != nil {
		t.Fatalf("PutWave() error: %v", err)
	}

	t.Run("unmarshal and get", func(t *testing.T) {
		got := &pb.Wave{}
		if err := db.UnmarshalGet(BucketWaves, wave.WaveId, got); err != nil {
			t.Fatalf("UnmarshalGet() error: %v", err)
		}
		if !proto.Equal(got, wave) {
			t.Errorf("Retrieved wave doesn't match original")
		}
	})

	t.Run("key not found", func(t *testing.T) {
		got := &pb.Wave{}
		if err := db.UnmarshalGet(BucketWaves, []byte("nonexistent"), got); err != nil {
			t.Fatalf("UnmarshalGet() error: %v", err)
		}
		// Got should remain empty
		if got.WaveId != nil {
			t.Error("Expected empty wave for nonexistent key")
		}
	})
}

func TestMarshalPutBatch(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	items := map[string]proto.Message{
		"peer1": &pb.PeerRecord{PeerId: "peer1", LastSeen: 1000},
		"peer2": &pb.PeerRecord{PeerId: "peer2", LastSeen: 2000},
		"peer3": &pb.PeerRecord{PeerId: "peer3", LastSeen: 3000},
	}

	t.Run("batch put", func(t *testing.T) {
		if err := db.MarshalPutBatch(BucketPeers, items); err != nil {
			t.Fatalf("MarshalPutBatch() error: %v", err)
		}

		// Verify all items
		for id := range items {
			got, err := db.GetPeerRecord(id)
			if err != nil {
				t.Fatalf("GetPeerRecord(%s) error: %v", id, err)
			}
			if got == nil {
				t.Errorf("GetPeerRecord(%s) returned nil", id)
			}
		}
	})

	t.Run("batch with nil item", func(t *testing.T) {
		bad := map[string]proto.Message{
			"good": &pb.PeerRecord{PeerId: "good"},
			"bad":  nil,
		}
		if err := db.MarshalPutBatch(BucketPeers, bad); err == nil {
			t.Error("MarshalPutBatch() with nil item expected error")
		}
	})
}

func TestClone(t *testing.T) {
	original := &pb.Wave{
		WaveId:   []byte("original"),
		WaveType: pb.WaveType_WAVE_TYPE_SPECTER,
		Content:  []byte("hello world"),
	}

	cloned, err := Clone(original)
	if err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	if !proto.Equal(original, cloned) {
		t.Error("Cloned message doesn't equal original")
	}

	// Verify independence - modifying clone shouldn't affect original
	cloned.Content = []byte("modified")
	if proto.Equal(original, cloned) {
		t.Error("Clone is not independent from original")
	}
}

func TestSize(t *testing.T) {
	wave := &pb.Wave{
		WaveId:   []byte("size-test"),
		WaveType: pb.WaveType_WAVE_TYPE_SURFACE,
		Content:  []byte("content"),
	}

	size := Size(wave)
	if size <= 0 {
		t.Errorf("Size() = %d, want > 0", size)
	}

	// Nil message should return 0
	if Size(nil) != 0 {
		t.Error("Size(nil) should be 0")
	}
}

func TestEqual(t *testing.T) {
	a := &pb.Wave{WaveId: []byte("a"), WaveType: pb.WaveType_WAVE_TYPE_SURFACE}
	b := &pb.Wave{WaveId: []byte("a"), WaveType: pb.WaveType_WAVE_TYPE_SURFACE}
	c := &pb.Wave{WaveId: []byte("c"), WaveType: pb.WaveType_WAVE_TYPE_REPLY}

	if !Equal(a, b) {
		t.Error("Equal(a, b) = false, want true")
	}
	if Equal(a, c) {
		t.Error("Equal(a, c) = true, want false")
	}
}
