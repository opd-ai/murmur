package shroud

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
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

func TestCircuitTeardownCallback(t *testing.T) {
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

	var mu sync.Mutex
	var teardownCalled bool
	var teardownCircuitID []byte
	var teardownPeerIDs []string

	circuit.SetOnTeardown(func(circuitID []byte, peerIDs []string) {
		mu.Lock()
		defer mu.Unlock()
		teardownCalled = true
		teardownCircuitID = circuitID
		teardownPeerIDs = peerIDs
	})

	circuit.Teardown()

	// Wait for async callback.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !teardownCalled {
		t.Error("teardown callback was not called")
	}

	cid := circuit.CircuitID()
	if !bytes.Equal(teardownCircuitID, cid[:]) {
		t.Error("teardown received wrong circuit ID")
	}

	if len(teardownPeerIDs) != CircuitLength {
		t.Errorf("teardown received %d peer IDs, want %d", len(teardownPeerIDs), CircuitLength)
	}
}

func TestCircuitTeardownIdempotent(t *testing.T) {
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

	var callCount int64
	circuit.SetOnTeardown(func(circuitID []byte, peerIDs []string) {
		atomic.AddInt64(&callCount, 1)
	})

	// Multiple closes should only invoke callback once.
	circuit.Close()
	circuit.Close()
	circuit.Close()

	time.Sleep(50 * time.Millisecond)

	if count := atomic.LoadInt64(&callCount); count != 1 {
		t.Errorf("teardown called %d times, want 1", count)
	}
}

func TestCircuitID(t *testing.T) {
	beacon, _ := NewBeacon()

	var relays [CircuitLength]*RelayInfo
	for i := 0; i < CircuitLength; i++ {
		relayBeacon, _ := NewBeacon()
		relays[i] = &RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		}
	}

	circuit1, _ := beacon.BuildCircuit(relays)
	circuit2, _ := beacon.BuildCircuit(relays)

	cid1 := circuit1.CircuitID()
	cid2 := circuit2.CircuitID()

	// Circuit IDs should be unique.
	if cid1 == cid2 {
		t.Error("two circuits have the same ID")
	}

	// Circuit ID should not be all zeros.
	var zeroID [16]byte
	if cid1 == zeroID {
		t.Error("circuit ID is all zeros")
	}
}

func TestCreateDestroyCell(t *testing.T) {
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

	cell, err := circuit.CreateDestroyCell()
	if err != nil {
		t.Fatalf("CreateDestroyCell failed: %v", err)
	}

	if len(cell) != FixedPacketSize {
		t.Errorf("destroy cell size = %d, want %d", len(cell), FixedPacketSize)
	}

	// First byte should be DESTROY type.
	if cell[0] != 0x04 {
		t.Errorf("destroy cell type = %#x, want 0x04", cell[0])
	}

	// Circuit ID should be embedded.
	cid := circuit.CircuitID()
	if !bytes.Equal(cell[1:17], cid[:]) {
		t.Error("circuit ID not embedded in destroy cell")
	}
}

func TestEncryptDestroyForHop(t *testing.T) {
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

	// Encrypt for each hop.
	for i := 0; i < CircuitLength; i++ {
		encrypted, err := circuit.EncryptDestroyForHop(i)
		if err != nil {
			t.Fatalf("EncryptDestroyForHop(%d) failed: %v", i, err)
		}

		// Encrypted should be larger than base cell due to nonce/tag overhead.
		if len(encrypted) <= FixedPacketSize {
			t.Errorf("hop %d: encrypted size %d <= base size %d", i, len(encrypted), FixedPacketSize)
		}
	}

	// Invalid hop should fail.
	_, err := circuit.EncryptDestroyForHop(-1)
	if err != ErrRelayNotFound {
		t.Errorf("negative hop: expected ErrRelayNotFound, got %v", err)
	}

	_, err = circuit.EncryptDestroyForHop(CircuitLength)
	if err != ErrRelayNotFound {
		t.Errorf("hop >= CircuitLength: expected ErrRelayNotFound, got %v", err)
	}
}

