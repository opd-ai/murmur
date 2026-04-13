package relay

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestNATTypeString(t *testing.T) {
	tests := []struct {
		natType NATType
		want    string
	}{
		{NATTypeUnknown, "unknown"},
		{NATTypePublic, "public"},
		{NATTypeCone, "cone"},
		{NATTypeSymmetric, "symmetric"},
	}

	for _, tt := range tests {
		if got := tt.natType.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.natType, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.EnableRelay {
		t.Error("EnableRelay should be true by default")
	}
	if !cfg.EnableHolePunch {
		t.Error("EnableHolePunch should be true by default")
	}
	if cfg.RelayOnly {
		t.Error("RelayOnly should be false by default")
	}
	if cfg.AutoNATProbeInterval != 30*time.Second {
		t.Errorf("AutoNATProbeInterval = %v, want 30s", cfg.AutoNATProbeInterval)
	}
}

func TestHostOptions(t *testing.T) {
	cfg := DefaultConfig()
	opts := HostOptions(cfg)

	// Should return at least 3 options (relay, holepunch, autonat)
	if len(opts) < 3 {
		t.Errorf("HostOptions returned %d options, want >= 3", len(opts))
	}
}

func TestTraverserBasic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test host
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.DisableRelay(),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	// Create traverser
	traverser := New(h)

	// Check initial state
	if traverser.NATType() != NATTypeUnknown {
		t.Errorf("NATType = %v, want unknown", traverser.NATType())
	}

	// Set NAT type
	traverser.SetNATType(NATTypeCone)
	if traverser.NATType() != NATTypeCone {
		t.Errorf("NATType = %v, want cone", traverser.NATType())
	}

	// Add relays
	testRelay := peer.AddrInfo{ID: h.ID()} // Use self as dummy relay
	traverser.AddRelays([]peer.AddrInfo{testRelay})

	relays := traverser.Relays()
	if len(relays) != 1 {
		t.Errorf("Relays count = %d, want 1", len(relays))
	}

	// Verify no reservation initially
	if traverser.HasReservation(h.ID()) {
		t.Error("HasReservation should be false initially")
	}

	_ = ctx // Used for timeout
}

func TestBuildRelayAddr(t *testing.T) {
	// Generate two random peer IDs
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey1, _ := crypto.UnmarshalEd25519PrivateKey(priv1)
	relayID, _ := peer.IDFromPrivateKey(libp2pKey1)

	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey2, _ := crypto.UnmarshalEd25519PrivateKey(priv2)
	targetID, _ := peer.IDFromPrivateKey(libp2pKey2)

	addr, err := buildRelayAddr(relayID, targetID)
	if err != nil {
		t.Fatalf("buildRelayAddr failed: %v", err)
	}

	// Verify the address contains the expected components
	addrStr := addr.String()
	if addrStr == "" {
		t.Error("buildRelayAddr returned empty address")
	}

	// Should contain p2p-circuit
	protocols := addr.Protocols()
	hasCircuit := false
	for _, p := range protocols {
		if p.Name == "p2p-circuit" {
			hasCircuit = true
			break
		}
	}
	if !hasCircuit {
		t.Errorf("Address %s missing p2p-circuit protocol", addrStr)
	}
}

func TestTraverserConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)

	// Concurrent reads and writes
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			traverser.SetNATType(NATTypeCone)
			traverser.AddRelays([]peer.AddrInfo{{ID: h.ID()}})
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		_ = traverser.NATType()
		_ = traverser.Relays()
		_ = traverser.HasReservation(h.ID())
	}

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("Timeout waiting for concurrent operations")
	}
}

func TestSetHolePunchService(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)

	// Setting nil should work without panic.
	traverser.SetHolePunchService(nil)

	// Reading when service is nil.
	traverser.mu.RLock()
	hps := traverser.hpService
	traverser.mu.RUnlock()
	if hps != nil {
		t.Error("Expected hpService to be nil")
	}
}

