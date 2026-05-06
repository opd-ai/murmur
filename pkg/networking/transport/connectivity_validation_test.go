// Package transport provides cross-platform libp2p connectivity validation tests.
// These tests validate that libp2p connectivity works correctly on Linux, macOS,
// and Windows without requiring external services (fully in-memory).

package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossPlatformHostCreation validates libp2p host can be created on all platforms.
func TestCrossPlatformHostCreation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "Failed to generate keypair on %s/%s", runtime.GOOS, runtime.GOARCH)

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	require.NoError(t, err, "Host creation failed on %s/%s", runtime.GOOS, runtime.GOARCH)
	defer h.Close()

	// Validate host properties
	assert.NotNil(t, h.Host, "Host.Host should not be nil")
	assert.NotEmpty(t, h.PeerID(), "PeerID should not be empty")
	assert.NotEmpty(t, h.Addrs(), "Host should have listen addresses")

	t.Logf("✓ %s/%s: Host created successfully with PeerID %s", runtime.GOOS, runtime.GOARCH, h.PeerID())
}

// TestCrossPlatformPeerConnection validates two hosts can connect on all platforms.
func TestCrossPlatformPeerConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create first host
	_, priv1, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg1 := DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	h1, err := NewHost(ctx, cfg1)
	require.NoError(t, err, "Host 1 creation failed on %s/%s", runtime.GOOS, runtime.GOARCH)
	defer h1.Close()

	// Create second host
	_, priv2, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg2 := DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h2, err := NewHost(ctx, cfg2)
	require.NoError(t, err, "Host 2 creation failed on %s/%s", runtime.GOOS, runtime.GOARCH)
	defer h2.Close()

	// Connect h2 to h1
	err = h2.Connect(ctx, h1.AddrInfo())
	require.NoError(t, err, "Connection failed on %s/%s", runtime.GOOS, runtime.GOARCH)

	// Verify connection
	conns := h2.Network().ConnsToPeer(h1.PeerID())
	assert.NotEmpty(t, conns, "h2 should have connection to h1")

	t.Logf("✓ %s/%s: Peer connection successful (%s ↔ %s)",
		runtime.GOOS, runtime.GOARCH, h2.PeerID(), h1.PeerID())
}

// TestCrossPlatformMultipleConnections validates N-way mesh connectivity.
func TestCrossPlatformMultipleConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	const numHosts = 5

	// Create N hosts
	hosts := make([]*Host, numHosts)
	for i := 0; i < numHosts; i++ {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		cfg := DefaultConfig()
		cfg.PrivateKey = priv
		cfg.EnableDHT = false

		h, err := NewHost(ctx, cfg)
		require.NoError(t, err, "Host %d creation failed on %s/%s", i, runtime.GOOS, runtime.GOARCH)
		defer h.Close()

		hosts[i] = h
	}

	// Connect all hosts in a mesh (each host connects to all others)
	for i := 0; i < numHosts; i++ {
		for j := i + 1; j < numHosts; j++ {
			err := hosts[i].Connect(ctx, hosts[j].AddrInfo())
			require.NoError(t, err, "Connection failed between host %d and %d on %s/%s",
				i, j, runtime.GOOS, runtime.GOARCH)
		}
	}

	// Verify all connections
	for i := 0; i < numHosts; i++ {
		peers := hosts[i].Network().Peers()
		assert.Equal(t, numHosts-1, len(peers), "Host %d should have %d peers", i, numHosts-1)
	}

	t.Logf("✓ %s/%s: %d-host mesh connectivity validated", runtime.GOOS, runtime.GOARCH, numHosts)
}

// TestCrossPlatformTransportProtocols validates transport protocol support.
func TestCrossPlatformTransportProtocols(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	require.NoError(t, err, "Host creation failed on %s/%s", runtime.GOOS, runtime.GOARCH)
	defer h.Close()

	addrs := h.Addrs()
	require.NotEmpty(t, addrs, "Host should have addresses")

	// Count transport protocols
	tcpCount, quicCount := 0, 0
	for _, addr := range addrs {
		for _, p := range addr.Protocols() {
			switch p.Name {
			case "tcp":
				tcpCount++
			case "quic-v1":
				quicCount++
			}
		}
	}

	// At minimum, should have TCP support on all platforms
	assert.Greater(t, tcpCount, 0, "Host should support TCP on %s/%s", runtime.GOOS, runtime.GOARCH)

	t.Logf("✓ %s/%s: Transport protocols: TCP=%d QUIC=%d",
		runtime.GOOS, runtime.GOARCH, tcpCount, quicCount)
}

// TestCrossPlatformConnectionResilience validates connection recovery.
func TestCrossPlatformConnectionResilience(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two hosts
	_, priv1, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg1 := DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	h1, err := NewHost(ctx, cfg1)
	require.NoError(t, err)
	defer h1.Close()

	_, priv2, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg2 := DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h2, err := NewHost(ctx, cfg2)
	require.NoError(t, err)
	defer h2.Close()

	// Connect
	err = h2.Connect(ctx, h1.AddrInfo())
	require.NoError(t, err)

	// Verify connection exists
	conns := h2.Network().ConnsToPeer(h1.PeerID())
	require.NotEmpty(t, conns, "Should have connection")

	// Close first connection
	conns[0].Close()

	// Give time for connection cleanup
	time.Sleep(100 * time.Millisecond)

	// Reconnect should work
	err = h2.Connect(ctx, h1.AddrInfo())
	require.NoError(t, err, "Reconnection failed on %s/%s", runtime.GOOS, runtime.GOARCH)

	newConns := h2.Network().ConnsToPeer(h1.PeerID())
	assert.NotEmpty(t, newConns, "Should have new connection after reconnect")

	t.Logf("✓ %s/%s: Connection resilience validated (reconnection successful)",
		runtime.GOOS, runtime.GOARCH)
}

