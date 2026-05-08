package onrampi2p

import (
	"context"
	"encoding/base64"
	"net"
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validI2PDestination is a valid 516-byte I2P destination (387 bytes minimum after base64 decode).
// This is a real I2P destination format but with dummy data.
var validI2PDestination = base64.StdEncoding.EncodeToString(make([]byte, 387))

func TestParseGarlicAddr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid garlic64 address",
			input: "/garlic64/" + validI2PDestination,
			want:  validI2PDestination,
		},
		{
			name:    "no garlic64 protocol",
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

			got, err := parseGarlicAddr(maddr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGarlicAddrToMultiaddr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid I2P destination",
			input: validI2PDestination,
			want:  "/garlic64/" + validI2PDestination,
		},
		{
			name:    "invalid base64",
			input:   "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "too short destination",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 100)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := garlicAddrToMultiaddr(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestHasGarlicProtocol(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "garlic64 address",
			input: "/garlic64/" + validI2PDestination,
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
			got := hasGarlicProtocol(maddr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tcp with port",
			input: "/ip4/0.0.0.0/tcp/9001",
			want:  "9001",
		},
		{
			name:  "udp with port",
			input: "/ip4/0.0.0.0/udp/7656",
			want:  "7656",
		},
		{
			name:  "garlic64 without port",
			input: "/garlic64/" + validI2PDestination,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.input)
			require.NoError(t, err)
			got := parsePort(maddr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendPortIfPresent(t *testing.T) {
	dest := validI2PDestination

	tests := []struct {
		name     string
		dest     string
		maddr    string
		expected string
	}{
		{
			name:     "no port in multiaddr",
			dest:     dest,
			maddr:    "/garlic64/" + dest,
			expected: dest,
		},
		{
			name:     "tcp port in multiaddr",
			dest:     dest,
			maddr:    "/garlic64/" + dest + "/tcp/9001",
			expected: dest + ":9001",
		},
		{
			name:     "destination already has port",
			dest:     dest + ":9001",
			maddr:    "/garlic64/" + dest + "/tcp/8080",
			expected: dest + ":9001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.maddr)
			if err != nil {
				// If the multiaddr is invalid, construct a simple one
				maddr, _ = ma.NewMultiaddr("/garlic64/" + dest)
			}
			got := appendPortIfPresent(tt.dest, maddr)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCanDial(t *testing.T) {
	tr := &Transport{}

	tests := []struct {
		name string
		addr string
		want bool
	}{
		{
			name: "garlic64 address",
			addr: "/garlic64/" + validI2PDestination,
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
	assert.Equal(t, []int{ma.P_GARLIC64}, protocols)
}

func TestProxy(t *testing.T) {
	tr := &Transport{}
	assert.True(t, tr.Proxy())
}

func TestNewTransportValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("nil upgrader", func(t *testing.T) {
		_, err := NewTransport(ctx, "test", DefaultSAMAddr, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "upgrader cannot be nil")
	})

	t.Run("nil resource manager", func(t *testing.T) {
		mockUpgrader := &mockUpgrader{}
		_, err := NewTransport(ctx, "test", DefaultSAMAddr, nil, mockUpgrader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource manager cannot be nil")
	})

	// Note: We cannot test actual I2P connection without a running I2P router with SAM enabled.
	// The NewGarlic constructor will attempt to connect to the SAM bridge, which will fail
	// in test environments. Integration tests requiring I2P should use docker-compose to
	// spin up i2pd/java-i2p with SAMv3 enabled.
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

// mockResourceManager is a minimal mock for testing validation
type mockResourceManager struct{}

func (m *mockResourceManager) OpenConnection(network.Direction, bool, ma.Multiaddr) (network.ConnManagementScope, error) {
	return &mockConnScope{}, nil
}

func (m *mockResourceManager) OpenStream(peer.ID, network.Direction) (network.StreamManagementScope, error) {
	return nil, nil
}
func (m *mockResourceManager) Close() error { return nil }
func (m *mockResourceManager) ViewSystem(func(network.ResourceScope) error) error {
	return nil
}

func (m *mockResourceManager) ViewTransient(func(network.ResourceScope) error) error {
	return nil
}

func (m *mockResourceManager) ViewService(string, func(network.ServiceScope) error) error {
	return nil
}

func (m *mockResourceManager) ViewProtocol(protocol.ID, func(network.ProtocolScope) error) error {
	return nil
}

func (m *mockResourceManager) ViewPeer(peer.ID, func(network.PeerScope) error) error {
	return nil
}

func (m *mockResourceManager) VerifySourceAddress(addr net.Addr) bool {
	return true
}

type mockConnScope struct{}

func (m *mockConnScope) Done()                          {}
func (m *mockConnScope) ReserveMemory(int, uint8) error { return nil }
func (m *mockConnScope) ReleaseMemory(int)              {}
func (m *mockConnScope) Stat() network.ScopeStat        { return network.ScopeStat{} }
func (m *mockConnScope) BeginSpan() (network.ResourceScopeSpan, error) {
	return nil, nil
}
func (m *mockConnScope) SetPeer(peer.ID) error        { return nil }
func (m *mockConnScope) PeerScope() network.PeerScope { return nil }
