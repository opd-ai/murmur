// Package shroud - Advertisement tests.
package shroud

import (
	"crypto/ed25519"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// TestGenerateAdvertisement tests relay advertisement generation.
func TestGenerateAdvertisement(t *testing.T) {
	beacon, err := NewBeacon()
	if err != nil {
		t.Fatalf("NewBeacon: %v", err)
	}

	// Should return nil when not a relay.
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	ad := beacon.GenerateAdvertisement(pubKey, privKey, []string{"/ip4/127.0.0.1/tcp/4001"})
	if ad != nil {
		t.Error("expected nil advertisement when not a relay")
	}

	// Enable relay mode.
	beacon.EnableRelay("test-peer-id", 1024*1024)

	// Generate advertisement.
	addrs := []string{"/ip4/127.0.0.1/tcp/4001", "/ip4/192.168.1.1/tcp/4001"}
	ad = beacon.GenerateAdvertisement(pubKey, privKey, addrs)
	if ad == nil {
		t.Fatal("expected advertisement when relay is enabled")
	}

	// Validate advertisement fields.
	if len(ad.Curve25519Pubkey) != 32 {
		t.Error("expected 32-byte Curve25519 pubkey")
	}
	if len(ad.Ed25519Pubkey) != ed25519.PublicKeySize {
		t.Error("expected Ed25519 pubkey")
	}
	if len(ad.Addrs) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(ad.Addrs))
	}
	if ad.Bandwidth != 1024*1024 {
		t.Errorf("expected bandwidth 1048576, got %d", ad.Bandwidth)
	}
	if len(ad.Signature) != ed25519.SignatureSize {
		t.Error("expected signature")
	}

	// Timestamp should be recent.
	now := time.Now().Unix()
	if ad.Timestamp < now-1 || ad.Timestamp > now+1 {
		t.Error("timestamp not recent")
	}

	// Expiry should be in the future.
	if ad.ExpiresAt <= now {
		t.Error("expiry not in future")
	}
}

// TestValidateAdvertisement tests advertisement validation.
func TestValidateAdvertisement(t *testing.T) {
	beacon, _ := NewBeacon()
	beacon.EnableRelay("test-peer-id", 1024*1024)
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	addrs := []string{"/ip4/127.0.0.1/tcp/4001"}

	// Generate valid advertisement.
	ad := beacon.GenerateAdvertisement(pubKey, privKey, addrs)
	if err := ValidateAdvertisement(ad); err != nil {
		t.Errorf("valid advertisement should validate: %v", err)
	}

	// Test nil advertisement.
	if err := ValidateAdvertisement(nil); err == nil {
		t.Error("nil advertisement should fail validation")
	}

	// Test expired advertisement.
	expiredAd := proto.Clone(ad).(*pb.RelayAdvertisement)
	expiredAd.ExpiresAt = time.Now().Add(-time.Hour).Unix()
	if err := ValidateAdvertisement(expiredAd); err == nil {
		t.Error("expired advertisement should fail validation")
	}

	// Test future timestamp.
	futureAd := proto.Clone(ad).(*pb.RelayAdvertisement)
	futureAd.Timestamp = time.Now().Add(10 * time.Minute).Unix()
	if err := ValidateAdvertisement(futureAd); err == nil {
		t.Error("future timestamp should fail validation")
	}

	// Test invalid Ed25519 pubkey.
	invalidPubkeyAd := proto.Clone(ad).(*pb.RelayAdvertisement)
	invalidPubkeyAd.Ed25519Pubkey = []byte("too short")
	if err := ValidateAdvertisement(invalidPubkeyAd); err == nil {
		t.Error("invalid pubkey should fail validation")
	}

	// Test invalid signature.
	invalidSigAd := proto.Clone(ad).(*pb.RelayAdvertisement)
	invalidSigAd.Signature = make([]byte, ed25519.SignatureSize)
	if err := ValidateAdvertisement(invalidSigAd); err == nil {
		t.Error("invalid signature should fail validation")
	}

	// Test tampered advertisement.
	tamperedAd := proto.Clone(ad).(*pb.RelayAdvertisement)
	tamperedAd.Bandwidth = 9999
	if err := ValidateAdvertisement(tamperedAd); err == nil {
		t.Error("tampered advertisement should fail validation")
	}

	// Test invalid Curve25519 pubkey.
	invalidCurveAd := proto.Clone(ad).(*pb.RelayAdvertisement)
	invalidCurveAd.Curve25519Pubkey = []byte("too short")
	if err := ValidateAdvertisement(invalidCurveAd); err == nil {
		t.Error("invalid Curve25519 pubkey should fail validation")
	}
}

// TestProcessAdvertisement tests processing a received advertisement.
func TestProcessAdvertisement(t *testing.T) {
	// Create two beacons: one as relay, one as client.
	relayBeacon, _ := NewBeacon()
	relayBeacon.EnableRelay("relay-peer-id", 1024*1024)
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	addrs := []string{"/ip4/127.0.0.1/tcp/4001"}

	clientBeacon, _ := NewBeacon()

	// Generate advertisement from relay.
	ad := relayBeacon.GenerateAdvertisement(pubKey, privKey, addrs)
	if ad == nil {
		t.Fatal("failed to generate advertisement")
	}

	// Client processes advertisement.
	info := clientBeacon.ProcessAdvertisement(ad, "relay-peer-id")
	if info == nil {
		t.Fatal("failed to process advertisement")
	}

	// Verify relay was added.
	if clientBeacon.RelayCount() != 1 {
		t.Errorf("expected 1 relay, got %d", clientBeacon.RelayCount())
	}

	storedInfo, ok := clientBeacon.GetRelay("relay-peer-id")
	if !ok {
		t.Fatal("relay not found after processing")
	}
	if storedInfo.PeerID != "relay-peer-id" {
		t.Errorf("expected peer ID 'relay-peer-id', got '%s'", storedInfo.PeerID)
	}
	if storedInfo.Bandwidth != 1024*1024 {
		t.Errorf("expected bandwidth 1048576, got %d", storedInfo.Bandwidth)
	}

	// Test processing invalid advertisement.
	invalidAd := &pb.RelayAdvertisement{}
	info = clientBeacon.ProcessAdvertisement(invalidAd, "invalid-peer")
	if info != nil {
		t.Error("invalid advertisement should return nil")
	}
}

// TestPruneExpiredRelays tests removal of stale relays.
func TestPruneExpiredRelays(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add some relays.
	for i := 0; i < 5; i++ {
		var pubKey [32]byte
		pubKey[0] = byte(i)
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: pubKey,
			Bandwidth: 1024,
		})
	}

	if beacon.RelayCount() != 5 {
		t.Errorf("expected 5 relays, got %d", beacon.RelayCount())
	}

	// Prune with 1 hour max age - should prune nothing (all just added).
	pruned := beacon.PruneExpiredRelays(time.Hour)
	if pruned != 0 {
		t.Errorf("expected 0 pruned (relays are fresh), got %d", pruned)
	}

	// Manually set old SeenAt for some relays.
	beacon.mu.Lock()
	for peerID, info := range beacon.relays {
		if peerID == "a" || peerID == "b" {
			info.SeenAt = time.Now().Add(-2 * time.Hour)
		}
	}
	beacon.mu.Unlock()

	// Prune with 1 hour max age - should prune 2.
	pruned = beacon.PruneExpiredRelays(time.Hour)
	if pruned != 2 {
		t.Errorf("expected 2 pruned, got %d", pruned)
	}

	if beacon.RelayCount() != 3 {
		t.Errorf("expected 3 relays remaining, got %d", beacon.RelayCount())
	}
}
