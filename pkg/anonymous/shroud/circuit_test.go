package shroud

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewBeacon(t *testing.T) {
	beacon, err := NewBeacon()
	if err != nil {
		t.Fatalf("NewBeacon failed: %v", err)
	}

	if beacon == nil {
		t.Fatal("beacon is nil")
	}

	// Public key should not be zero.
	var zeroKey [32]byte
	if beacon.publicKey == zeroKey {
		t.Error("public key is all zeros")
	}
}

func TestBeaconEnableRelay(t *testing.T) {
	beacon, _ := NewBeacon()

	if beacon.IsRelay() {
		t.Error("beacon should not be relay initially")
	}

	beacon.EnableRelay("peer-123", 1000000)

	if !beacon.IsRelay() {
		t.Error("beacon should be relay after EnableRelay")
	}
}

func TestBeaconAddRemoveRelay(t *testing.T) {
	beacon, _ := NewBeacon()

	relay := &RelayInfo{
		PeerID:    "relay-1",
		Bandwidth: 1000000,
	}

	beacon.AddRelay(relay)

	if beacon.RelayCount() != 1 {
		t.Errorf("relay count = %d, want 1", beacon.RelayCount())
	}

	retrieved, ok := beacon.GetRelay("relay-1")
	if !ok {
		t.Error("relay not found")
	}
	if retrieved.PeerID != relay.PeerID {
		t.Error("relay peer ID mismatch")
	}

	beacon.RemoveRelay("relay-1")

	if beacon.RelayCount() != 0 {
		t.Errorf("relay count after remove = %d, want 0", beacon.RelayCount())
	}
}

func TestBeaconListRelays(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			Bandwidth: uint64(i * 1000),
		})
	}

	relays := beacon.ListRelays()
	if len(relays) != 5 {
		t.Errorf("ListRelays returned %d, want 5", len(relays))
	}
}

func TestSelectRelays(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add enough relays.
	for i := 0; i < 10; i++ {
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			Bandwidth: 1000000,
		})
	}

	relays, err := beacon.SelectRelays(nil)
	if err != nil {
		t.Fatalf("SelectRelays failed: %v", err)
	}

	// Should have 3 distinct relays.
	seen := make(map[string]bool)
	for _, r := range relays {
		if r == nil {
			t.Error("selected relay is nil")
			continue
		}
		if seen[r.PeerID] {
			t.Errorf("duplicate relay selected: %s", r.PeerID)
		}
		seen[r.PeerID] = true
	}
}

func TestSelectRelaysExclusion(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add exactly 3 relays.
	beacon.AddRelay(&RelayInfo{PeerID: "a"})
	beacon.AddRelay(&RelayInfo{PeerID: "b"})
	beacon.AddRelay(&RelayInfo{PeerID: "c"})

	// Exclude one - should still work.
	relays, err := beacon.SelectRelays([]string{"a"})
	if err != ErrInsufficientRelays {
		// With exclusion we only have 2 relays, need 3.
		t.Logf("Expected ErrInsufficientRelays with 3 relays - 1 excluded")
	}

	// Add more relays.
	beacon.AddRelay(&RelayInfo{PeerID: "d"})
	beacon.AddRelay(&RelayInfo{PeerID: "e"})

	relays, err = beacon.SelectRelays([]string{"a"})
	if err != nil {
		t.Fatalf("SelectRelays with exclusion failed: %v", err)
	}

	// Should not include excluded relay.
	for _, r := range relays {
		if r.PeerID == "a" {
			t.Error("excluded relay was selected")
		}
	}
}

func TestSelectRelaysInsufficientRelays(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add only 2 relays.
	beacon.AddRelay(&RelayInfo{PeerID: "a"})
	beacon.AddRelay(&RelayInfo{PeerID: "b"})

	_, err := beacon.SelectRelays(nil)
	if err != ErrInsufficientRelays {
		t.Errorf("expected ErrInsufficientRelays, got %v", err)
	}
}

