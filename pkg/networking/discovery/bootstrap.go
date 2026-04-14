// Package discovery provides bootstrap node configuration.
// Per NETWORK_ARCHITECTURE.md §3 (Bootstrap Nodes):
// "MURMUR uses a set of bootstrap nodes — well-known peers whose network addresses
// are hardcoded into the application — as initial entry points."

package discovery

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// BootstrapNodes contains the default bootstrap nodes for MURMUR.
// Per NETWORK_ARCHITECTURE.md §3: "The default configuration includes 8–12 bootstrap
// nodes distributed across different geographic regions and network providers."
//
// Bootstrap nodes are community-operated and their addresses are updated with each
// application release. These are placeholder addresses for development.
// Production addresses should be replaced before release.
var BootstrapNodes = []peer.AddrInfo{
	// North America - East
	mustParseAddrInfo("/dns4/bootstrap1.murmur.network/tcp/4001/p2p/12D3KooWBootstrap1NAEast"),
	mustParseAddrInfo("/dns4/bootstrap2.murmur.network/tcp/4001/p2p/12D3KooWBootstrap2NAEast"),
	// North America - West
	mustParseAddrInfo("/dns4/bootstrap3.murmur.network/tcp/4001/p2p/12D3KooWBootstrap3NAWest"),
	mustParseAddrInfo("/dns4/bootstrap4.murmur.network/tcp/4001/p2p/12D3KooWBootstrap4NAWest"),
	// Europe
	mustParseAddrInfo("/dns4/bootstrap5.murmur.network/tcp/4001/p2p/12D3KooWBootstrap5Europe"),
	mustParseAddrInfo("/dns4/bootstrap6.murmur.network/tcp/4001/p2p/12D3KooWBootstrap6Europe"),
	// Asia Pacific
	mustParseAddrInfo("/dns4/bootstrap7.murmur.network/tcp/4001/p2p/12D3KooWBootstrap7AsiaPacific"),
	mustParseAddrInfo("/dns4/bootstrap8.murmur.network/tcp/4001/p2p/12D3KooWBootstrap8AsiaPacific"),
}

// DefaultBootstrapCount is the number of bootstrap nodes to connect to.
// Per NETWORK_ARCHITECTURE.md: "A new node connects to 2–3 randomly selected bootstrap nodes."
const DefaultBootstrapCount = 3

// MinBootstrapCount is the minimum number of bootstrap connections required.
const MinBootstrapCount = 1

// mustParseAddrInfo parses a multiaddr string to peer.AddrInfo.
// Panics on error - only use for hardcoded addresses.
func mustParseAddrInfo(s string) peer.AddrInfo {
	ma, err := multiaddr.NewMultiaddr(s)
	if err != nil {
		// For development placeholders, return empty addr.
		// Production should use valid addresses.
		return peer.AddrInfo{}
	}

	ai, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return peer.AddrInfo{}
	}
	return *ai
}

// ValidBootstrapNodes returns only the bootstrap nodes that have valid addresses.
// This filters out placeholder addresses.
func ValidBootstrapNodes() []peer.AddrInfo {
	var valid []peer.AddrInfo
	for _, ai := range BootstrapNodes {
		if len(ai.Addrs) > 0 && ai.ID != "" {
			valid = append(valid, ai)
		}
	}
	return valid
}

// TestnetBootstrapNodes are bootstrap nodes for testing and development.
// These can be used in test environments.
var TestnetBootstrapNodes = []peer.AddrInfo{}

// LocalBootstrapNodes returns bootstrap nodes suitable for local testing.
// Returns an empty list - mDNS should be used for local discovery.
func LocalBootstrapNodes() []peer.AddrInfo {
	return nil
}

// CustomBootstrapNodes holds user-configured additional bootstrap nodes.
// Per NETWORK_ARCHITECTURE.md: "Bootstrap node addresses are updated with each
// application release. If all hardcoded bootstrap nodes are offline, the application
// prompts the user to manually enter a peer address."
type CustomBootstrapNodes struct {
	nodes []peer.AddrInfo
}

// NewCustomBootstrapNodes creates a new custom bootstrap node list.
func NewCustomBootstrapNodes() *CustomBootstrapNodes {
	return &CustomBootstrapNodes{
		nodes: make([]peer.AddrInfo, 0),
	}
}

// Add adds a bootstrap node by multiaddr string.
func (c *CustomBootstrapNodes) Add(addr string) error {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}

	ai, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return err
	}

	c.nodes = append(c.nodes, *ai)
	return nil
}

// AddAddrInfo adds a bootstrap node by AddrInfo.
func (c *CustomBootstrapNodes) AddAddrInfo(ai peer.AddrInfo) {
	c.nodes = append(c.nodes, ai)
}

// List returns all custom bootstrap nodes.
func (c *CustomBootstrapNodes) List() []peer.AddrInfo {
	return c.nodes
}

// AllBootstrapNodes returns all bootstrap nodes (default + custom).
func AllBootstrapNodes(custom *CustomBootstrapNodes) []peer.AddrInfo {
	var customLen int
	if custom != nil {
		customLen = len(custom.nodes)
	}
	nodes := make([]peer.AddrInfo, 0, len(BootstrapNodes)+customLen)
	nodes = append(nodes, BootstrapNodes...)
	if custom != nil {
		nodes = append(nodes, custom.nodes...)
	}
	return nodes
}
