package health

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/stretchr/testify/require"
)

func TestHealthServer(t *testing.T) {
	// Create a test libp2p host
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create health server (pubsub can be nil for this test)
	server := NewServer(host, nil)
	require.NotNil(t, server)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server on a random available port
	port := 18080
	err = server.Start(ctx, port)
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make HTTP request to /health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Decode JSON response
	var healthResp HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	require.NoError(t, err)

	// Verify response fields
	require.NotEmpty(t, healthResp.PeerID)
	require.Equal(t, host.ID().String(), healthResp.PeerID)
	require.GreaterOrEqual(t, healthResp.Connections, 0)
	require.NotEmpty(t, healthResp.Status)
	require.GreaterOrEqual(t, healthResp.UptimeSeconds, int64(0))
	require.Greater(t, healthResp.Timestamp, int64(0))

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	err = server.Stop(stopCtx)
	require.NoError(t, err)
}

func TestHealthServerMultipleStarts(t *testing.T) {
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	server := NewServer(host, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	err = server.Start(ctx, 18081)
	require.NoError(t, err)

	// Try to start again - should fail
	err = server.Start(ctx, 18081)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already started")

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	err = server.Stop(stopCtx)
	require.NoError(t, err)
}
