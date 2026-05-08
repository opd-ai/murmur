// Package discovery provides HTTP-based bootstrap resolver utilities.
package discovery

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/libp2p/go-libp2p/core/peer"
)

// maxBootstrapResponseBytes is the maximum number of bytes read from a remote
// bootstrap response.  Responses exceeding this limit are rejected to prevent
// memory-exhaustion DoS from malicious or oversized bootstrap endpoints.
const maxBootstrapResponseBytes = 1 << 20 // 1 MiB

// readLimitedBody reads at most maxBootstrapResponseBytes from r.
// It returns an error if the response body exceeds that limit.
func readLimitedBody(r io.Reader, label string) ([]byte, error) {
	limited := io.LimitReader(r, maxBootstrapResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > maxBootstrapResponseBytes {
		return nil, fmt.Errorf("%s response exceeds %d-byte limit", label, maxBootstrapResponseBytes)
	}
	return data, nil
}

// fetchAndVerifyPeerList fetches a signed peer list from an HTTP URL and verifies it.
// Returns the verified peer.AddrInfo slice or an error.
func fetchAndVerifyPeerList(ctx context.Context, client *http.Client, url string, verifyKey []byte, source string) ([]peer.AddrInfo, error) {
	body, err := fetchPeerListBody(ctx, client, url, source)
	if err != nil {
		return nil, err
	}

	signedList, err := parsePeerList(body)
	if err != nil {
		return nil, err
	}

	if err := verifyPeerListSignature(signedList, verifyKey); err != nil {
		return nil, err
	}

	return signedList.ToPeerAddrInfos()
}

// fetchPeerListBody fetches the HTTP response body.
func fetchPeerListBody(ctx context.Context, client *http.Client, url, source string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s fetch failed: status %d", source, resp.StatusCode)
	}

	return readLimitedBody(resp.Body, source)
}

// parsePeerList unmarshals JSON into a SignedPeerList.
func parsePeerList(body []byte) (*SignedPeerList, error) {
	var signedList SignedPeerList
	if err := json.Unmarshal(body, &signedList); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return &signedList, nil
}

// verifyPeerListSignature checks signature if verify key is provided.
func verifyPeerListSignature(signedList *SignedPeerList, verifyKey []byte) error {
	if len(verifyKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid bootstrap verification key size: got %d, want %d", len(verifyKey), ed25519.PublicKeySize)
	}
	if err := signedList.Verify(verifyKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	return nil
}