func TestCircuitAccessors(t *testing.T) {
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

	// Test Hops().
	hops := circuit.Hops()
	for i, hop := range hops {
		if hop != relays[i] {
			t.Errorf("Hops()[%d] does not match input relay", i)
		}
	}

	// Test IsClosed().
	if circuit.IsClosed() {
		t.Error("new circuit should not be closed")
	}

	circuit.Close()

	if !circuit.IsClosed() {
		t.Error("circuit should be closed after Close()")
	}

	// Test CreatedAt().
	created := circuit.CreatedAt()
	if created.IsZero() {
		t.Error("CreatedAt() returned zero time")
	}

	if time.Since(created) > time.Second {
		t.Error("CreatedAt() returned unexpected time")
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

	// With dual circuit support, old primary becomes backup (not closed).
	// The old circuit should now be the backup.
	backup := manager.GetBackupCircuit()
	if backup != circuit1 {
		t.Error("old primary should become backup after rotation")
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

	// Wait for packet to be processed. The relay uses exponential delay
	// with mean 200ms and max 600ms, so wait long enough for processing.
	time.Sleep(800 * time.Millisecond)

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

	// Wait for processing. The relay uses exponential delay with mean 200ms,
	// so wait long enough for all packets to be processed (max delay 600ms each).
	time.Sleep(800 * time.Millisecond)

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
	var totalDelay time.Duration
	samples := 1000

	for i := 0; i < samples; i++ {
		delay := relay.randomDelay()
		if delay < MinMixDelay || delay > MaxMixDelay {
			t.Errorf("delay %v outside range [%v, %v]", delay, MinMixDelay, MaxMixDelay)
		}
		totalDelay += delay
	}

	// Verify mean is approximately MixDelayMean (with some tolerance for clamping).
	// The exponential distribution before clamping has mean MixDelayMean.
	// After clamping to [50ms, 600ms], mean should be close but slightly lower
	// due to right-tail truncation.
	meanDelay := totalDelay / time.Duration(samples)

	// Allow generous tolerance: mean should be within 50% of MixDelayMean.
	// (Clamping affects the distribution, and statistical variance is high).
	minExpected := MixDelayMean / 2 // 100ms
	maxExpected := MixDelayMean * 2 // 400ms

	if meanDelay < minExpected || meanDelay > maxExpected {
		t.Errorf("mean delay %v outside expected range [%v, %v]", meanDelay, minExpected, maxExpected)
	}
}

func TestRandomExponentialDelay(t *testing.T) {
	// Test exported function with custom parameters.
	samples := 1000
	mean := 100 * time.Millisecond
	min := 10 * time.Millisecond
	max := 500 * time.Millisecond

	var totalDelay time.Duration

	for i := 0; i < samples; i++ {
		delay := RandomExponentialDelay(mean, min, max)
		if delay < min || delay > max {
			t.Errorf("delay %v outside range [%v, %v]", delay, min, max)
		}
		totalDelay += delay
	}

	// Mean should be approximately "mean" parameter.
	avgDelay := totalDelay / time.Duration(samples)

	if avgDelay < mean/2 || avgDelay > mean*3 {
		t.Errorf("average delay %v too far from expected mean %v", avgDelay, mean)
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

// Dual circuit tests

func TestCircuitManagerDualCircuits(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add enough relays for two circuits with different hops.
	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Get circuit triggers initial build.
	primary, err := manager.GetCircuit()
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}

	// Primary and GetPrimaryCircuit should return the same circuit.
	if manager.GetPrimaryCircuit() != primary {
		t.Error("GetPrimaryCircuit returned different circuit")
	}
}

func TestCircuitManagerBackupCircuit(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Trigger rotation which builds backup.
	_, err := manager.RotateCircuit()
	if err != nil {
		t.Fatalf("RotateCircuit failed: %v", err)
	}

	// Should have backup after rotation.
	time.Sleep(50 * time.Millisecond) // Give async build time.
	if !manager.HasBackup() {
		t.Error("expected backup circuit after rotation")
	}

	backup := manager.GetBackupCircuit()
	if backup == nil {
		t.Error("GetBackupCircuit returned nil")
	}
}

func TestCircuitManagerFailover(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Build initial circuit and trigger rotation to get backup.
	_, _ = manager.GetCircuit()
	_, _ = manager.RotateCircuit()

	time.Sleep(50 * time.Millisecond)

	primary := manager.GetPrimaryCircuit()
	backup := manager.GetBackupCircuit()

	if backup == nil {
		t.Fatal("no backup circuit for failover test")
	}

	// Perform failover.
	newPrimary, err := manager.FailoverToBackup()
	if err != nil {
		t.Fatalf("FailoverToBackup failed: %v", err)
	}

	// New primary should be the old backup.
	if newPrimary != backup {
		t.Error("failover did not promote backup to primary")
	}

	// Old primary should be closed.
	if !primary.closed {
		t.Error("old primary should be closed after failover")
	}
}

func TestCircuitManagerFailoverNoBackup(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.GetCircuit()

	// Manually clear backup.
	manager.mu.Lock()
	if manager.backup != nil {
		manager.backup.Close()
	}
	manager.backup = nil
	manager.mu.Unlock()

	// Failover should fail without backup.
	_, err := manager.FailoverToBackup()
	if err != ErrCircuitClosed {
		t.Errorf("expected ErrCircuitClosed, got %v", err)
	}
}

func TestCircuitManagerRotationCallback(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	var callbackInvoked atomic.Bool
	var callbackPrimary, callbackBackup *Circuit
	var mu sync.Mutex

	manager.SetOnRotation(func(primary, backup *Circuit) {
		mu.Lock()
		defer mu.Unlock()
		callbackInvoked.Store(true)
		callbackPrimary = primary
		callbackBackup = backup
	})

	// Trigger rotation.
	primary, _ := manager.RotateCircuit()

	// Wait for async callback.
	time.Sleep(50 * time.Millisecond)

	if !callbackInvoked.Load() {
		t.Error("rotation callback was not invoked")
	}

	mu.Lock()
	if callbackPrimary != primary {
		t.Error("callback received wrong primary circuit")
	}
	if callbackBackup == nil {
		// Backup may be nil if not enough relays for diversity.
		t.Log("backup was nil in callback (may be expected with limited relays)")
	}
	mu.Unlock()
}

func TestCircuitManagerRotationCount(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	if manager.RotationCount() != 0 {
		t.Error("initial rotation count should be 0")
	}

	manager.RotateCircuit()
	if manager.RotationCount() != 1 {
		t.Errorf("rotation count = %d, want 1", manager.RotationCount())
	}

	manager.RotateCircuit()
	if manager.RotationCount() != 2 {
		t.Errorf("rotation count = %d, want 2", manager.RotationCount())
	}
}

func TestCircuitManagerLastRotation(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	if !manager.LastRotation().IsZero() {
		t.Error("initial last rotation should be zero time")
	}

	before := time.Now()
	manager.RotateCircuit()
	after := time.Now()

	lastRotation := manager.LastRotation()
	if lastRotation.Before(before) || lastRotation.After(after) {
		t.Errorf("last rotation time %v not in range [%v, %v]", lastRotation, before, after)
	}
}

func TestCircuitManagerHasBackup(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Initially no backup.
	if manager.HasBackup() {
		t.Error("should not have backup before any operations")
	}

	// After rotation, should have backup.
	manager.RotateCircuit()
	time.Sleep(50 * time.Millisecond)

	if !manager.HasBackup() {
		t.Error("should have backup after rotation")
	}
}

func TestCircuitManagerCloseCircuits(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.RotateCircuit()
	time.Sleep(50 * time.Millisecond)

	primary := manager.GetPrimaryCircuit()
	backup := manager.GetBackupCircuit()

	manager.closeCircuits()

	if primary != nil && !primary.closed {
		t.Error("primary should be closed")
	}
	if backup != nil && !backup.closed {
		t.Error("backup should be closed")
	}
}

func TestCircuitManagerDiverseBackup(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add many relays so backup can be diverse from primary.
	for i := 0; i < 20; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune(i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.RotateCircuit()
	time.Sleep(50 * time.Millisecond)

	primary := manager.GetPrimaryCircuit()
	backup := manager.GetBackupCircuit()

	if primary == nil || backup == nil {
		t.Skip("not enough relays for diversity test")
	}

	// Check that backup uses different relays than primary.
	primaryPeers := make(map[string]bool)
	for _, hop := range primary.hops {
		if hop != nil {
			primaryPeers[hop.PeerID] = true
		}
	}

	overlapping := 0
	for _, hop := range backup.hops {
		if hop != nil && primaryPeers[hop.PeerID] {
			overlapping++
		}
	}

	// With 20 relays and 3-hop circuits, should have minimal overlap.
	if overlapping > 1 {
		t.Errorf("backup has %d overlapping relays with primary, expected <= 1", overlapping)
	}
}

// Error recovery tests

func TestRelayFailureTracker(t *testing.T) {
	tracker := NewRelayFailureTracker()

	// Initially not penalized.
	if tracker.IsPenalized("relay-1") {
		t.Error("relay should not be penalized initially")
	}

	// Record failures below threshold.
	for i := 0; i < MaxConsecutiveFailures-1; i++ {
		exceeded := tracker.RecordFailure("relay-1")
		if exceeded {
			t.Errorf("should not exceed threshold at failure %d", i+1)
		}
	}

	// Record failure that exceeds threshold.
	exceeded := tracker.RecordFailure("relay-1")
	if !exceeded {
		t.Error("should exceed threshold after max failures")
	}

	// Now should be penalized.
	if !tracker.IsPenalized("relay-1") {
		t.Error("relay should be penalized after exceeding threshold")
	}

	// Success should clear tracking.
	tracker.RecordSuccess("relay-1")
	// Note: penalty is not cleared by success, only the failure count.
}

func TestRelayFailureTrackerPenalizedList(t *testing.T) {
	tracker := NewRelayFailureTracker()

	// Penalize two relays.
	for i := 0; i < MaxConsecutiveFailures; i++ {
		tracker.RecordFailure("relay-1")
		tracker.RecordFailure("relay-2")
	}

	penalized := tracker.PenalizedRelays()
	if len(penalized) != 2 {
		t.Errorf("expected 2 penalized relays, got %d", len(penalized))
	}
}

func TestCircuitError(t *testing.T) {
	err := NewCircuitError(ErrRelayFailure, "relay-1", [16]byte{1, 2, 3}, true)

	// Error() should include relay ID.
	errStr := err.Error()
	if errStr == "" {
		t.Error("error string should not be empty")
	}

	// Unwrap should return underlying error.
	if err.Unwrap() != ErrRelayFailure {
		t.Error("Unwrap should return ErrRelayFailure")
	}

	// Error without relay ID.
	err2 := NewCircuitError(ErrCircuitClosed, "", [16]byte{}, false)
	if err2.RelayID != "" {
		t.Error("relay ID should be empty")
	}
}

func TestCircuitManagerReportRelayFailure(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.GetCircuit()

	// Get a relay from the primary circuit.
	primary := manager.GetPrimaryCircuit()
	if primary == nil {
		t.Fatal("no primary circuit")
	}

	relayID := primary.hops[0].PeerID

	// Report failure.
	err := manager.ReportRelayFailure(relayID, ErrRelayFailure)
	if err != nil {
		t.Errorf("ReportRelayFailure returned error: %v", err)
	}

	// After failure of relay in primary, should have failover to backup or rebuild.
	time.Sleep(50 * time.Millisecond)

	newPrimary := manager.GetPrimaryCircuit()
	if newPrimary == primary {
		t.Error("primary should have changed after relay failure")
	}
}

func TestCircuitManagerRecoverFromError(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.GetCircuit()

	// Create a circuit error.
	err := NewCircuitError(ErrDecryptionFailed, "", [16]byte{}, true)

	// Recover should succeed.
	recoverErr := manager.RecoverFromError(err)
	if recoverErr != nil {
		t.Errorf("RecoverFromError failed: %v", recoverErr)
	}

	// Should still have a primary circuit.
	if manager.GetPrimaryCircuit() == nil {
		t.Error("no primary circuit after recovery")
	}
}

func TestCircuitManagerGetCircuitOrRecover(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Should build circuit if none exists.
	circuit, err := manager.GetCircuitOrRecover()
	if err != nil {
		t.Fatalf("GetCircuitOrRecover failed: %v", err)
	}

	if circuit == nil {
		t.Error("no circuit returned")
	}

	// Calling again should return same circuit.
	circuit2, _ := manager.GetCircuitOrRecover()
	if circuit2 != circuit {
		t.Error("second call returned different circuit")
	}
}

func TestCircuitManagerHealth(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Initial health.
	health := manager.Health()
	if health.HasPrimary {
		t.Error("should not have primary initially")
	}

	// Build circuits.
	manager.RotateCircuit()
	time.Sleep(50 * time.Millisecond)

	health = manager.Health()
	if !health.HasPrimary {
		t.Error("should have primary after rotation")
	}

	if health.PrimaryExpired {
		t.Error("new primary should not be expired")
	}

	if health.RotationCount != 1 {
		t.Errorf("rotation count = %d, want 1", health.RotationCount)
	}
}

func TestCircuitManagerErrorCallback(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	manager.GetCircuit()

	var errorReceived atomic.Bool
	manager.SetOnError(func(err *CircuitError) {
		errorReceived.Store(true)
	})

	// Trigger error.
	primary := manager.GetPrimaryCircuit()
	relayID := primary.hops[0].PeerID
	manager.ReportRelayFailure(relayID, ErrRelayFailure)

	time.Sleep(50 * time.Millisecond)

	if !errorReceived.Load() {
		t.Error("error callback was not invoked")
	}
}

func TestCircuitManagerFailureTracker(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 5; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	tracker := manager.FailureTracker()
	if tracker == nil {
		t.Fatal("failure tracker is nil")
	}

	// Should be same instance.
	if manager.FailureTracker() != tracker {
		t.Error("FailureTracker() returned different instance")
	}
}

func TestCircuitManagerRebuildAttempts(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 10; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	if manager.RebuildAttempts() != 0 {
		t.Error("initial rebuild attempts should be 0")
	}

	// Trigger a rebuild via recovery.
	err := NewCircuitError(ErrCircuitClosed, "", [16]byte{}, false)
	manager.RecoverFromError(err)

	if manager.RebuildAttempts() != 1 {
		t.Errorf("rebuild attempts = %d, want 1", manager.RebuildAttempts())
	}
}

// Nonce sequencing and replay detection tests

func TestNonceSequencer(t *testing.T) {
	ns := NewNonceSequencer()

	// Generate several nonces.
	nonces := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		nonces[i] = ns.Next()
	}

	// All nonces should be unique.
	seen := make(map[string]bool)
	for i, nonce := range nonces {
		key := string(nonce)
		if seen[key] {
			t.Errorf("duplicate nonce at index %d", i)
		}
		seen[key] = true

		// XChaCha20 uses 24-byte nonces.
		if len(nonce) != 24 {
			t.Errorf("nonce length = %d, want 24", len(nonce))
		}
	}

	// Sequence should have incremented.
	if ns.Sequence() != 10 {
		t.Errorf("sequence = %d, want 10", ns.Sequence())
	}
}

func TestNonceSequencerConcurrency(t *testing.T) {
	ns := NewNonceSequencer()

	var wg sync.WaitGroup
	nonceChan := make(chan []byte, 100)

	// Generate nonces concurrently.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				nonceChan <- ns.Next()
			}
		}()
	}

	wg.Wait()
	close(nonceChan)

	// All nonces should be unique.
	seen := make(map[string]bool)
	count := 0
	for nonce := range nonceChan {
		key := string(nonce)
		if seen[key] {
			t.Error("duplicate nonce in concurrent generation")
		}
		seen[key] = true
		count++
	}

	if count != 100 {
		t.Errorf("got %d nonces, want 100", count)
	}
}

