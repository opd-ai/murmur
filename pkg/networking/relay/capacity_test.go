package relay

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRelayCapacityConfig(t *testing.T) {
	cfg := DefaultRelayCapacityConfig()

	// Per NETWORK_ARCHITECTURE.md §Relay Nodes
	assert.Equal(t, 128, cfg.MaxReservations)
	assert.Equal(t, int64(128*1024), cfg.MaxDataPerConnection) // 128 KB
	assert.Equal(t, 2*time.Minute, cfg.ConnectionDuration)
	assert.Equal(t, 1*time.Hour, cfg.ReservationTTL)
	assert.Equal(t, 16, cfg.MaxCircuitsPerPeer)
	assert.Equal(t, 8, cfg.MaxReservationsPerIP)
	assert.Equal(t, 32, cfg.MaxReservationsPerASN)
	assert.Equal(t, 2048, cfg.BufferSize)
}

func TestLowCapacityConfig(t *testing.T) {
	cfg := LowCapacityConfig()

	// Should be lower than defaults
	assert.Equal(t, 32, cfg.MaxReservations)
	assert.Equal(t, int64(64*1024), cfg.MaxDataPerConnection)
	assert.Equal(t, 8, cfg.MaxCircuitsPerPeer)
	assert.Equal(t, 4, cfg.MaxReservationsPerIP)
	assert.Equal(t, 16, cfg.MaxReservationsPerASN)
	assert.Equal(t, 1024, cfg.BufferSize)
}

func TestHighCapacityConfig(t *testing.T) {
	cfg := HighCapacityConfig()

	// Should be higher than defaults
	assert.Equal(t, 512, cfg.MaxReservations)
	assert.Equal(t, int64(256*1024), cfg.MaxDataPerConnection)
	assert.Equal(t, 32, cfg.MaxCircuitsPerPeer)
	assert.Equal(t, 16, cfg.MaxReservationsPerIP)
	assert.Equal(t, 64, cfg.MaxReservationsPerASN)
	assert.Equal(t, 4096, cfg.BufferSize)
}

func TestRelayCapacityConfig_ToLibp2pResources(t *testing.T) {
	cfg := DefaultRelayCapacityConfig()
	res := cfg.ToLibp2pResources()

	assert.Equal(t, cfg.MaxReservations, res.MaxReservations)
	assert.Equal(t, cfg.MaxCircuitsPerPeer, res.MaxCircuits)
	assert.Equal(t, cfg.BufferSize, res.BufferSize)
	assert.Equal(t, cfg.MaxReservationsPerIP, res.MaxReservationsPerIP)
	assert.Equal(t, cfg.MaxReservationsPerASN, res.MaxReservationsPerASN)
	assert.Equal(t, cfg.ReservationTTL, res.ReservationTTL)

	require.NotNil(t, res.Limit)
	assert.Equal(t, cfg.ConnectionDuration, res.Limit.Duration)
	assert.Equal(t, cfg.MaxDataPerConnection, res.Limit.Data)
}

func TestRelayCapacityConfig_ToLibp2pOptions(t *testing.T) {
	cfg := DefaultRelayCapacityConfig()
	opts := cfg.ToLibp2pOptions()

	assert.NotEmpty(t, opts)
}

func TestNewRelayService(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	cfg := DefaultRelayCapacityConfig()
	rs, err := NewRelayService(h, cfg)
	require.NoError(t, err)
	require.NotNil(t, rs)

	assert.Equal(t, cfg, rs.Config())

	err = rs.Close()
	assert.NoError(t, err)
}

func TestNewRelayServiceWithDefaults(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	rs, err := NewRelayServiceWithDefaults(h)
	require.NoError(t, err)
	require.NotNil(t, rs)

	// Should have default config
	assert.Equal(t, DefaultRelayCapacityConfig(), rs.Config())

	err = rs.Close()
	assert.NoError(t, err)
}

func TestCapacityConstants(t *testing.T) {
	// Verify constants match spec from NETWORK_ARCHITECTURE.md
	assert.Equal(t, 128, DefaultMaxReservations, "max 128 concurrent reservations")
	assert.Equal(t, 128*1024, DefaultMaxDataPerConn, "128 KB/s per connection")
	assert.Equal(t, 2*time.Minute, DefaultConnectionDuration)
	assert.Equal(t, 1*time.Hour, DefaultReservationTTL)
}
