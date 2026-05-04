// Package discovery provides HTTP-based bootstrap resolver utilities.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/libp2p/go-libp2p/core/peer"
)

// fetchAndVerifyPeerList fetches a signed peer list from an HTTP URL and verifies it.
// Returns the verified peer.AddrInfo slice or an error.
func fetchAndVerifyPeerList(ctx context.Context, client *http.Client, url string, verifyKey []byte, source string) ([]peer.AddrInfo, error) {
	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Fetch
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s fetch failed: status %d", source, resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Parse JSON
	var signedList SignedPeerList
	if err := json.Unmarshal(body, &signedList); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	// Verify signature if verify key is set
	if len(verifyKey) > 0 {
		if err := signedList.Verify(verifyKey); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Convert to peer.AddrInfo
	peers, err := signedList.ToPeerAddrInfos()
	if err != nil {
		return nil, fmt.Errorf("convert to peer infos: %w", err)
	}

	return peers, nil
}