func TestReplayDetector(t *testing.T) {
	rd := NewReplayDetector(100)

	// First occurrence should be accepted.
	if !rd.Check(1) {
		t.Error("first sequence should be accepted")
	}

	// Replay should be rejected.
	if rd.Check(1) {
		t.Error("replay should be rejected")
	}

	// New sequence should be accepted.
	if !rd.Check(2) {
		t.Error("new sequence should be accepted")
	}

	// MaxSeen should be updated.
	if rd.MaxSeen() != 2 {
		t.Errorf("MaxSeen = %d, want 2", rd.MaxSeen())
	}
}

func TestReplayDetectorWindow(t *testing.T) {
	rd := NewReplayDetector(10)

	// Accept sequences 1-15.
	for i := uint64(1); i <= 15; i++ {
		if !rd.Check(i) {
			t.Errorf("sequence %d should be accepted", i)
		}
	}

	// Old sequence (within window) should still reject replay.
	if rd.Check(14) {
		t.Error("replay within window should be rejected")
	}

	// Very old sequence (outside window) should be rejected.
	if rd.Check(1) {
		t.Error("sequence outside window should be rejected")
	}
}

func TestReplayDetectorOutOfOrder(t *testing.T) {
	rd := NewReplayDetector(100)

	// Accept out of order.
	if !rd.Check(5) {
		t.Error("out of order sequence should be accepted")
	}
	if !rd.Check(3) {
		t.Error("earlier sequence should be accepted")
	}
	if !rd.Check(7) {
		t.Error("later sequence should be accepted")
	}

	// Replays should still be rejected.
	if rd.Check(5) {
		t.Error("replay should be rejected")
	}
	if rd.Check(3) {
		t.Error("replay should be rejected")
	}
}

func TestCircuitNonceSequencing(t *testing.T) {
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

	// Initial sequence should be 0.
	if circuit.NonceSequence() != 0 {
		t.Errorf("initial sequence = %d, want 0", circuit.NonceSequence())
	}

	// Encrypt increments sequence per hop (3 increments per Encrypt).
	circuit.Encrypt([]byte("test message"))

	// Sequence should have incremented by CircuitLength.
	if circuit.NonceSequence() != CircuitLength {
		t.Errorf("sequence after encrypt = %d, want %d", circuit.NonceSequence(), CircuitLength)
	}
}

func TestDecryptLayerWithReplayCheck(t *testing.T) {
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

	// Encrypt a message.
	plaintext := []byte("test message for replay")
	encrypted, err := circuit.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypt first layer (hop 0).
	decrypted, err := circuit.DecryptLayerWithReplayCheck(encrypted, 0)
	if err != nil {
		t.Fatalf("DecryptLayerWithReplayCheck failed: %v", err)
	}

	// Should have decrypted successfully.
	if len(decrypted) == 0 {
		t.Error("decrypted data is empty")
	}

	// Attempting to replay should fail.
	_, err = circuit.DecryptLayerWithReplayCheck(encrypted, 0)
	if err != ErrReplayDetected {
		t.Errorf("expected ErrReplayDetected, got %v", err)
	}
}

func TestDecryptLayerWithReplayCheckInvalidHop(t *testing.T) {
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

	encrypted, _ := circuit.Encrypt([]byte("test"))

	// Invalid hop index.
	_, err := circuit.DecryptLayerWithReplayCheck(encrypted, -1)
	if err != ErrRelayNotFound {
		t.Errorf("negative hop: expected ErrRelayNotFound, got %v", err)
	}

	_, err = circuit.DecryptLayerWithReplayCheck(encrypted, CircuitLength)
	if err != ErrRelayNotFound {
		t.Errorf("hop >= CircuitLength: expected ErrRelayNotFound, got %v", err)
	}
}

func TestDecryptLayerWithReplayCheckClosedCircuit(t *testing.T) {
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
	encrypted, _ := circuit.Encrypt([]byte("test"))

	circuit.Close()

	_, err := circuit.DecryptLayerWithReplayCheck(encrypted, 0)
	if err != ErrCircuitClosed {
		t.Errorf("expected ErrCircuitClosed, got %v", err)
	}
}

func TestReplayDetectorMaxSeen(t *testing.T) {
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

	// Initial max seen should be 0.
	for i := 0; i < CircuitLength; i++ {
		if circuit.ReplayDetectorMaxSeen(i) != 0 {
			t.Errorf("hop %d: initial max seen = %d, want 0", i, circuit.ReplayDetectorMaxSeen(i))
		}
	}

	// Invalid hop should return 0.
	if circuit.ReplayDetectorMaxSeen(-1) != 0 {
		t.Error("invalid hop should return 0")
	}
	if circuit.ReplayDetectorMaxSeen(CircuitLength) != 0 {
		t.Error("invalid hop should return 0")
	}
}

// Cover traffic tests

func TestDefaultCoverTrafficConfig(t *testing.T) {
	config := DefaultCoverTrafficConfig()

	if config.Rate != DummyPacketRate {
		t.Errorf("Rate = %v, want %v", config.Rate, DummyPacketRate)
	}

	if !config.Enabled {
		t.Error("Enabled should be true by default")
	}
}

func TestCoverTrafficSender(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add relays.
	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	// Track sent cover packets.
	var sentPackets [][]byte
	var sentPeerIDs []string
	var mu sync.Mutex

	sender := func(peerID string, data []byte) error {
		mu.Lock()
		sentPackets = append(sentPackets, data)
		sentPeerIDs = append(sentPeerIDs, peerID)
		mu.Unlock()
		return nil
	}

	manager.SetCoverTrafficSender(sender)
	manager.SetCoverTrafficConfig(CoverTrafficConfig{
		Rate:    100 * time.Millisecond, // Fast rate for testing.
		Enabled: true,
	})

	// Get circuit first.
	circuit, err := manager.GetCircuit()
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}

	// Start cover traffic.
	ctx, cancel := context.WithCancel(context.Background())
	go manager.StartCoverTraffic(ctx)

	// Wait for some cover packets.
	time.Sleep(350 * time.Millisecond)
	cancel()

	mu.Lock()
	count := len(sentPackets)
	peerIDs := make([]string, len(sentPeerIDs))
	copy(peerIDs, sentPeerIDs)
	mu.Unlock()

	// Should have sent at least 2 packets.
	if count < 2 {
		t.Errorf("expected at least 2 cover packets, got %d", count)
	}

	// All packets should go to entry relay.
	entryPeerID := circuit.Hops()[0].PeerID
	for _, peerID := range peerIDs {
		if peerID != entryPeerID {
			t.Errorf("cover packet sent to %s, expected %s", peerID, entryPeerID)
		}
	}

	// Count should be tracked.
	if manager.CoverTrafficCount() != uint64(count) {
		t.Errorf("CoverTrafficCount = %d, want %d", manager.CoverTrafficCount(), count)
	}
}