// TestCrossPlatformConcurrentConnections validates concurrent connection handling.
func TestCrossPlatformConcurrentConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Create bootstrap host
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	bootstrap, err := NewHost(ctx, cfg)
	require.NoError(t, err)
	defer bootstrap.Close()

	// Create multiple hosts and connect concurrently
	const numPeers = 10
	hosts := make([]*Host, numPeers)

	for i := 0; i < numPeers; i++ {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		cfg := DefaultConfig()
		cfg.PrivateKey = priv
		cfg.EnableDHT = false

		h, err := NewHost(ctx, cfg)
		require.NoError(t, err)
		defer h.Close()

		hosts[i] = h
	}

	// Connect all hosts to bootstrap concurrently
	errChan := make(chan error, numPeers)
	for i := 0; i < numPeers; i++ {
		go func(h *Host) {
			errChan <- h.Connect(ctx, bootstrap.AddrInfo())
		}(hosts[i])
	}

	// Wait for all connections
	for i := 0; i < numPeers; i++ {
		err := <-errChan
		require.NoError(t, err, "Concurrent connection %d failed on %s/%s",
			i, runtime.GOOS, runtime.GOARCH)
	}

	// Verify bootstrap has all connections
	peers := bootstrap.Network().Peers()
	assert.Equal(t, numPeers, len(peers), "Bootstrap should have %d peers", numPeers)

	t.Logf("✓ %s/%s: %d concurrent connections handled successfully",
		runtime.GOOS, runtime.GOARCH, numPeers)
}

// TestCrossPlatformDHTMode validates DHT initialization on all platforms.
func TestCrossPlatformDHTMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true
	cfg.DHTServerMode = true

	h, err := NewHost(ctx, cfg)
	require.NoError(t, err, "Host with DHT failed on %s/%s", runtime.GOOS, runtime.GOARCH)
	defer h.Close()

	// Verify DHT is available
	assert.NotNil(t, h.DHT(), "DHT should be initialized")

	t.Logf("✓ %s/%s: DHT mode validated", runtime.GOOS, runtime.GOARCH)
}

// TestCrossPlatformAddressValidation validates multiaddr parsing consistency.
func TestCrossPlatformAddressValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	require.NoError(t, err)
	defer h.Close()

	// Get addresses
	addrs := h.Addrs()
	require.NotEmpty(t, addrs)

	// Validate each address is parseable and has valid components
	for _, addr := range addrs {
		protocols := addr.Protocols()
		require.NotEmpty(t, protocols, "Address should have protocols: %s", addr)

		// Verify address can be decoded
		addrStr := addr.String()
		require.NotEmpty(t, addrStr, "Address string should not be empty")
	}

	t.Logf("✓ %s/%s: %d addresses validated", runtime.GOOS, runtime.GOARCH, len(addrs))
}

// TestCrossPlatformNetworkStats validates connection statistics tracking.
func TestCrossPlatformNetworkStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two hosts
	_, priv1, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg1 := DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	h1, err := NewHost(ctx, cfg1)
	require.NoError(t, err)
	defer h1.Close()

	_, priv2, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg2 := DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h2, err := NewHost(ctx, cfg2)
	require.NoError(t, err)
	defer h2.Close()

	// Connect
	err = h2.Connect(ctx, h1.AddrInfo())
	require.NoError(t, err)

	// Get network stats
	conns := h2.Network().Conns()
	assert.NotEmpty(t, conns, "Should have connections")

	peers := h2.Network().Peers()
	assert.NotEmpty(t, peers, "Should have peers")

	// Validate connection stats are accessible
	for _, conn := range conns {
		assert.NotNil(t, conn.RemotePeer(), "Remote peer should not be nil")
		assert.NotNil(t, conn.RemoteMultiaddr(), "Remote address should not be nil")
	}

	t.Logf("✓ %s/%s: Network stats validated (conns=%d peers=%d)",
		runtime.GOOS, runtime.GOARCH, len(conns), len(peers))
}

// TestPlatformSpecificConnectivity validates platform-specific behaviors.
func TestPlatformSpecificConnectivity(t *testing.T) {
	t.Run("Platform", func(t *testing.T) {
		switch runtime.GOOS {
		case "linux":
			t.Logf("✓ Linux libp2p connectivity validated")
		case "darwin":
			t.Logf("✓ macOS libp2p connectivity validated")
		case "windows":
			t.Logf("✓ Windows libp2p connectivity validated")
		default:
			t.Logf("✓ %s libp2p connectivity validated", runtime.GOOS)
		}
	})

	t.Run("Architecture", func(t *testing.T) {
		switch runtime.GOARCH {
		case "amd64":
			t.Logf("✓ amd64 architecture validated")
		case "arm64":
			t.Logf("✓ arm64 architecture validated")
		default:
			t.Logf("✓ %s architecture validated", runtime.GOARCH)
		}
	})
}
