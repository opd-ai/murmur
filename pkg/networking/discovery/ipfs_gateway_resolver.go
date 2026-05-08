// Package discovery provides IPFS gateway-based bootstrap resolver.
// Per PLAN.md "Layer 4 — IPFS CID fallback (dweb.link)".
package discovery

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// IPFSGatewayResolver fetches bootstrap peers from IPFS via HTTP gateway.
// Per PLAN.md: "Reads cid.txt from the Pages URL to get the latest IPFS CID,
// then fetches https://dweb.link/ipfs/<CID>/peers.json. 20s timeout."
type IPFSGatewayResolver struct {
	cidURL      string // URL to cid.txt on GitHub Pages
	gatewayURL  string // IPFS gateway base URL (e.g., https://dweb.link)
	verifyKey   []byte
	client      *http.Client
	pagesClient *http.Client
}

// NewIPFSGatewayResolver creates an IPFS gateway-based bootstrap resolver.
// cidURL should point to the cid.txt file on GitHub Pages.
// gatewayURL is the IPFS HTTP gateway (e.g., "https://dweb.link")
func NewIPFSGatewayResolver(cidURL, gatewayURL string, verifyKey []byte) *IPFSGatewayResolver {
	return &IPFSGatewayResolver{
		cidURL:     cidURL,
		gatewayURL: gatewayURL,
		verifyKey:  verifyKey,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		pagesClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Resolve fetches the CID from Pages, then fetches and verifies the peer list from IPFS.
func (i *IPFSGatewayResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	cid, err := i.fetchCID(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cid: %w", err)
	}
	if cid == "" {
		return nil, fmt.Errorf("empty cid")
	}

	peerListURL := fmt.Sprintf("%s/ipfs/%s/peers.json", i.gatewayURL, cid)
	signedList, err := i.fetchSignedPeerList(ctx, peerListURL)
	if err != nil {
		return nil, err
	}

	if err := i.verifySignature(signedList); err != nil {
		return nil, err
	}

	return signedList.ToPeerAddrInfos()
}

// fetchSignedPeerList downloads and parses the signed peer list from IPFS gateway.
func (i *IPFSGatewayResolver) fetchSignedPeerList(ctx context.Context, url string) (*SignedPeerList, error) {
	body, err := i.fetchURL(ctx, url, i.client)
	if err != nil {
		return nil, fmt.Errorf("fetch from ipfs: %w", err)
	}

	var signedList SignedPeerList
	if err := json.Unmarshal(body, &signedList); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return &signedList, nil
}

// fetchURL performs an HTTP GET request and returns the body.
// Responses exceeding maxBootstrapResponseBytes are rejected to prevent DoS.
func (i *IPFSGatewayResolver) fetchURL(ctx context.Context, url string, client *http.Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch failed: status %d", resp.StatusCode)
	}

	return readLimitedBody(resp.Body, "ipfs")
}

// verifySignature checks the peer list signature if a verification key is configured.
func (i *IPFSGatewayResolver) verifySignature(signedList *SignedPeerList) error {
	if len(i.verifyKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid bootstrap verification key size: got %d, want %d", len(i.verifyKey), ed25519.PublicKeySize)
	}
	if err := signedList.Verify(i.verifyKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	return nil
}

// fetchCID fetches the IPFS CID from the cid.txt file on GitHub Pages.
// The response is capped at maxBootstrapResponseBytes to prevent DoS.
func (i *IPFSGatewayResolver) fetchCID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", i.cidURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := i.pagesClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch cid.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cid fetch failed: status %d", resp.StatusCode)
	}

	body, err := readLimitedBody(resp.Body, "cid")
	if err != nil {
		return "", err
	}

	// Trim whitespace and return
	cid := strings.TrimSpace(string(body))
	if cid == "" {
		return "", fmt.Errorf("empty cid in response")
	}

	return cid, nil
}

// Name returns the resolver name for logging.
func (i *IPFSGatewayResolver) Name() string {
	return "ipfs-gateway"
}