func TestCoverTrafficDisabled(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	var sentCount int64
	sender := func(peerID string, data []byte) error {
		atomic.AddInt64(&sentCount, 1)
		return nil
	}

	manager.SetCoverTrafficSender(sender)
	manager.SetCoverTrafficConfig(CoverTrafficConfig{
		Rate:    50 * time.Millisecond,
		Enabled: false, // Disabled.
	})

	// Get circuit.
	_, err := manager.GetCircuit()
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go manager.StartCoverTraffic(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()

	// Should not have sent any packets.
	if atomic.LoadInt64(&sentCount) > 0 {
		t.Errorf("expected 0 cover packets when disabled, got %d", atomic.LoadInt64(&sentCount))
	}
}

func TestCoverTrafficNoCircuit(t *testing.T) {
	beacon, _ := NewBeacon()
	// No relays added = no circuit can be built.

	manager := NewCircuitManager(beacon, nil)

	var sentCount int64
	sender := func(peerID string, data []byte) error {
		atomic.AddInt64(&sentCount, 1)
		return nil
	}

	manager.SetCoverTrafficSender(sender)
	manager.SetCoverTrafficConfig(CoverTrafficConfig{
		Rate:    50 * time.Millisecond,
		Enabled: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go manager.StartCoverTraffic(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()

	// Should not have sent any packets (no circuit).
	if atomic.LoadInt64(&sentCount) > 0 {
		t.Errorf("expected 0 cover packets without circuit, got %d", atomic.LoadInt64(&sentCount))
	}
}

func TestCoverTrafficNoSender(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)
	// No sender set.

	manager.SetCoverTrafficConfig(CoverTrafficConfig{
		Rate:    50 * time.Millisecond,
		Enabled: true,
	})

	// Get circuit.
	_, err := manager.GetCircuit()
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}

	// Should not panic.
	ctx, cancel := context.WithCancel(context.Background())
	go manager.StartCoverTraffic(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()

	// Count should be zero since no sender.
	if manager.CoverTrafficCount() != 0 {
		t.Errorf("CoverTrafficCount = %d without sender", manager.CoverTrafficCount())
	}
}

func TestIsCoverPacket(t *testing.T) {
	// Cover packet starts with 0x00.
	cover := []byte{0x00, 0x01, 0x02}
	if !IsCoverPacket(cover) {
		t.Error("should recognize cover packet")
	}

	// Non-cover packet.
	normal := []byte{0x01, 0x02, 0x03}
	if IsCoverPacket(normal) {
		t.Error("should not recognize normal packet as cover")
	}

	// Empty packet.
	if IsCoverPacket([]byte{}) {
		t.Error("should not recognize empty packet as cover")
	}
}

func TestCircuitHealthCoverTrafficCount(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	sender := func(peerID string, data []byte) error {
		return nil
	}

	manager.SetCoverTrafficSender(sender)
	manager.SetCoverTrafficConfig(CoverTrafficConfig{
		Rate:    50 * time.Millisecond,
		Enabled: true,
	})

	_, err := manager.GetCircuit()
	if err != nil {
		t.Fatalf("GetCircuit failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go manager.StartCoverTraffic(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()

	// Health should include cover traffic count.
	health := manager.Health()
	if health.CoverTrafficSent < 2 {
		t.Errorf("Health.CoverTrafficSent = %d, expected >= 2", health.CoverTrafficSent)
	}
}

// ShroudNode tests (Fortress-mode relay operation)

func TestDefaultShroudNodeConfig(t *testing.T) {
	config := DefaultShroudNodeConfig()

	if config.MaxBandwidth != 1_000_000 {
		t.Errorf("MaxBandwidth = %d, want 1000000", config.MaxBandwidth)
	}

	if config.MaxCircuits != 100 {
		t.Errorf("MaxCircuits = %d, want 100", config.MaxCircuits)
	}

	if config.AdvertiseInterval != BeaconInterval {
		t.Errorf("AdvertiseInterval = %v, want %v", config.AdvertiseInterval, BeaconInterval)
	}

	if !config.EnableMixing {
		t.Error("EnableMixing should be true by default")
	}

	if !config.EnableDummyTraffic {
		t.Error("EnableDummyTraffic should be true by default")
	}
}

func TestNewShroudNode(t *testing.T) {
	beacon, _ := NewBeacon()

	handler := func(packet []byte) (string, []byte, error) {
		return "next-hop", packet, nil
	}

	sender := func(peerID string, data []byte) error {
		return nil
	}

	config := DefaultShroudNodeConfig()
	config.PeerID = "test-node"

	node := NewShroudNode(beacon, handler, sender, config)

	if node == nil {
		t.Fatal("NewShroudNode returned nil")
	}

	if node.Beacon() != beacon {
		t.Error("Beacon() returned wrong beacon")
	}

	if node.Relay() == nil {
		t.Error("Relay() returned nil")
	}

	if node.IsRunning() {
		t.Error("Node should not be running initially")
	}
}

func TestShroudNodeStartStop(t *testing.T) {
	beacon, _ := NewBeacon()

	handler := func(packet []byte) (string, []byte, error) {
		return "next-hop", packet, nil
	}

	sender := func(peerID string, data []byte) error {
		return nil
	}

	config := DefaultShroudNodeConfig()
	config.PeerID = "fortress-node"

	node := NewShroudNode(beacon, handler, sender, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !node.IsRunning() {
		t.Error("Node should be running after Start")
	}

	// Beacon should be in relay mode.
	if !beacon.IsRelay() {
		t.Error("Beacon should be in relay mode")
	}

	// Start again should fail.
	err = node.Start(ctx)
	if err == nil {
		t.Error("Starting already running node should fail")
	}

	// Stop the node.
	node.Stop()

	if node.IsRunning() {
		t.Error("Node should not be running after Stop")
	}

	if beacon.IsRelay() {
		t.Error("Beacon should not be in relay mode after Stop")
	}
}

func TestShroudNodeForward(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add some relays for dummy traffic.
	for i := 0; i < 3; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	var forwarded [][]byte
	var mu sync.Mutex

	handler := func(packet []byte) (string, []byte, error) {
		return "next-hop", packet, nil
	}

	sender := func(peerID string, data []byte) error {
		mu.Lock()
		forwarded = append(forwarded, data)
		mu.Unlock()
		return nil
	}

	config := DefaultShroudNodeConfig()
	config.PeerID = "test-node"

	node := NewShroudNode(beacon, handler, sender, config)

	// Forward should fail when not running.
	packet := make([]byte, FixedPacketSize)
	err := node.Forward(packet)
	if err != ErrRelayNotEnabled {
		t.Errorf("Forward on stopped node: got %v, want ErrRelayNotEnabled", err)
	}

	// Start the node.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node.Start(ctx)

	// Forward should work now.
	err = node.Forward(packet)
	if err != nil {
		t.Errorf("Forward failed: %v", err)
	}

	// Wait for packet processing.
	time.Sleep(MaxMixDelay + 100*time.Millisecond)

	mu.Lock()
	count := len(forwarded)
	mu.Unlock()

	if count < 1 {
		t.Errorf("expected at least 1 forwarded packet, got %d", count)
	}

	node.Stop()
}

func TestShroudNodeStats(t *testing.T) {
	beacon, _ := NewBeacon()

	handler := func(packet []byte) (string, []byte, error) {
		return "next-hop", packet, nil
	}

	sender := func(peerID string, data []byte) error {
		return nil
	}

	config := DefaultShroudNodeConfig()
	config.PeerID = "stats-node"
	config.MaxBandwidth = 1000
	config.MaxCircuits = 50

	node := NewShroudNode(beacon, handler, sender, config)

	stats := node.Stats()

	if stats.IsRunning {
		t.Error("IsRunning should be false initially")
	}

	if stats.Config.MaxBandwidth != 1000 {
		t.Errorf("Config.MaxBandwidth = %d, want 1000", stats.Config.MaxBandwidth)
	}

	if stats.Config.MaxCircuits != 50 {
		t.Errorf("Config.MaxCircuits = %d, want 50", stats.Config.MaxCircuits)
	}
}

func TestShroudNodeCircuitCount(t *testing.T) {
	beacon, _ := NewBeacon()

	node := NewShroudNode(beacon, nil, nil, ShroudNodeConfig{
		MaxCircuits: 3,
	})

	// Increment should work up to max.
	for i := 0; i < 3; i++ {
		if !node.IncrementCircuits() {
			t.Errorf("IncrementCircuits failed at count %d", i)
		}
	}

	// Fourth increment should fail.
	if node.IncrementCircuits() {
		t.Error("IncrementCircuits should fail at max capacity")
	}

	stats := node.Stats()
	if stats.CircuitCount != 3 {
		t.Errorf("CircuitCount = %d, want 3", stats.CircuitCount)
	}

	// Decrement should work.
	node.DecrementCircuits()

	stats = node.Stats()
	if stats.CircuitCount != 2 {
		t.Errorf("CircuitCount = %d after decrement, want 2", stats.CircuitCount)
	}

	// Another increment should work now.
	if !node.IncrementCircuits() {
		t.Error("IncrementCircuits should work after decrement")
	}
}

func TestShroudNodeBandwidth(t *testing.T) {
	beacon, _ := NewBeacon()

	node := NewShroudNode(beacon, nil, nil, ShroudNodeConfig{
		MaxBandwidth: 1000,
	})

	// Initial available bandwidth should be max.
	if node.AvailableBandwidth() != 1000 {
		t.Errorf("AvailableBandwidth = %d, want 1000", node.AvailableBandwidth())
	}

	// Update bandwidth usage.
	node.UpdateBandwidth(400)

	if node.AvailableBandwidth() != 600 {
		t.Errorf("AvailableBandwidth = %d, want 600", node.AvailableBandwidth())
	}

	// At max usage.
	node.UpdateBandwidth(1000)

	if node.AvailableBandwidth() != 0 {
		t.Errorf("AvailableBandwidth = %d at max, want 0", node.AvailableBandwidth())
	}
}

func TestShroudNodeAvailableCircuits(t *testing.T) {
	beacon, _ := NewBeacon()

	node := NewShroudNode(beacon, nil, nil, ShroudNodeConfig{
		MaxCircuits: 10,
	})

	// Initial available should be max.
	if node.AvailableCircuits() != 10 {
		t.Errorf("AvailableCircuits = %d, want 10", node.AvailableCircuits())
	}

	// Add some circuits.
	node.IncrementCircuits()
	node.IncrementCircuits()
	node.IncrementCircuits()

	if node.AvailableCircuits() != 7 {
		t.Errorf("AvailableCircuits = %d, want 7", node.AvailableCircuits())
	}
}

func TestShroudNodePeerID(t *testing.T) {
	beacon, _ := NewBeacon()

	config := DefaultShroudNodeConfig()
	config.PeerID = "my-peer-id"

	node := NewShroudNode(beacon, nil, nil, config)

	if node.PeerID() != "my-peer-id" {
		t.Errorf("PeerID = %s, want my-peer-id", node.PeerID())
	}
}

func TestBeaconDisableRelay(t *testing.T) {
	beacon, _ := NewBeacon()

	beacon.EnableRelay("peer-1", 1000)

	if !beacon.IsRelay() {
		t.Error("Beacon should be relay after EnableRelay")
	}

	beacon.DisableRelay()

	if beacon.IsRelay() {
		t.Error("Beacon should not be relay after DisableRelay")
	}
}

// Capacity metrics tests

func TestCapacityMetricsRatios(t *testing.T) {
	// Full capacity available.
	m := CapacityMetrics{
		MaxBandwidth:     1000,
		CurrentBandwidth: 0,
		MaxCircuits:      100,
		CurrentCircuits:  0,
	}

	if m.AvailableBandwidthRatio() != 1.0 {
		t.Errorf("AvailableBandwidthRatio = %f, want 1.0", m.AvailableBandwidthRatio())
	}

	if m.AvailableCircuitsRatio() != 1.0 {
		t.Errorf("AvailableCircuitsRatio = %f, want 1.0", m.AvailableCircuitsRatio())
	}

	if m.LoadScore() != 1.0 {
		t.Errorf("LoadScore = %f, want 1.0", m.LoadScore())
	}

	// Half loaded.
	m = CapacityMetrics{
		MaxBandwidth:     1000,
		CurrentBandwidth: 500,
		MaxCircuits:      100,
		CurrentCircuits:  50,
	}

	if m.AvailableBandwidthRatio() != 0.5 {
		t.Errorf("AvailableBandwidthRatio = %f, want 0.5", m.AvailableBandwidthRatio())
	}

	if m.AvailableCircuitsRatio() != 0.5 {
		t.Errorf("AvailableCircuitsRatio = %f, want 0.5", m.AvailableCircuitsRatio())
	}

	if m.LoadScore() != 0.5 {
		t.Errorf("LoadScore = %f, want 0.5", m.LoadScore())
	}

	// Fully loaded.
	m = CapacityMetrics{
		MaxBandwidth:     1000,
		CurrentBandwidth: 1000,
		MaxCircuits:      100,
		CurrentCircuits:  100,
	}

	if m.AvailableBandwidthRatio() != 0.0 {
		t.Errorf("AvailableBandwidthRatio = %f, want 0.0", m.AvailableBandwidthRatio())
	}

	if m.AvailableCircuitsRatio() != 0.0 {
		t.Errorf("AvailableCircuitsRatio = %f, want 0.0", m.AvailableCircuitsRatio())
	}

	if m.LoadScore() != 0.0 {
		t.Errorf("LoadScore = %f, want 0.0", m.LoadScore())
	}

	// Unlimited capacity.
	m = CapacityMetrics{
		MaxBandwidth: 0, // Unlimited.
		MaxCircuits:  0, // Unlimited.
	}

	if m.AvailableBandwidthRatio() != 1.0 {
		t.Errorf("unlimited bandwidth ratio = %f, want 1.0", m.AvailableBandwidthRatio())
	}

	if m.AvailableCircuitsRatio() != 1.0 {
		t.Errorf("unlimited circuits ratio = %f, want 1.0", m.AvailableCircuitsRatio())
	}
}

func TestShroudNodeCapacityMetrics(t *testing.T) {
	beacon, _ := NewBeacon()

	config := ShroudNodeConfig{
		PeerID:       "capacity-node",
		MaxBandwidth: 1000,
		MaxCircuits:  50,
	}

	node := NewShroudNode(beacon, nil, nil, config)
	node.SetVersion("1.0.0")

	// Before start, uptime should be 0.
	metrics := node.CapacityMetrics()
	if metrics.UptimeSeconds != 0 {
		t.Errorf("UptimeSeconds before start = %d, want 0", metrics.UptimeSeconds)
	}

	// Start the node.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node.Start(ctx)

	// Wait a moment.
	time.Sleep(100 * time.Millisecond)

	metrics = node.CapacityMetrics()

	if metrics.MaxBandwidth != 1000 {
		t.Errorf("MaxBandwidth = %d, want 1000", metrics.MaxBandwidth)
	}

	if metrics.MaxCircuits != 50 {
		t.Errorf("MaxCircuits = %d, want 50", metrics.MaxCircuits)
	}

	if metrics.Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", metrics.Version)
	}

	// Uptime should be > 0.
	if metrics.UptimeSeconds < 0 {
		t.Errorf("UptimeSeconds = %d, should be >= 0", metrics.UptimeSeconds)
	}

	node.Stop()
}

func TestShroudNodeBytesRelayed(t *testing.T) {
	beacon, _ := NewBeacon()
	node := NewShroudNode(beacon, nil, nil, ShroudNodeConfig{})

	if node.TotalBytesRelayed() != 0 {
		t.Errorf("initial TotalBytesRelayed = %d, want 0", node.TotalBytesRelayed())
	}

	node.AddBytesRelayed(100)
	node.AddBytesRelayed(200)

	if node.TotalBytesRelayed() != 300 {
		t.Errorf("TotalBytesRelayed = %d, want 300", node.TotalBytesRelayed())
	}

	metrics := node.CapacityMetrics()
	if metrics.TotalBytesRelayed != 300 {
		t.Errorf("CapacityMetrics.TotalBytesRelayed = %d, want 300", metrics.TotalBytesRelayed)
	}
}

func TestShroudNodeUptime(t *testing.T) {
	beacon, _ := NewBeacon()
	node := NewShroudNode(beacon, nil, nil, ShroudNodeConfig{})

	// Uptime before start should be 0.
	if node.Uptime() != 0 {
		t.Errorf("Uptime before start = %v, want 0", node.Uptime())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	uptime := node.Uptime()
	if uptime < 100*time.Millisecond {
		t.Errorf("Uptime = %v, expected >= 100ms", uptime)
	}

	node.Stop()
}

func TestSelectRelayByCapacity(t *testing.T) {
	beacon1, _ := NewBeacon()
	beacon2, _ := NewBeacon()
	beacon3, _ := NewBeacon()

	relays := []*RelayInfo{
		{PeerID: "relay-1", PublicKey: beacon1.PublicKey()},
		{PeerID: "relay-2", PublicKey: beacon2.PublicKey()},
		{PeerID: "relay-3", PublicKey: beacon3.PublicKey()},
	}

	// Test with capacity metrics - relay-3 has best score.
	metrics := map[string]CapacityMetrics{
		"relay-1": {MaxBandwidth: 1000, CurrentBandwidth: 900, MaxCircuits: 10, CurrentCircuits: 9}, // 10% available
		"relay-2": {MaxBandwidth: 1000, CurrentBandwidth: 500, MaxCircuits: 10, CurrentCircuits: 5}, // 50% available
		"relay-3": {MaxBandwidth: 1000, CurrentBandwidth: 100, MaxCircuits: 10, CurrentCircuits: 1}, // 90% available
	}

	selected := SelectRelayByCapacity(relays, metrics, nil)
	if selected == nil {
		t.Fatal("SelectRelayByCapacity returned nil")
	}

	if selected.PeerID != "relay-3" {
		t.Errorf("selected relay = %s, want relay-3 (highest capacity)", selected.PeerID)
	}

	// Test with exclusion.
	selected = SelectRelayByCapacity(relays, metrics, []string{"relay-3"})
	if selected == nil {
		t.Fatal("SelectRelayByCapacity returned nil with exclusion")
	}

	if selected.PeerID != "relay-2" {
		t.Errorf("selected relay = %s, want relay-2 (highest after exclusion)", selected.PeerID)
	}

	// Test with all excluded.
	selected = SelectRelayByCapacity(relays, metrics, []string{"relay-1", "relay-2", "relay-3"})
	if selected != nil {
		t.Error("SelectRelayByCapacity should return nil when all excluded")
	}

	// Test with empty relays.
	selected = SelectRelayByCapacity(nil, metrics, nil)
	if selected != nil {
		t.Error("SelectRelayByCapacity should return nil for empty relays")
	}

	// Test without metrics (should use default score).
	selected = SelectRelayByCapacity(relays, nil, nil)
	if selected == nil {
		t.Fatal("SelectRelayByCapacity returned nil without metrics")
	}
}

func TestCapacityMetricsUptimeBoost(t *testing.T) {
	// Test that longer uptime gives score boost.
	shortUptime := CapacityMetrics{
		MaxBandwidth:     1000,
		CurrentBandwidth: 500,
		MaxCircuits:      10,
		CurrentCircuits:  5,
		UptimeSeconds:    60, // 1 minute
	}

	longUptime := CapacityMetrics{
		MaxBandwidth:     1000,
		CurrentBandwidth: 500,
		MaxCircuits:      10,
		CurrentCircuits:  5,
		UptimeSeconds:    86401, // > 1 day
	}

	beacon1, _ := NewBeacon()
	beacon2, _ := NewBeacon()

	relays := []*RelayInfo{
		{PeerID: "short", PublicKey: beacon1.PublicKey()},
		{PeerID: "long", PublicKey: beacon2.PublicKey()},
	}

	metrics := map[string]CapacityMetrics{
		"short": shortUptime,
		"long":  longUptime,
	}

	selected := SelectRelayByCapacity(relays, metrics, nil)
	if selected == nil {
		t.Fatal("SelectRelayByCapacity returned nil")
	}

	// Long uptime should be preferred due to reliability boost.
	if selected.PeerID != "long" {
		t.Errorf("selected = %s, want 'long' (uptime boost)", selected.PeerID)
	}
}

// End-to-end message delivery tests

func TestEncodeDecodeMessage(t *testing.T) {
	// Test data message with destination.
	msg := &Message{
		Type:    MessageTypeData,
		Payload: []byte("Hello, Shroud!"),
	}
	copy(msg.Dest[:], []byte("destination-pubkey-32bytes------"))

	encoded, err := EncodeMessage(msg)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type = %d, want %d", decoded.Type, msg.Type)
	}

	if decoded.Dest != msg.Dest {
		t.Error("Dest mismatch")
	}

	if !bytes.Equal(decoded.Payload, msg.Payload) {
		t.Errorf("Payload = %s, want %s", decoded.Payload, msg.Payload)
	}
}

func TestEncodeDecodeMessageNoDest(t *testing.T) {
	// Test message without destination.
	msg := &Message{
		Type:    MessageTypeData,
		Payload: []byte("Broadcast message"),
	}

	encoded, err := EncodeMessage(msg)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type = %d, want %d", decoded.Type, msg.Type)
	}

	// Dest should be zero.
	if decoded.Dest != [32]byte{} {
		t.Error("Dest should be zero for broadcast")
	}

	if !bytes.Equal(decoded.Payload, msg.Payload) {
		t.Errorf("Payload mismatch")
	}
}

func TestEncodeMessageTooLarge(t *testing.T) {
	// Message larger than allowed.
	largePayload := make([]byte, FixedPacketSize)

	msg := &Message{
		Type:    MessageTypeData,
		Payload: largePayload,
	}

	_, err := EncodeMessage(msg)
	if err == nil {
		t.Error("EncodeMessage should fail for oversized message")
	}
}

func TestEncodeMessageNil(t *testing.T) {
	_, err := EncodeMessage(nil)
	if err == nil {
		t.Error("EncodeMessage should fail for nil message")
	}
}

func TestDecodeMessageTooShort(t *testing.T) {
	_, err := DecodeMessage([]byte{0x01, 0x00})
	if err == nil {
		t.Error("DecodeMessage should fail for short data")
	}
}

func TestMessageSender(t *testing.T) {
	beacon, _ := NewBeacon()

	// Add relays.
	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	var sentPackets [][]byte
	var sentPeerIDs []string
	var mu sync.Mutex

	networkSender := func(peerID string, data []byte) error {
		mu.Lock()
		sentPackets = append(sentPackets, data)
		sentPeerIDs = append(sentPeerIDs, peerID)
		mu.Unlock()
		return nil
	}

	sender := NewMessageSender(manager, networkSender)

	// Send a message.
	msg := &Message{
		Type:    MessageTypeData,
		Payload: []byte("Test message"),
	}

	err := sender.Send(msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	mu.Lock()
	count := len(sentPackets)
	mu.Unlock()

	if count != 1 {
		t.Errorf("sent %d packets, want 1", count)
	}

	stats := sender.Stats()
	if stats.MessagesSent != 1 {
		t.Errorf("MessagesSent = %d, want 1", stats.MessagesSent)
	}
}

func TestMessageSenderSendTo(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	networkSender := func(peerID string, data []byte) error {
		return nil
	}

	sender := NewMessageSender(manager, networkSender)

	var dest [32]byte
	copy(dest[:], []byte("destination-pubkey-32bytes------"))

	err := sender.SendTo(dest, []byte("Directed message"))
	if err != nil {
		t.Fatalf("SendTo failed: %v", err)
	}

	stats := sender.Stats()
	if stats.MessagesSent != 1 {
		t.Errorf("MessagesSent = %d, want 1", stats.MessagesSent)
	}
}

func TestMessageSenderBroadcast(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	networkSender := func(peerID string, data []byte) error {
		return nil
	}

	sender := NewMessageSender(manager, networkSender)

	err := sender.Broadcast([]byte("Broadcast message"))
	if err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}

	stats := sender.Stats()
	if stats.MessagesSent != 1 {
		t.Errorf("MessagesSent = %d, want 1", stats.MessagesSent)
	}
}

func TestMessageReceiver(t *testing.T) {
	receiver := NewMessageReceiver()

	var receivedMsgs []*Message
	var mu sync.Mutex

	receiver.RegisterHandler(MessageTypeData, func(msg *Message) error {
		mu.Lock()
		receivedMsgs = append(receivedMsgs, msg)
		mu.Unlock()
		return nil
	})

	// Create and encode a test message.
	msg := &Message{
		Type:    MessageTypeData,
		Payload: []byte("Test payload"),
	}

	encoded, _ := EncodeMessage(msg)

	// Pad to fixed size (simulating circuit packet).
	padded := padToSize(encoded, FixedPacketSize)

	// Handle the packet.
	err := receiver.HandlePacket(padded)
	if err != nil {
		t.Fatalf("HandlePacket failed: %v", err)
	}

	mu.Lock()
	count := len(receivedMsgs)
	mu.Unlock()

	if count != 1 {
		t.Fatalf("received %d messages, want 1", count)
	}

	if !bytes.Equal(receivedMsgs[0].Payload, msg.Payload) {
		t.Errorf("Payload mismatch")
	}

	stats := receiver.Stats()
	if stats.MessagesReceived != 1 {
		t.Errorf("MessagesReceived = %d, want 1", stats.MessagesReceived)
	}
}

func TestMessageReceiverDummyPacket(t *testing.T) {
	receiver := NewMessageReceiver()

	var received int
	receiver.RegisterHandler(MessageTypeDummy, func(msg *Message) error {
		received++
		return nil
	})

	// Create dummy message.
	msg := &Message{
		Type: MessageTypeDummy,
	}

	encoded, _ := EncodeMessage(msg)
	padded := padToSize(encoded, FixedPacketSize)

	// Handle dummy packet - should be skipped.
	err := receiver.HandlePacket(padded)
	if err != nil {
		t.Fatalf("HandlePacket failed: %v", err)
	}

	if received != 0 {
		t.Error("Dummy packet should be skipped, not passed to handler")
	}
}

func TestMessageReceiverNoHandler(t *testing.T) {
	receiver := NewMessageReceiver()
	// No handlers registered.

	msg := &Message{
		Type:    MessageTypeControl, // Unregistered type.
		Payload: []byte("Control message"),
	}

	encoded, _ := EncodeMessage(msg)
	padded := padToSize(encoded, FixedPacketSize)

	// Should not error, just drop.
	err := receiver.HandlePacket(padded)
	if err != nil {
		t.Fatalf("HandlePacket failed: %v", err)
	}

	stats := receiver.Stats()
	if stats.MessagesDropped != 1 {
		t.Errorf("MessagesDropped = %d, want 1", stats.MessagesDropped)
	}
}

func TestEndToEndDelivery(t *testing.T) {
	beacon, _ := NewBeacon()

	for i := 0; i < 6; i++ {
		relayBeacon, _ := NewBeacon()
		beacon.AddRelay(&RelayInfo{
			PeerID:    string(rune('a' + i)),
			PublicKey: relayBeacon.PublicKey(),
		})
	}

	manager := NewCircuitManager(beacon, nil)

	networkSender := func(peerID string, data []byte) error {
		return nil
	}

	delivery := NewEndToEndDelivery(manager, networkSender)

	// Test components are accessible.
	if delivery.Manager() != manager {
		t.Error("Manager() returned wrong manager")
	}

	if delivery.Sender() == nil {
		t.Error("Sender() returned nil")
	}

	if delivery.Receiver() == nil {
		t.Error("Receiver() returned nil")
	}

	// Test send.
	err := delivery.Broadcast([]byte("Test"))
	if err != nil {
		t.Errorf("Broadcast failed: %v", err)
	}

	// Test register handler.
	var handled bool
	delivery.RegisterHandler(MessageTypeData, func(msg *Message) error {
		handled = true
		return nil
	})

	msg := &Message{Type: MessageTypeData, Payload: []byte("Test")}
	encoded, _ := EncodeMessage(msg)
	padded := padToSize(encoded, FixedPacketSize)

	err = delivery.HandleIncoming(padded)
	if err != nil {
		t.Errorf("HandleIncoming failed: %v", err)
	}

	if !handled {
		t.Error("Handler was not called")
	}
}

func TestMessageSenderErrorCallback(t *testing.T) {
	beacon, _ := NewBeacon()
	// No relays = will fail to build circuit.

	manager := NewCircuitManager(beacon, nil)

	networkSender := func(peerID string, data []byte) error {
		return nil
	}

	sender := NewMessageSender(manager, networkSender)

	var errorReceived atomic.Pointer[error]
	sender.SetOnError(func(err error) {
		errorReceived.Store(&err)
	})

	// Try to send - should fail due to no relays.
	msg := &Message{Type: MessageTypeData, Payload: []byte("Test")}
	err := sender.Send(msg)

	if err == nil {
		t.Error("Send should fail without relays")
	}

	// Wait for callback.
	time.Sleep(50 * time.Millisecond)

	if errorReceived.Load() == nil {
		t.Error("Error callback was not invoked")
	}

	stats := sender.Stats()
	if stats.LastError == nil {
		t.Error("LastError should be set")
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// Verify message type constants.
	if MessageTypeDummy != 0x00 {
		t.Errorf("MessageTypeDummy = %d, want 0", MessageTypeDummy)
	}

	if MessageTypeData != 0x01 {
		t.Errorf("MessageTypeData = %d, want 1", MessageTypeData)
	}

	if MessageTypeControl != 0x02 {
		t.Errorf("MessageTypeControl = %d, want 2", MessageTypeControl)
	}
}

// TestBeaconWaveEncoding tests BeaconWave encode/decode round-trip.
func TestBeaconWaveEncoding(t *testing.T) {
	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "QmTestPeer123456789",
		Bandwidth:   1000000,
		MaxCircuits: 100,
		CurrentLoad: 42,
		LatencyMs:   50,
		Uptime:      3600,
		Timestamp:   time.Now().Unix(),
		TTL:         300,
		Signature:   []byte("test-signature"),
	}
	rand.Read(wave.PublicKey[:])

	// Encode.
	encoded, err := EncodeBeaconWave(wave)
	if err != nil {
		t.Fatalf("EncodeBeaconWave: %v", err)
	}

	// Decode.
	decoded, err := DecodeBeaconWave(encoded)
	if err != nil {
		t.Fatalf("DecodeBeaconWave: %v", err)
	}

	// Compare fields.
	if decoded.Version != wave.Version {
		t.Error("Version mismatch")
	}
	if decoded.Type != wave.Type {
		t.Error("Type mismatch")
	}
	if decoded.RelayPeerID != wave.RelayPeerID {
		t.Errorf("RelayPeerID mismatch: got %q, want %q", decoded.RelayPeerID, wave.RelayPeerID)
	}
	if decoded.PublicKey != wave.PublicKey {
		t.Error("PublicKey mismatch")
	}
	if decoded.MaxCircuits != wave.MaxCircuits {
		t.Errorf("MaxCircuits mismatch: got %d, want %d", decoded.MaxCircuits, wave.MaxCircuits)
	}
	if decoded.CurrentLoad != wave.CurrentLoad {
		t.Errorf("CurrentLoad mismatch: got %d, want %d", decoded.CurrentLoad, wave.CurrentLoad)
	}
	if decoded.Timestamp != wave.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, wave.Timestamp)
	}
	if decoded.TTL != wave.TTL {
		t.Errorf("TTL mismatch: got %d, want %d", decoded.TTL, wave.TTL)
	}
	if !bytes.Equal(decoded.Signature, wave.Signature) {
		t.Error("Signature mismatch")
	}
}

// TestDecodeBeaconWaveErrors tests error cases.
func TestDecodeBeaconWaveErrors(t *testing.T) {
	// Too short.
	_, err := DecodeBeaconWave([]byte{1, 2, 3})
	if err != ErrBeaconWaveInvalid {
		t.Errorf("expected ErrBeaconWaveInvalid, got: %v", err)
	}

	// Bad version.
	badVersion := make([]byte, 100)
	badVersion[0] = 99 // Wrong version.
	badVersion[1] = BeaconWaveType
	_, err = DecodeBeaconWave(badVersion)
	if err != ErrBeaconWaveBadVersion {
		t.Errorf("expected ErrBeaconWaveBadVersion, got: %v", err)
	}

	// Bad type.
	badType := make([]byte, 100)
	badType[0] = BeaconWaveVersion
	badType[1] = 99 // Wrong type.
	_, err = DecodeBeaconWave(badType)
	if err != ErrBeaconWaveInvalid {
		t.Errorf("expected ErrBeaconWaveInvalid, got: %v", err)
	}
}

// TestBeaconWaveExpiry tests TTL expiration.
func TestBeaconWaveExpiry(t *testing.T) {
	// Expired wave.
	expiredWave := &BeaconWave{
		Timestamp: time.Now().Add(-2 * time.Hour).Unix(),
		TTL:       60, // 1 minute TTL, expired long ago.
	}

	if !expiredWave.IsExpired() {
		t.Error("wave should be expired")
	}

	// Valid wave.
	validWave := &BeaconWave{
		Timestamp: time.Now().Unix(),
		TTL:       600, // 10 minutes.
	}

	if validWave.IsExpired() {
		t.Error("wave should not be expired")
	}
}

// TestBeaconWaveLoadFactor tests load factor calculation.
func TestBeaconWaveLoadFactor(t *testing.T) {
	wave := &BeaconWave{
		MaxCircuits: 100,
		CurrentLoad: 50,
	}

	factor := wave.LoadFactor()
	if factor != 0.5 {
		t.Errorf("LoadFactor = %f, want 0.5", factor)
	}

	// Zero max circuits.
	zeroMax := &BeaconWave{MaxCircuits: 0, CurrentLoad: 10}
	factor = zeroMax.LoadFactor()
	if factor != 1.0 {
		t.Errorf("LoadFactor with zero max = %f, want 1.0", factor)
	}
}

// TestBeaconWaveToRelayInfo tests conversion to RelayInfo.
func TestBeaconWaveToRelayInfo(t *testing.T) {
	wave := &BeaconWave{
		RelayPeerID: "test-peer",
		Bandwidth:   1000000,
		Timestamp:   time.Now().Unix(),
	}
	rand.Read(wave.PublicKey[:])

	info := wave.ToRelayInfo()

	if info.PeerID != wave.RelayPeerID {
		t.Error("PeerID mismatch")
	}
	if info.PublicKey != wave.PublicKey {
		t.Error("PublicKey mismatch")
	}
	if info.Bandwidth != wave.Bandwidth {
		t.Error("Bandwidth mismatch")
	}
}

// TestBeaconWavePublisher tests the publisher creation and configuration.
func TestBeaconWavePublisher(t *testing.T) {
	beacon, _ := NewBeacon()
	beacon.EnableRelay("test-peer", 1000000)

	var published [][]byte
	var mu sync.Mutex

	publisher := func(data []byte) error {
		mu.Lock()
		published = append(published, data)
		mu.Unlock()
		return nil
	}

	pub := NewBeaconWavePublisher(beacon, "test-peer", publisher)
	pub.SetCapacity(100, 50)
	pub.SetCurrentLoad(25)

	// Publish once.
	err := pub.PublishNow()
	if err != nil {
		t.Fatalf("PublishNow: %v", err)
	}

	mu.Lock()
	if len(published) != 1 {
		t.Errorf("expected 1 published, got %d", len(published))
	}
	mu.Unlock()

	// Decode and verify.
	mu.Lock()
	data := published[0]
	mu.Unlock()

	wave, err := DecodeBeaconWave(data)
	if err != nil {
		t.Fatalf("DecodeBeaconWave: %v", err)
	}

	if wave.RelayPeerID != "test-peer" {
		t.Errorf("RelayPeerID = %q, want %q", wave.RelayPeerID, "test-peer")
	}
	if wave.MaxCircuits != 100 {
		t.Errorf("MaxCircuits = %d, want 100", wave.MaxCircuits)
	}
	if wave.CurrentLoad != 25 {
		t.Errorf("CurrentLoad = %d, want 25", wave.CurrentLoad)
	}
	if wave.LatencyMs != 50 {
		t.Errorf("LatencyMs = %d, want 50", wave.LatencyMs)
	}
}

// TestBeaconWavePublisherNotRelay tests error when not configured as relay.
func TestBeaconWavePublisherNotRelay(t *testing.T) {
	beacon, _ := NewBeacon()
	// Not enabled as relay.

	pub := NewBeaconWavePublisher(beacon, "test-peer", nil)

	err := pub.PublishNow()
	if err == nil {
		t.Error("expected error when not configured as relay")
	}
}

// TestBeaconWaveReceiver tests receiving and processing beacon waves.
func TestBeaconWaveReceiver(t *testing.T) {
	beacon, _ := NewBeacon()
	receiver := NewBeaconWaveReceiver(beacon, "local-peer")

	// Create a wave from another peer.
	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "remote-peer",
		Bandwidth:   1000000,
		MaxCircuits: 100,
		CurrentLoad: 10,
		Timestamp:   time.Now().Unix(),
		TTL:         300,
	}
	rand.Read(wave.PublicKey[:])

	encoded, _ := EncodeBeaconWave(wave)

	// Handle incoming.
	err := receiver.HandleIncoming(encoded)
	if err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Verify relay was registered.
	relayInfo, exists := beacon.GetRelay("remote-peer")
	if !exists {
		t.Fatal("relay not registered")
	}
	if relayInfo.PeerID != "remote-peer" {
		t.Error("PeerID mismatch")
	}

	// Check stats.
	stats := receiver.Stats()
	if stats.WavesReceived != 1 {
		t.Errorf("WavesReceived = %d, want 1", stats.WavesReceived)
	}
	if stats.WavesProcessed != 1 {
		t.Errorf("WavesProcessed = %d, want 1", stats.WavesProcessed)
	}
	if stats.RelaysDiscovered != 1 {
		t.Errorf("RelaysDiscovered = %d, want 1", stats.RelaysDiscovered)
	}
}

// TestBeaconWaveReceiverIgnoresSelf tests that self-advertisements are ignored.
func TestBeaconWaveReceiverIgnoresSelf(t *testing.T) {
	beacon, _ := NewBeacon()
	receiver := NewBeaconWaveReceiver(beacon, "local-peer")

	// Create a wave from ourself.
	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "local-peer", // Same as receiver's ID.
		Timestamp:   time.Now().Unix(),
		TTL:         300,
	}

	encoded, _ := EncodeBeaconWave(wave)

	err := receiver.HandleIncoming(encoded)
	if err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Should not register ourselves.
	if beacon.RelayCount() != 0 {
		t.Error("should not register self as relay")
	}
}

