package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestWaveAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	wave := &pb.Wave{
		WaveId:     []byte("test-wave-id-12345678901234567890"),
		WaveType:   pb.WaveType_WAVE_TYPE_SURFACE,
		Content:    []byte("hello world"),
		CreatedAt:  time.Now().Unix(),
		TtlSeconds: 7 * 24 * 60 * 60,
	}

	t.Run("put and get wave", func(t *testing.T) {
		if err := db.PutWave(wave); err != nil {
			t.Fatalf("PutWave() error: %v", err)
		}

		got, err := db.GetWave(wave.WaveId)
		if err != nil {
			t.Fatalf("GetWave() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetWave() returned nil")
		}
		if string(got.Content) != "hello world" {
			t.Errorf("Content = %q, want %q", string(got.Content), "hello world")
		}
	})

	t.Run("get non-existent wave", func(t *testing.T) {
		got, err := db.GetWave([]byte("non-existent"))
		if err != nil {
			t.Fatalf("GetWave() error: %v", err)
		}
		if got != nil {
			t.Error("GetWave() expected nil for non-existent wave")
		}
	})

	t.Run("list waves", func(t *testing.T) {
		waves, err := db.ListWaves()
		if err != nil {
			t.Fatalf("ListWaves() error: %v", err)
		}
		if len(waves) != 1 {
			t.Errorf("ListWaves() count = %d, want 1", len(waves))
		}
	})

	t.Run("delete wave", func(t *testing.T) {
		if err := db.DeleteWave(wave.WaveId); err != nil {
			t.Fatalf("DeleteWave() error: %v", err)
		}
		got, _ := db.GetWave(wave.WaveId)
		if got != nil {
			t.Error("Wave should be deleted")
		}
	})

	t.Run("put nil wave", func(t *testing.T) {
		if err := db.PutWave(nil); err == nil {
			t.Error("PutWave(nil) expected error")
		}
	})

	t.Run("put wave without ID", func(t *testing.T) {
		badWave := &pb.Wave{Content: []byte("test")}
		if err := db.PutWave(badWave); err == nil {
			t.Error("PutWave() without ID expected error")
		}
	})
}

func TestIdentityAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}

	decl := &pb.IdentityDeclaration{
		PublicKey:   pubKey,
		DisplayName: "Test User",
		Bio:         "Hello, I am a test user",
	}

	t.Run("put and get identity", func(t *testing.T) {
		if err := db.PutIdentityDeclaration(decl); err != nil {
			t.Fatalf("PutIdentityDeclaration() error: %v", err)
		}

		got, err := db.GetIdentityDeclaration(pubKey)
		if err != nil {
			t.Fatalf("GetIdentityDeclaration() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetIdentityDeclaration() returned nil")
		}
		if got.DisplayName != "Test User" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test User")
		}
	})

	t.Run("delete identity", func(t *testing.T) {
		if err := db.DeleteIdentityDeclaration(pubKey); err != nil {
			t.Fatalf("DeleteIdentityDeclaration() error: %v", err)
		}
		got, _ := db.GetIdentityDeclaration(pubKey)
		if got != nil {
			t.Error("Identity should be deleted")
		}
	})

	t.Run("put nil identity", func(t *testing.T) {
		if err := db.PutIdentityDeclaration(nil); err == nil {
			t.Error("PutIdentityDeclaration(nil) expected error")
		}
	})
}

func TestPeerAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	peer := &pb.PeerRecord{
		PeerId:      "12D3KooWTestPeerID",
		DisplayName: "Test Peer",
		TrustScore:  0.8,
	}

	t.Run("put and get peer", func(t *testing.T) {
		if err := db.PutPeerRecord(peer); err != nil {
			t.Fatalf("PutPeerRecord() error: %v", err)
		}

		got, err := db.GetPeerRecord("12D3KooWTestPeerID")
		if err != nil {
			t.Fatalf("GetPeerRecord() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetPeerRecord() returned nil")
		}
		if got.DisplayName != "Test Peer" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test Peer")
		}
	})

	t.Run("list peers", func(t *testing.T) {
		peers, err := db.ListPeerRecords()
		if err != nil {
			t.Fatalf("ListPeerRecords() error: %v", err)
		}
		if len(peers) != 1 {
			t.Errorf("ListPeerRecords() count = %d, want 1", len(peers))
		}
	})

	t.Run("delete peer", func(t *testing.T) {
		if err := db.DeletePeerRecord("12D3KooWTestPeerID"); err != nil {
			t.Fatalf("DeletePeerRecord() error: %v", err)
		}
		got, _ := db.GetPeerRecord("12D3KooWTestPeerID")
		if got != nil {
			t.Error("Peer should be deleted")
		}
	})

	t.Run("put nil peer", func(t *testing.T) {
		if err := db.PutPeerRecord(nil); err == nil {
			t.Error("PutPeerRecord(nil) expected error")
		}
	})
}

func TestResonanceAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	identityKey := make([]byte, 32)
	score := &pb.ResonanceScore{
		Score:     100.5,
		Milestone: pb.ResonanceMilestone_RESONANCE_MILESTONE_PHANTOM,
	}

	t.Run("put and get resonance", func(t *testing.T) {
		if err := db.PutResonanceScore(identityKey, score); err != nil {
			t.Fatalf("PutResonanceScore() error: %v", err)
		}

		got, err := db.GetResonanceScore(identityKey)
		if err != nil {
			t.Fatalf("GetResonanceScore() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetResonanceScore() returned nil")
		}
		if got.Score != 100.5 {
			t.Errorf("Score = %f, want 100.5", got.Score)
		}
	})

	t.Run("put nil resonance", func(t *testing.T) {
		if err := db.PutResonanceScore(identityKey, nil); err == nil {
			t.Error("PutResonanceScore(nil) expected error")
		}
	})
}

func TestRelayAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}

	relay := &pb.RelayAdvertisement{
		Ed25519Pubkey:    pubKey,
		Curve25519Pubkey: make([]byte, 32),
		Bandwidth:        1000,
	}

	t.Run("put and get relay", func(t *testing.T) {
		if err := db.PutRelayAdvertisement(relay); err != nil {
			t.Fatalf("PutRelayAdvertisement() error: %v", err)
		}

		got, err := db.GetRelayAdvertisement(pubKey)
		if err != nil {
			t.Fatalf("GetRelayAdvertisement() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetRelayAdvertisement() returned nil")
		}
		if got.Bandwidth != 1000 {
			t.Errorf("Bandwidth = %d, want 1000", got.Bandwidth)
		}
	})

	t.Run("list relays", func(t *testing.T) {
		relays, err := db.ListRelayAdvertisements()
		if err != nil {
			t.Fatalf("ListRelayAdvertisements() error: %v", err)
		}
		if len(relays) != 1 {
			t.Errorf("ListRelayAdvertisements() count = %d, want 1", len(relays))
		}
	})

	t.Run("put nil relay", func(t *testing.T) {
		if err := db.PutRelayAdvertisement(nil); err == nil {
			t.Error("PutRelayAdvertisement(nil) expected error")
		}
	})
}

func TestConfigAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	t.Run("put and get config", func(t *testing.T) {
		if err := db.PutConfigValue("test-key", []byte("test-value")); err != nil {
			t.Fatalf("PutConfigValue() error: %v", err)
		}

		got, err := db.GetConfigValue("test-key")
		if err != nil {
			t.Fatalf("GetConfigValue() error: %v", err)
		}
		if string(got) != "test-value" {
			t.Errorf("value = %q, want %q", string(got), "test-value")
		}
	})

	t.Run("delete config", func(t *testing.T) {
		if err := db.DeleteConfigValue("test-key"); err != nil {
			t.Fatalf("DeleteConfigValue() error: %v", err)
		}
		got, _ := db.GetConfigValue("test-key")
		if got != nil {
			t.Error("Config value should be deleted")
		}
	})
}

func TestThreadAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	waveID := []byte("wave-id-123")
	rootID := []byte("root-id-456")

	t.Run("put and get thread root", func(t *testing.T) {
		if err := db.PutThreadRoot(waveID, rootID); err != nil {
			t.Fatalf("PutThreadRoot() error: %v", err)
		}

		got, err := db.GetThreadRoot(waveID)
		if err != nil {
			t.Fatalf("GetThreadRoot() error: %v", err)
		}
		if string(got) != string(rootID) {
			t.Errorf("root = %q, want %q", string(got), string(rootID))
		}
	})

	t.Run("delete thread root", func(t *testing.T) {
		if err := db.DeleteThreadRoot(waveID); err != nil {
			t.Fatalf("DeleteThreadRoot() error: %v", err)
		}
		got, _ := db.GetThreadRoot(waveID)
		if got != nil {
			t.Error("Thread root should be deleted")
		}
	})
}

func TestMain(m *testing.M) {
	// Clean test environment
	os.Exit(m.Run())
}
