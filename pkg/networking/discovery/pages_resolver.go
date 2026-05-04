// Package discovery provides GitHub Pages-based bootstrap resolver.
// Per PLAN.md "Layer 2 — GitHub Pages (durable, versioned, CID-indexed)".
package discovery

import (
	"context"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PagesResolver fetches bootstrap peers from GitHub Pages.
// Per PLAN.md: "same logic against Pages URL. 3s timeout."
type PagesResolver struct {
	url       string
	verifyKey []byte
	client    *http.Client
}

// NewPagesResolver creates a GitHub Pages-based bootstrap resolver.
// url should be the Pages URL (e.g., https://opd-ai.github.io/murmur/peers.json)
func NewPagesResolver(url string, verifyKey []byte) *PagesResolver {
	return &PagesResolver{
		url:       url,
		verifyKey: verifyKey,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Resolve fetches and verifies the peer list from GitHub Pages.
func (p *PagesResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	return fetchAndVerifyPeerList(ctx, p.client, p.url, p.verifyKey, "pages")
}

// Name returns the resolver name for logging.
func (p *PagesResolver) Name() string {
	return "pages"
}