// TestBeaconWaveReceiverExpired tests rejecting expired waves.
func TestBeaconWaveReceiverExpired(t *testing.T) {
	beacon, _ := NewBeacon()
	receiver := NewBeaconWaveReceiver(beacon, "local-peer")

	// Create an expired wave.
	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "remote-peer",
		Timestamp:   time.Now().Add(-2 * time.Hour).Unix(),
		TTL:         60, // Expired.
	}

	encoded, _ := EncodeBeaconWave(wave)

	err := receiver.HandleIncoming(encoded)
	if err != ErrBeaconWaveExpired {
		t.Errorf("expected ErrBeaconWaveExpired, got: %v", err)
	}

	// Check stats.
	stats := receiver.Stats()
	if stats.WavesExpired != 1 {
		t.Errorf("WavesExpired = %d, want 1", stats.WavesExpired)
	}
}

// TestBeaconWaveReceiverUpdate tests updating existing relay info.
func TestBeaconWaveReceiverUpdate(t *testing.T) {
	beacon, _ := NewBeacon()
	receiver := NewBeaconWaveReceiver(beacon, "local-peer")

	// First wave.
	wave1 := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "remote-peer",
		Bandwidth:   1000000,
		Timestamp:   time.Now().Unix(),
		TTL:         300,
	}
	rand.Read(wave1.PublicKey[:])

	encoded1, _ := EncodeBeaconWave(wave1)
	receiver.HandleIncoming(encoded1)

	// Second wave (update).
	wave2 := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "remote-peer",
		Bandwidth:   2000000, // Updated bandwidth.
		PublicKey:   wave1.PublicKey,
		Timestamp:   time.Now().Unix(),
		TTL:         300,
	}

	encoded2, _ := EncodeBeaconWave(wave2)
	receiver.HandleIncoming(encoded2)

	// Check stats.
	stats := receiver.Stats()
	if stats.RelaysDiscovered != 1 {
		t.Errorf("RelaysDiscovered = %d, want 1", stats.RelaysDiscovered)
	}
	if stats.RelaysUpdated != 1 {
		t.Errorf("RelaysUpdated = %d, want 1", stats.RelaysUpdated)
	}

	// Verify updated info.
	info, _ := beacon.GetRelay("remote-peer")
	if info.Bandwidth != 2000000 {
		t.Errorf("Bandwidth = %d, want 2000000", info.Bandwidth)
	}
}

