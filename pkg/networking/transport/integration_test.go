// Package transport integration tests for Tor and I2P transport adapters.
// These tests exercise reachability checks and validate host creation scenarios
// per PLAN.md §5.8. Full dial/listen lifecycle tests require running Tor/I2P daemons.
//
//go:build integration

package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opd-ai/murmur/pkg/networking/transport/diagnostics"
)

// TestTorReachability tests Tor daemon reachability check.
func TestTorReachability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := diagnostics.CheckTor(ctx, "127.0.0.1:9051")
	if !status.Reachable {
		t.Fatalf("daemon not reachable: %v", status.Error)
	}

	t.Logf("Tor daemon reachable: latency=%dms", status.LatencyMs)
	assert.True(t, status.Reachable)
	assert.Empty(t, status.Error)
}

// TestI2PReachability tests I2P SAM bridge reachability check.
func TestI2PReachability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := diagnostics.CheckI2P(ctx, "127.0.0.1:7656")
	if !status.Reachable {
		t.Fatalf("SAM bridge not reachable: %v", status.Error)
	}

	t.Logf("I2P SAM bridge reachable: latency=%dms", status.LatencyMs)
	assert.True(t, status.Reachable)
	assert.Empty(t, status.Error)
}

// TestHostCreationWithTor tests libp2p host creation with Tor transport.
func TestHostCreationWithTor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg := newIntegrationHostConfig(t)
	cfg.EnableTor = true
	cfg.TorControlAddr = "127.0.0.1:9051"
	cfg.EnableI2P = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("host creation failed (Tor unavailable?): %v", err)
	}
	defer h.Close()

	addrs := h.Addrs()
	require.NotEmpty(t, addrs, "host should have addresses")

	hasOnion := false
	for _, addr := range addrs {
		for _, p := range addr.Protocols() {
			if p.Name == "onion3" {
				hasOnion = true
				t.Logf("listening on: %s", addr)
			}
		}
	}
	assert.True(t, hasOnion, "host should have onion3 address when Tor enabled")
}

// TestHostCreationWithI2P tests libp2p host creation with I2P transport.
func TestHostCreationWithI2P(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg := newIntegrationHostConfig(t)
	cfg.EnableTor = false
	cfg.EnableI2P = true
	cfg.I2PSAMAddr = "127.0.0.1:7656"

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("host creation failed (I2P unavailable?): %v", err)
	}
	defer h.Close()

	addrs := h.Addrs()
	require.NotEmpty(t, addrs, "host should have addresses")

	hasGarlic := false
	for _, addr := range addrs {
		for _, p := range addr.Protocols() {
			if p.Name == "garlic64" {
				hasGarlic = true
				t.Logf("listening on: %s", addr)
			}
		}
	}
	assert.True(t, hasGarlic, "host should have garlic64 address when I2P enabled")
}

// TestHostCreationWithBoth tests host creation with both Tor and I2P.
func TestHostCreationWithBoth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg := newIntegrationHostConfig(t)
	cfg.EnableTor = true
	cfg.TorControlAddr = "127.0.0.1:9051"
	cfg.EnableI2P = true
	cfg.I2PSAMAddr = "127.0.0.1:7656"

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("host creation failed (Tor/I2P unavailable?): %v", err)
	}
	defer h.Close()

	addrs := h.Addrs()
	require.NotEmpty(t, addrs, "host should have addresses")

	hasOnion, hasGarlic, hasClearnet := false, false, false
	for _, addr := range addrs {
		for _, p := range addr.Protocols() {
			switch p.Name {
			case "onion3":
				hasOnion = true
			case "garlic64":
				hasGarlic = true
			case "tcp", "quic-v1":
				hasClearnet = true
			}
		}
	}

	assert.True(t, hasOnion, "should have onion3 address")
	assert.True(t, hasGarlic, "should have garlic64 address")
	assert.True(t, hasClearnet, "should have clearnet addresses")
	t.Logf("multi-transport: clearnet=%v tor=%v i2p=%v", hasClearnet, hasOnion, hasGarlic)
}

// TestFallbackToClearnet verifies fail-fast when transports unavailable.
func TestFallbackToClearnet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := newIntegrationHostConfig(t)
	cfg.EnableTor = true
	cfg.EnableI2P = true
	cfg.TorControlAddr = "127.0.0.1:9999" // Unreachable
	cfg.I2PSAMAddr = "127.0.0.1:9998"     // Unreachable

	_, err := NewHost(ctx, cfg)
	require.Error(t, err, "should fail when required transports unavailable")
	t.Logf("expected failure: %v", err)
}

// TestMultiaddrProtocols validates onion3 and garlic64 multiaddr parsing.
func TestMultiaddrProtocols(t *testing.T) {
	onionMaddr, err := ma.NewMultiaddr("/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001")
	require.NoError(t, err)

	hasOnion := false
	for _, p := range onionMaddr.Protocols() {
		if p.Name == "onion3" {
			hasOnion = true
		}
	}
	assert.True(t, hasOnion, "should parse onion3 protocol")

	garlicDest := base64.StdEncoding.EncodeToString(make([]byte, 387))
	garlicMaddr, err := ma.NewMultiaddr("/garlic64/" + garlicDest)
	require.NoError(t, err)

	hasGarlic := false
	for _, p := range garlicMaddr.Protocols() {
		if p.Name == "garlic64" {
			hasGarlic = true
		}
	}
	assert.True(t, hasGarlic, "should parse garlic64 protocol")
}

// TestAnonymityTransportDiagnosticsIntegration tests CheckAll function.
func TestAnonymityTransportDiagnosticsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statuses, err := diagnostics.CheckAll(ctx, true, "127.0.0.1:9051", true, "127.0.0.1:7656")
	if err != nil {
		t.Fatalf("anonymity transports unavailable: %v", err)
	}

	require.NotEmpty(t, statuses, "should return status for each transport")

	for _, status := range statuses {
		t.Logf("transport %s: reachable=%v latency=%dms", status.Name, status.Reachable, status.LatencyMs)
	}
}

func newIntegrationHostConfig(t *testing.T) Config {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false
	return cfg
}