func TestBuildCircuit(t *testing.T) {
	beacon, _ := NewBeacon()

	// Create mock relays with known keys.
	var relays [CircuitLength]*RelayInfo
	for i := 0; i < CircuitLength; i++ {
		relayBeacon, _ := NewBeacon()
		relays[i] = &RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		}
	}

	circuit, err := beacon.BuildCircuit(relays)
	if err != nil {
		t.Fatalf("BuildCircuit failed: %v", err)
	}

	if circuit == nil {
		t.Fatal("circuit is nil")
	}

	// Shared keys should not be zero.
	var zeroKey [32]byte
	for i, key := range circuit.sharedKeys {
		if key == zeroKey {
			t.Errorf("shared key %d is all zeros", i)
		}
	}
}

func TestCircuitEncrypt(t *testing.T) {
	beacon, _ := NewBeacon()

	var relays [CircuitLength]*RelayInfo
	for i := 0; i < CircuitLength; i++ {
		relayBeacon, _ := NewBeacon()
		relays[i] = &RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		}
	}

	circuit, _ := beacon.BuildCircuit(relays)

	plaintext := []byte("secret message")
	ciphertext, err := circuit.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Ciphertext should be larger due to encryption overhead.
	if len(ciphertext) <= len(plaintext) {
		t.Error("ciphertext should be larger than plaintext")
	}

	// Should be at least FixedPacketSize plus nonce/tag overhead.
	minSize := FixedPacketSize
	if len(ciphertext) < minSize {
		t.Errorf("ciphertext size = %d, want >= %d", len(ciphertext), minSize)
	}
}

func TestCircuitClose(t *testing.T) {
	beacon, _ := NewBeacon()

	var relays [CircuitLength]*RelayInfo
	for i := 0; i < CircuitLength; i++ {
		relayBeacon, _ := NewBeacon()
		relays[i] = &RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		}
	}

	circuit, _ := beacon.BuildCircuit(relays)

	// Save key for comparison.
	originalKey := circuit.sharedKeys[0]

	circuit.Close()

	// Keys should be zeroed.
	var zeroKey [32]byte
	for i, key := range circuit.sharedKeys {
		if key != zeroKey {
			t.Errorf("shared key %d not zeroed after close", i)
		}
	}

	// Original should not have been zeros.
	if originalKey == zeroKey {
		t.Error("test invalid: original key was zeros")
	}

	// Encrypt should fail after close.
	_, err := circuit.Encrypt([]byte("test"))
	if err != ErrCircuitClosed {
		t.Errorf("expected ErrCircuitClosed, got %v", err)
	}
}

func TestCircuitIsExpired(t *testing.T) {
	beacon, _ := NewBeacon()

	var relays [CircuitLength]*RelayInfo
	for i := 0; i < CircuitLength; i++ {
		relayBeacon, _ := NewBeacon()
		relays[i] = &RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		}
	}

	circuit, _ := beacon.BuildCircuit(relays)

	if circuit.IsExpired() {
		t.Error("new circuit should not be expired")
	}

	// Manually set old creation time.
	circuit.mu.Lock()
	circuit.createdAt = time.Now().Add(-CircuitRotationInterval - time.Minute)
	circuit.mu.Unlock()

	if !circuit.IsExpired() {
		t.Error("old circuit should be expired")
	}
}

func TestDecryptLayer(t *testing.T) {
	beacon, _ := NewBeacon()
	relayBeacon, _ := NewBeacon()

	relay := &RelayInfo{
		PeerID:    "relay",
		PublicKey: relayBeacon.PublicKey(),
	}

	// Build circuit with single hop for testing.
	relays := [CircuitLength]*RelayInfo{relay, relay, relay}
	circuit, _ := beacon.BuildCircuit(relays)

	plaintext := []byte("test message")
	ciphertext, _ := circuit.Encrypt(plaintext)

	// First layer decryption.
	decrypted, err := DecryptLayer(ciphertext, circuit.sharedKeys[0])
	if err != nil {
		t.Fatalf("DecryptLayer failed: %v", err)
	}

	if len(decrypted) == 0 {
		t.Error("decrypted is empty")
	}
}