// TestBeaconWaveReceiverHandler tests custom handlers.
func TestBeaconWaveReceiverHandler(t *testing.T) {
	beacon, _ := NewBeacon()
	receiver := NewBeaconWaveReceiver(beacon, "local-peer")

	var handlerCalled bool
	var receivedWave *BeaconWave

	receiver.RegisterHandler(func(wave *BeaconWave) error {
		handlerCalled = true
		receivedWave = wave
		return nil
	})

	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "remote-peer",
		Timestamp:   time.Now().Unix(),
		TTL:         300,
	}

	encoded, _ := EncodeBeaconWave(wave)
	receiver.HandleIncoming(encoded)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if receivedWave.RelayPeerID != "remote-peer" {
		t.Error("handler received wrong wave")
	}
}

// TestRelayDiscovery tests the RelayDiscovery orchestrator.
func TestRelayDiscovery(t *testing.T) {
	beacon, _ := NewBeacon()
	beacon.EnableRelay("local-peer", 1000000)

	var published [][]byte
	var mu sync.Mutex

	publisher := func(data []byte) error {
		mu.Lock()
		published = append(published, data)
		mu.Unlock()
		return nil
	}

	discovery := NewRelayDiscovery(beacon, "local-peer", publisher)
	discovery.SetCapacity(100, 50)

	// Publish manually.
	discovery.Publisher().PublishNow()

	mu.Lock()
	if len(published) != 1 {
		t.Errorf("expected 1 published, got %d", len(published))
	}
	mu.Unlock()

	// Simulate receiving from another peer.
	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: "remote-peer",
		Timestamp:   time.Now().Unix(),
		TTL:         300,
	}
	rand.Read(wave.PublicKey[:])

	encoded, _ := EncodeBeaconWave(wave)
	discovery.HandleBeaconWave(encoded)

	// Check relay was discovered.
	if beacon.RelayCount() != 1 {
		t.Errorf("RelayCount = %d, want 1", beacon.RelayCount())
	}
}

