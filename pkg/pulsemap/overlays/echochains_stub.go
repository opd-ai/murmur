// Package overlays - Echo Chain overlay stub for non-Ebiten builds.
//
//go:build test
// +build test

package overlays

import (
	"time"
)

// ChainLayer indicates which layer the chain belongs to.
type ChainLayer uint8

const (
	ChainLayerSurface ChainLayer = iota + 1
	ChainLayerAnonymous
)

// ChainNodeInfo contains node information for visualization.
type ChainNodeInfo struct {
	NodeID      [32]byte
	X, Y        float64
	AmplifiedAt time.Time
	Position    int
}

// EchoChainInfo contains chain information for visualization.
type EchoChainInfo struct {
	ChainID    [32]byte
	OriginalID [32]byte
	Layer      ChainLayer
	Nodes      []*ChainNodeInfo
	FormedAt   time.Time
	ExpiresAt  time.Time
	HasShimmer bool
}

// EchoChainOverlay is a stub for non-Ebiten builds.
type EchoChainOverlay struct {
	visible bool
	chains  map[[32]byte]*EchoChainInfo
}

// NewEchoChainOverlay creates a new stub overlay.
func NewEchoChainOverlay() *EchoChainOverlay {
	return &EchoChainOverlay{
		visible: true,
		chains:  make(map[[32]byte]*EchoChainInfo),
	}
}

// SetVisible controls visibility.
func (o *EchoChainOverlay) SetVisible(visible bool) { o.visible = visible }

// IsVisible returns visibility status.
func (o *EchoChainOverlay) IsVisible() bool { return o.visible }

// SetChain adds or updates an echo chain.
func (o *EchoChainOverlay) SetChain(chain *EchoChainInfo) {
	if chain != nil {
		o.chains[chain.ChainID] = chain
	}
}

// RemoveChain removes a chain by ID.
func (o *EchoChainOverlay) RemoveChain(id [32]byte) { delete(o.chains, id) }

// GetChain returns a chain by ID.
func (o *EchoChainOverlay) GetChain(id [32]byte) *EchoChainInfo { return o.chains[id] }

// GetAllChains returns all chains.
func (o *EchoChainOverlay) GetAllChains() []*EchoChainInfo {
	chains := make([]*EchoChainInfo, 0, len(o.chains))
	for _, c := range o.chains {
		chains = append(chains, c)
	}
	return chains
}

// GetActiveChains returns non-expired chains.
func (o *EchoChainOverlay) GetActiveChains() []*EchoChainInfo {
	now := time.Now()
	var active []*EchoChainInfo
	for _, c := range o.chains {
		if now.Before(c.ExpiresAt) {
			active = append(active, c)
		}
	}
	return active
}

// UpdateNodePosition updates the position of a node in all chains.
func (o *EchoChainOverlay) UpdateNodePosition(nodeID [32]byte, x, y float64) {
	for _, chain := range o.chains {
		for _, node := range chain.Nodes {
			if node.NodeID == nodeID {
				node.X = x
				node.Y = y
			}
		}
	}
}

// Update is a no-op stub.
func (o *EchoChainOverlay) Update(dt float64) {}

// ChainCount returns the total number of chains.
func (o *EchoChainOverlay) ChainCount() int { return len(o.chains) }

// ActiveChainCount returns the number of non-expired chains.
func (o *EchoChainOverlay) ActiveChainCount() int {
	now := time.Now()
	count := 0
	for _, c := range o.chains {
		if now.Before(c.ExpiresAt) {
			count++
		}
	}
	return count
}

// ShimmeringChainCount returns the number of chains with shimmer effect.
func (o *EchoChainOverlay) ShimmeringChainCount() int {
	now := time.Now()
	count := 0
	for _, c := range o.chains {
		if c.HasShimmer && now.Before(c.ExpiresAt) {
			count++
		}
	}
	return count
}

// ClearExpired removes expired chains.
func (o *EchoChainOverlay) ClearExpired() int {
	now := time.Now()
	removed := 0
	for id, chain := range o.chains {
		if now.After(chain.ExpiresAt) {
			delete(o.chains, id)
			removed++
		}
	}
	return removed
}

// GetChainsByLayer returns chains of a specific layer.
func (o *EchoChainOverlay) GetChainsByLayer(layer ChainLayer) []*EchoChainInfo {
	now := time.Now()
	var chains []*EchoChainInfo
	for _, c := range o.chains {
		if c.Layer == layer && now.Before(c.ExpiresAt) {
			chains = append(chains, c)
		}
	}
	return chains
}

// ChainLayerString returns a human-readable name.
func ChainLayerString(layer ChainLayer) string {
	switch layer {
	case ChainLayerSurface:
		return "Surface"
	case ChainLayerAnonymous:
		return "Anonymous"
	default:
		return "Unknown"
	}
}

// AddNodeToChain adds a new node to an existing chain.
func (o *EchoChainOverlay) AddNodeToChain(chainID [32]byte, node *ChainNodeInfo) {
	if node == nil {
		return
	}
	if chain, ok := o.chains[chainID]; ok {
		node.Position = len(chain.Nodes)
		chain.Nodes = append(chain.Nodes, node)
		if len(chain.Nodes) >= 5 {
			chain.HasShimmer = true
		}
	}
}

// GetLongestChain returns the chain with the most nodes.
func (o *EchoChainOverlay) GetLongestChain() *EchoChainInfo {
	now := time.Now()
	var longest *EchoChainInfo
	maxLen := 0
	for _, c := range o.chains {
		if len(c.Nodes) > maxLen && now.Before(c.ExpiresAt) {
			longest = c
			maxLen = len(c.Nodes)
		}
	}
	return longest
}
