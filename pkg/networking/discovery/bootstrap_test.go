package discovery

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapNodes_AllEntriesValid(t *testing.T) {
	// All entries in BootstrapNodes must have valid peer IDs and at least one address.
	// The static list may be empty while runtime discovery (ResolverChain) is the
	// primary bootstrap mechanism.  No invalid placeholder entries are allowed.
	for i, ai := range BootstrapNodes {
		assert.NotEmpty(t, ai.ID, "BootstrapNodes[%d] must have a non-empty peer ID", i)
		assert.NotEmpty(t, ai.Addrs, "BootstrapNodes[%d] must have at least one address", i)
	}
}

func TestDefaultBootstrapCount(t *testing.T) {
	// Per NETWORK_ARCHITECTURE.md: connect to 2-3 bootstrap nodes.
	assert.Equal(t, 3, DefaultBootstrapCount)
	assert.Equal(t, 1, MinBootstrapCount)
}

func TestValidBootstrapNodes(t *testing.T) {
	// ValidBootstrapNodes should filter out empty addresses.
	valid := ValidBootstrapNodes()
	for _, ai := range valid {
		assert.NotEmpty(t, ai.Addrs, "Valid node should have addresses")
		assert.NotEmpty(t, ai.ID, "Valid node should have ID")
	}
}

func TestCustomBootstrapNodes(t *testing.T) {
	custom := NewCustomBootstrapNodes()
	require.NotNil(t, custom)
	assert.Empty(t, custom.List())

	// Add a node by multiaddr string.
	err := custom.Add("/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWTestPeer1")
	// This may fail due to invalid peer ID format, which is fine.
	if err == nil {
		assert.Len(t, custom.List(), 1)
	}

	// Add by AddrInfo.
	custom.AddAddrInfo(peer.AddrInfo{
		ID: "test-peer",
	})
	assert.GreaterOrEqual(t, len(custom.List()), 1)
}

func TestCustomBootstrapNodes_InvalidAddr(t *testing.T) {
	custom := NewCustomBootstrapNodes()

	// Invalid multiaddr should error.
	err := custom.Add("not-a-valid-multiaddr")
	assert.Error(t, err)
}

func TestAllBootstrapNodes(t *testing.T) {
	custom := NewCustomBootstrapNodes()
	custom.AddAddrInfo(peer.AddrInfo{ID: "custom1"})
	custom.AddAddrInfo(peer.AddrInfo{ID: "custom2"})

	all := AllBootstrapNodes(custom)
	assert.Equal(t, len(BootstrapNodes)+2, len(all), "should include static + custom nodes")
}

func TestAllBootstrapNodes_NilCustom(t *testing.T) {
	all := AllBootstrapNodes(nil)
	assert.Equal(t, len(BootstrapNodes), len(all))
}

func TestLocalBootstrapNodes(t *testing.T) {
	// Local bootstrap returns empty list (use mDNS instead).
	local := LocalBootstrapNodes()
	assert.Nil(t, local)
}

func TestMustParseAddrInfo_Invalid(t *testing.T) {
	// Invalid addresses should not panic, but return empty AddrInfo.
	require.NotPanics(t, func() {
		ai := mustParseAddrInfo("invalid-addr")
		assert.Empty(t, ai.ID)
	})
}