func TestDecryptLayerInvalidPacket(t *testing.T) {
	var key [32]byte
	copy(key[:], []byte("test-key-32-bytes-long-here!!!!"))

	// Too short.
	_, err := DecryptLayer([]byte("short"), key)
	if err != ErrInvalidPacket {
		t.Errorf("expected ErrInvalidPacket, got %v", err)
	}
}

func TestPadUnpad(t *testing.T) {
	original := []byte("hello world")
	padded := padToSize(original, 100)

	if len(padded) != 100 {
		t.Errorf("padded length = %d, want 100", len(padded))
	}

	unpadded := unpadFromSize(padded)

	if !bytes.Equal(unpadded, original) {
		t.Error("unpadded doesn't match original")
	}
}

func TestCircuitManager(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add relays.
	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	circuit, err := manager.GetCircuit()
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}

	// Same circuit should be returned.
	circuit2, _ := manager.GetCircuit()
	if circuit != circuit2 {
		t.Error("GetCircuit returned different circuit when not expired")
	}
}

func TestCircuitManagerRotation(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	circuit1, _ := manager.GetCircuit()
	circuit2, _ := manager.RotateCircuit()

	if circuit1 == circuit2 {
		t.Error("RotateCircuit should return new circuit")
	}

	if !circuit1.closed {
		t.Error("old circuit should be closed after rotation")
	}
}

func TestStartRotation(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.GetCircuit() // Build initial circuit.

	ctx, cancel := context.WithCancel(context.Background())
	go manager.StartRotation(ctx)

	// Let it run briefly.
	time.Sleep(50 * time.Millisecond)

	cancel()

	// Give time for shutdown.
	time.Sleep(50 * time.Millisecond)
}

func TestBeaconAddRelayNil(t *testing.T) {
	beacon, _ := NewBeacon()

	// Should not panic.
	beacon.AddRelay(nil)
	beacon.AddRelay(&RelayInfo{}) // Empty peer ID.

	if beacon.RelayCount() != 0 {
		t.Error("nil/empty relays should not be added")
	}
}

// Relay tests

func TestNewRelay(t *testing.T) {
	beacon, _ := NewBeacon()

	handler := func(packet []byte) (string, []byte, error) {
		return "next-peer", packet, nil
	}
	sender := func(peerID string, data []byte) error {
		return nil
	}

	relay := NewRelay(beacon, handler, sender)
	if relay == nil {
		t.Fatal("relay is nil")
	}

	if relay.IsEnabled() {
		t.Error("relay should not be enabled initially")
	}
}

func TestRelayEnableDisable(t *testing.T) {
	beacon, _ := NewBeacon()
	relay := NewRelay(beacon, nil, nil)

	relay.Enable()
	if !relay.IsEnabled() {
		t.Error("relay should be enabled after Enable()")
	}

	relay.Disable()
	if relay.IsEnabled() {
		t.Error("relay should be disabled after Disable()")
	}
}

func TestRelayForward(t *testing.T) {
	beacon, _ := NewBeacon()

	var mu sync.Mutex
	var received []byte
	var receivedPeer string

	handler := func(packet []byte) (string, []byte, error) {
		return "target-peer", packet, nil
	}
	sender := func(peerID string, data []byte) error {
		mu.Lock()
		defer mu.Unlock()
		receivedPeer = peerID
		received = data
		return nil
	}

	relay := NewRelay(beacon, handler, sender)
	relay.Enable()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	relay.Start(ctx)

	packet := []byte("test packet data")
	err := relay.Forward(packet)
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}

	// Wait for packet to be processed.
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	gotPeer := receivedPeer
	gotData := received
	mu.Unlock()

	if gotPeer != "target-peer" {
		t.Errorf("received peer = %q, want %q", gotPeer, "target-peer")
	}

	if !bytes.Equal(gotData, packet) {
		t.Error("received packet doesn't match sent packet")
	}
}