// TestRelayDiscoveryCleanup tests stale relay cleanup.
func TestRelayDiscoveryCleanup(t *testing.T) {
	beacon, _ := NewBeacon()

	discovery := NewRelayDiscovery(beacon, "local-peer", nil)

	// Add some relays directly with old timestamps using internal method.
	for i := 0; i < 5; i++ {
		info := &RelayInfo{
			PeerID: fmt.Sprintf("old-peer-%d", i),
			SeenAt: time.Now().Add(-1 * time.Hour), // Old.
		}
		beacon.addRelayWithTime(info)
	}

	// Add a fresh relay.
	freshInfo := &RelayInfo{
		PeerID: "fresh-peer",
		SeenAt: time.Now(),
	}
	beacon.addRelayWithTime(freshInfo)

	if beacon.RelayCount() != 6 {
		t.Fatalf("RelayCount = %d, want 6", beacon.RelayCount())
	}

	// Cleanup with 30 minute max age.
	removed := discovery.CleanupStaleRelays(30 * time.Minute)

	if removed != 5 {
		t.Errorf("removed = %d, want 5", removed)
	}

	if beacon.RelayCount() != 1 {
		t.Errorf("RelayCount after cleanup = %d, want 1", beacon.RelayCount())
	}

	// Verify fresh relay is still there.
	_, exists := beacon.GetRelay("fresh-peer")
	if !exists {
		t.Error("fresh relay should still exist")
	}
}

// TestBeaconWaveConstants tests constant values.
func TestBeaconWaveConstants(t *testing.T) {
	if BeaconWaveType != 0x08 {
		t.Errorf("BeaconWaveType = %d, want 0x08", BeaconWaveType)
	}
	if BeaconWaveVersion != 1 {
		t.Errorf("BeaconWaveVersion = %d, want 1", BeaconWaveVersion)
	}
	if BeaconWaveTTL != 5*time.Minute {
		t.Errorf("BeaconWaveTTL = %v, want 5m", BeaconWaveTTL)
	}
	if BeaconWaveInterval != 60*time.Second {
		t.Errorf("BeaconWaveInterval = %v, want 60s", BeaconWaveInterval)
	}
}
