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
// Per NETWORK_ARCHITECTURE.md §5, nodes connect to 2+ bootstrap peers within 10 seconds.
//
// DEVELOPMENT NOTE: These are placeholder addresses. Production deployment requires:
// - 8-12 long-running libp2p nodes on public infrastructure (DigitalOcean, AWS, Hetzner)
// - Each node runs `./murmur --bootstrap-mode` (to be implemented)
// - Multiaddrs recorded here after deployment
// - Geographic distribution across 3+ jurisdictions for resilience
//
// For local development/testing, users can override with `--bootstrap` flag or set
// BootstrapPeers in Config. For multi-machine local testing, run one instance as
// bootstrap and connect others to it:
//
//	Terminal 1: ./murmur --bootstrap-mode  (prints peer ID and multiaddr)
//	Terminal 2: ./murmur --bootstrap=/ip4/127.0.0.1/tcp/XXXXX/p2p/12D3K...
var DefaultBootstrapPeers = []string{
	// Production bootstrap nodes (to be deployed):
	// "/dns4/bootstrap-1.murmur.network/tcp/4001/p2p/12D3K...",
	// "/dns4/bootstrap-2.murmur.network/tcp/4001/p2p/12D3K...",
	// ... (8-12 total entries)

	// For now, empty list results in isolated mode warning.
	// Users must manually configure bootstrap peers or run local bootstrap node.
}

// BootstrapSources holds URLs for GitHub-based bootstrap discovery.
// Per PLAN.md "Bootstrap Strategy: Zero-Infrastructure Peer Discovery".
type BootstrapSources struct {
	GistURL        string // Raw Gist URL (set at compile time via ldflags)
	PagesURL       string // GitHub Pages URL (e.g., https://opd-ai.github.io/murmur/peers.json)
	IPFSCidURL     string // URL to cid.txt on Pages (for IPFS gateway fallback)
	IPFSGatewayURL string // IPFS HTTP gateway (e.g., https://dweb.link)
	DHTNamespace   string // DHT rendezvous namespace (e.g., /murmur/bootstrap/v1)
}

// DefaultBootstrapSources provides the default GitHub-based bootstrap sources.
// Per PLAN.md: These are layered with fallback behavior.
var DefaultBootstrapSources = BootstrapSources{
	GistURL:        "", // Set via ldflags: -X 'github.com/opd-ai/murmur/pkg/config.GistRawURL=...'
	PagesURL:       "https://opd-ai.github.io/murmur/peers.json",
	IPFSCidURL:     "https://opd-ai.github.io/murmur/cid.txt",
	IPFSGatewayURL: "https://dweb.link",
	DHTNamespace:   "/murmur/bootstrap/v1",
}

// GistRawURL can be set at compile time via ldflags.
var GistRawURL string

func init() {
	// Apply ldflags-provided Gist URL if set
	if GistRawURL != "" {
		DefaultBootstrapSources.GistURL = GistRawURL
	}
}
