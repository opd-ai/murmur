package onramp_tor

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOnion3Addr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid onion3 address with port",
			input: "/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
			want:  "vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd.onion:9001",
		},
		{
			name:    "no onion3 protocol",
			input:   "/ip4/127.0.0.1/tcp/9001",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.input)
			if tt.wantErr && err != nil {
				return
			}
			require.NoError(t, err)

			got, err := parseOnion3Addr(maddr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOnionAddrToMultiaddr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid onion address",
			input: "vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd.onion:9001",
			want:  "/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
		},
		{
			name:  "onion address without .onion suffix",
			input: "vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
			want:  "/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
		},
		{
			name:    "invalid address length",
			input:   "short.onion:9001",
			wantErr: true,
		},
		{
			name:    "missing port",
			input:   "vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd.onion",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := onionAddrToMultiaddr(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestHasOnion3Protocol(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "onion3 address",
			input: "/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
			want:  true,
		},
		{
			name:  "tcp address",
			input: "/ip4/127.0.0.1/tcp/9001",
			want:  false,
		},
		{
			name:  "quic address",
			input: "/ip4/127.0.0.1/udp/9001/quic-v1",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.input)
			require.NoError(t, err)
			got := hasOnion3Protocol(maddr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "onion3 with port",
			input: "/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
			want:  "9001",
		},
		{
			name:  "tcp with port",
			input: "/ip4/0.0.0.0/tcp/9001",
			want:  "9001",
		},
		{
			name:  "auto-assign port",
			input: "/ip4/0.0.0.0/tcp/0",
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.input)
			require.NoError(t, err)

			got, err := extractPort(maddr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCanDial(t *testing.T) {
	// Create a minimal transport instance for testing CanDial
	// We don't need a real onion instance for this test
	tr := &Transport{}

	tests := []struct {
		name string
		addr string
		want bool
	}{
		{
			name: "onion3 address",
			addr: "/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001",
			want: true,
		},
		{
			name: "tcp address",
			addr: "/ip4/127.0.0.1/tcp/9001",
			want: false,
		},
		{
			name: "quic address",
			addr: "/ip4/127.0.0.1/udp/9001/quic-v1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.addr)
			require.NoError(t, err)
			got := tr.CanDial(maddr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProtocols(t *testing.T) {
	tr := &Transport{}
	protocols := tr.Protocols()
	assert.Equal(t, []int{ma.P_ONION3}, protocols)
}

func TestProxy(t *testing.T) {
	tr := &Transport{}
	assert.True(t, tr.Proxy())
}

func TestNewTransportValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("nil upgrader", func(t *testing.T) {
		_, err := NewTransport(ctx, "test", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "upgrader cannot be nil")
	})

	t.Run("nil resource manager", func(t *testing.T) {
		// Create a mock upgrader (just for this test)
		mockUpgrader := &mockUpgrader{}
		_, err := NewTransport(ctx, "test", mockUpgrader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource manager cannot be nil")
	})
}

// mockUpgrader is a minimal mock for testing validation
type mockUpgrader struct{}

func (m *mockUpgrader) UpgradeListener(transport.Transport, manet.Listener) transport.Listener {
	return nil
}

func (m *mockUpgrader) GateMaListener(manet.Listener) transport.GatedMaListener {
	return nil
}

func (m *mockUpgrader) UpgradeGatedMaListener(transport.Transport, transport.GatedMaListener) transport.Listener {
	return nil
}

func (m *mockUpgrader) Upgrade(ctx context.Context, t transport.Transport, maconn manet.Conn, dir network.Direction, p peer.ID, scope network.ConnManagementScope) (transport.CapableConn, error) {
	return nil, nil
}