func TestDirectConnectNoService(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)
	ctx := context.Background()

	// Generate a random target peer ID.
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey2, _ := crypto.UnmarshalEd25519PrivateKey(priv2)
	targetID, _ := peer.IDFromPrivateKey(libp2pKey2)

	// DirectConnect without hole punch service should fail.
	err = traverser.DirectConnect(ctx, targetID)
	if err == nil {
		t.Error("DirectConnect should fail without hole punch service")
	}
}

func TestConnectViaRelayNoRelays(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)

	// Generate a random target peer.
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey2, _ := crypto.UnmarshalEd25519PrivateKey(priv2)
	targetID, _ := peer.IDFromPrivateKey(libp2pKey2)

	target := peer.AddrInfo{ID: targetID}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// No relays added, so should return nil (no error, but also no connection).
	err = traverser.ConnectViaRelay(ctx, target)
	// With no relays, the loop doesn't execute, lastErr is nil.
	if err != nil {
		t.Errorf("ConnectViaRelay with no relays returned error: %v", err)
	}
}

func TestConnectViaRelayFailedConnect(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)

	// Generate fake relay and target peer IDs.
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey2, _ := crypto.UnmarshalEd25519PrivateKey(priv2)
	relayID, _ := peer.IDFromPrivateKey(libp2pKey2)

	_, priv3, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey3, _ := crypto.UnmarshalEd25519PrivateKey(priv3)
	targetID, _ := peer.IDFromPrivateKey(libp2pKey3)

	// Add relay with unreachable address.
	relay := peer.AddrInfo{ID: relayID}
	traverser.AddRelays([]peer.AddrInfo{relay})

	target := peer.AddrInfo{ID: targetID}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Should fail to connect to relay (no addresses).
	err = traverser.ConnectViaRelay(ctx, target)
	if err == nil {
		t.Error("ConnectViaRelay should fail when relay has no addresses")
	}
}

func TestMakeReservationFailedConnect(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)

	// Generate fake relay peer ID with no addresses.
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey2, _ := crypto.UnmarshalEd25519PrivateKey(priv2)
	relayID, _ := peer.IDFromPrivateKey(libp2pKey2)

	relay := peer.AddrInfo{ID: relayID}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Should fail to connect to relay (no addresses).
	err = traverser.MakeReservation(ctx, relay)
	if err == nil {
		t.Error("MakeReservation should fail when relay has no addresses")
	}

	// Reservation should not exist.
	if traverser.HasReservation(relayID) {
		t.Error("HasReservation should be false after failed MakeReservation")
	}
}

func TestHostOptionsDisabled(t *testing.T) {
	// Test with relay disabled.
	cfg := Config{
		EnableRelay:     false,
		EnableHolePunch: false,
		RelayOnly:       false,
	}
	opts := HostOptions(cfg)

	// Should only have AutoNAT option.
	if len(opts) != 1 {
		t.Errorf("HostOptions returned %d options, want 1 (just AutoNAT)", len(opts))
	}
}

func TestRelaysReturnsDefensiveCopy(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(priv)

	h, err := libp2p.New(
		libp2p.Identity(libp2pKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("libp2p.New failed: %v", err)
	}
	defer h.Close()

	traverser := New(h)
	traverser.AddRelays([]peer.AddrInfo{{ID: h.ID()}})

	relays1 := traverser.Relays()
	relays2 := traverser.Relays()

	// Modifying returned slice should not affect internal state.
	if len(relays1) > 0 {
		relays1[0] = peer.AddrInfo{}
	}

	// Original should still be intact.
	relays3 := traverser.Relays()
	if len(relays3) != 1 {
		t.Error("Relays() should return defensive copy")
	}

	// Verify they're different slices.
	if &relays1[0] == &relays2[0] {
		t.Error("Relays() should return new slice each time")
	}
}
