// Package config provides configuration loading, defaults, and validation for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md §2, configuration is loaded from the user data
// directory and merged with command-line overrides.
package config

// DefaultListenAddrs are the default multiaddrs for the libp2p host.
var DefaultListenAddrs = []string{
	"/ip4/0.0.0.0/tcp/0",
	"/ip4/0.0.0.0/udp/0/quic-v1",
}

// DefaultBootstrapPeers are the hardcoded bootstrap nodes for initial network join.
// Per DESIGN_DOCUMENT.md, these are community-operated, multi-jurisdiction nodes.
// TODO: Replace with actual bootstrap node addresses once infrastructure is established.
var DefaultBootstrapPeers = []string{
	// Placeholder bootstrap nodes.
}
