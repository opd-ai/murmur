// Package discovery provides GitHub Gist-based bootstrap resolver.
// Per PLAN.md "Layer 1 — The Living Gist (fast, mutable, CI-maintained)".
package discovery

import (
	"context"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// GistResolver fetches bootstrap peers from a GitHub Gist.
// Per PLAN.md: "HTTP GET → JSON decode → Ed25519 signature verify → return []peer.AddrInfo. 2s timeout."
type GistResolver struct {
	url       string
	verifyKey []byte
	client    *http.Client
}

// NewGistResolver creates a Gist-based bootstrap resolver.
// url should be the raw Gist URL (e.g., https://gist.githubusercontent.com/...)
func NewGistResolver(url string, verifyKey []byte) *GistResolver {
	return &GistResolver{
		url:       url,
		verifyKey: verifyKey,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// Resolve fetches and verifies the peer list from the Gist.
func (g *GistResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	return fetchAndVerifyPeerList(ctx, g.client, g.url, g.verifyKey, "gist")
}

// Name returns the resolver name for logging.
func (g *GistResolver) Name() string {
	return "gist"
}