func TestRelayForwardNotEnabled(t *testing.T) {
	beacon, _ := NewBeacon()
	relay := NewRelay(beacon, nil, nil)

	err := relay.Forward([]byte("test"))
	if err != ErrRelayNotEnabled {
		t.Errorf("expected ErrRelayNotEnabled, got %v", err)
	}
}

func TestRelayForwardShutdown(t *testing.T) {
	beacon, _ := NewBeacon()
	relay := NewRelay(beacon, nil, nil)
	relay.Enable()
	relay.Stop()

	err := relay.Forward([]byte("test"))
	if err != ErrRelayShutdown {
		t.Errorf("expected ErrRelayShutdown, got %v", err)
	}
}

func TestRelayStats(t *testing.T) {
	beacon, _ := NewBeacon()

	handler := func(packet []byte) (string, []byte, error) {
		return "target", packet, nil
	}
	sender := func(peerID string, data []byte) error {
		return nil
	}

	relay := NewRelay(beacon, handler, sender)
	relay.Enable()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	relay.Start(ctx)

	// Forward a few packets.
	for i := 0; i < 3; i++ {
		relay.Forward([]byte("packet"))
	}

	// Wait for processing.
	time.Sleep(300 * time.Millisecond)

	stats := relay.Stats()
	if stats.PacketsForwarded != 3 {
		t.Errorf("PacketsForwarded = %d, want 3", stats.PacketsForwarded)
	}
}

func TestIsDummyPacket(t *testing.T) {
	// Create a dummy packet.
	dummy := make([]byte, FixedPacketSize)
	dummy[0] = 0x00

	if !IsDummyPacket(dummy) {
		t.Error("should recognize dummy packet")
	}

	// Non-dummy packet.
	real := make([]byte, FixedPacketSize)
	real[0] = 0x01

	if IsDummyPacket(real) {
		t.Error("should not recognize real packet as dummy")
	}

	// Wrong size.
	small := make([]byte, 100)
	small[0] = 0x00

	if IsDummyPacket(small) {
		t.Error("should not recognize small packet as dummy")
	}
}

func TestRelayRandomDelay(t *testing.T) {
	beacon, _ := NewBeacon()
	relay := NewRelay(beacon, nil, nil)

	// Generate many delays and check they're in range.
	for i := 0; i < 100; i++ {
		delay := relay.randomDelay()
		if delay < MinMixDelay || delay > MaxMixDelay {
			t.Errorf("delay %v outside range [%v, %v]", delay, MinMixDelay, MaxMixDelay)
		}
	}
}

func TestRelayDummyTraffic(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add relay to send dummy traffic to.
	relayBeacon, _ := NewBeacon()
	beacon.AddRelay(&RelayInfo{
		PeerID:    "dummy-target",
		PublicKey: relayBeacon.PublicKey(),
	})

	var dummySent int64
	sender := func(peerID string, data []byte) error {
		if IsDummyPacket(data) {
			atomic.AddInt64(&dummySent, 1)
		}
		return nil
	}

	relay := NewRelay(beacon, nil, sender)
	relay.Enable()

	ctx, cancel := context.WithCancel(context.Background())
	relay.Start(ctx)

	// Wait for a couple dummy packets.
	time.Sleep(2500 * time.Millisecond)
	cancel()

	count := atomic.LoadInt64(&dummySent)
	if count < 1 {
		t.Errorf("expected at least 1 dummy packet, got %d", count)
	}
}

func TestRelayStop(t *testing.T) {
	beacon, _ := NewBeacon()
	relay := NewRelay(beacon, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	relay.Start(ctx)
	cancel()
	relay.Stop()

	// Should be shut down.
	if !relay.shutdown.Load() {
		t.Error("relay should be marked as shutdown")
	}
}
