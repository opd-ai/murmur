// Package discovery provides signed peer list structures for bootstrap.
// Per PLAN.md: "signed `peers.json` is publicly readable" with Ed25519 verification.
package discovery

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// SignedPeerList represents a list of peer addresses with Ed25519 signature.
// Per PLAN.md: "JSON decode → Ed25519 signature verify → return []peer.AddrInfo".
type SignedPeerList struct {
	Version   int         `json:"version"`   // Format version (currently 1)
	Timestamp int64       `json:"timestamp"` // Unix timestamp of signing
	Peers     []PeerEntry `json:"peers"`     // List of peer multiaddrs
	Signature string      `json:"signature"` // Base64-encoded Ed25519 signature
	SignedBy  string      `json:"signed_by"` // Base64-encoded public key
}

// PeerEntry represents a single peer's address information.
type PeerEntry struct {
	ID    string   `json:"id"`    // Peer ID (e.g., "12D3KooW...")
	Addrs []string `json:"addrs"` // Multiaddr strings
	Seen  int64    `json:"seen"`  // Last seen Unix timestamp
}

// Sign signs the peer list with the given Ed25519 private key.
// Per PLAN.md: "merge, deduplicate, prune stale entries, sign".
func (spl *SignedPeerList) Sign(privateKey ed25519.PrivateKey) error {
	if len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key size: got %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}

	// Set signing metadata
	spl.Version = 1
	spl.Timestamp = time.Now().Unix()
	spl.SignedBy = encodeBase64(privateKey.Public().(ed25519.PublicKey))

	// Create canonical byte representation for signing
	payload, err := spl.canonicalBytes()
	if err != nil {
		return fmt.Errorf("canonical bytes: %w", err)
	}

	// Sign
	signature := ed25519.Sign(privateKey, payload)
	spl.Signature = encodeBase64(signature)

	return nil
}

// Verify verifies the signature using the provided Ed25519 public key.
// Per PLAN.md: "Signature is verified against the embedded BOOTSTRAP_VERIFY_KEY".
func (spl *SignedPeerList) Verify(publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: got %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}

	// Decode signature
	signature, err := decodeBase64(spl.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	// Create canonical byte representation
	payload, err := spl.canonicalBytes()
	if err != nil {
		return fmt.Errorf("canonical bytes: %w", err)
	}

	// Verify
	if !ed25519.Verify(publicKey, payload, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// ToPeerAddrInfos converts the signed peer list to libp2p peer.AddrInfo slice.
func (spl *SignedPeerList) ToPeerAddrInfos() ([]peer.AddrInfo, error) {
	result := make([]peer.AddrInfo, 0, len(spl.Peers))

	for _, entry := range spl.Peers {
		// Parse peer ID
		peerID, err := peer.Decode(entry.ID)
		if err != nil {
			continue // Skip invalid peer ID
		}

		// Parse multiaddrs
		addrs := make([]multiaddr.Multiaddr, 0, len(entry.Addrs))
		for _, addrStr := range entry.Addrs {
			ma, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				continue // Skip invalid multiaddr
			}
			addrs = append(addrs, ma)
		}

		if len(addrs) > 0 {
			result = append(result, peer.AddrInfo{
				ID:    peerID,
				Addrs: addrs,
			})
		}
	}

	return result, nil
}

// FromPeerAddrInfos creates a SignedPeerList from peer.AddrInfo slice.
func FromPeerAddrInfos(peers []peer.AddrInfo) *SignedPeerList {
	entries := make([]PeerEntry, 0, len(peers))

	for _, p := range peers {
		if p.ID == "" || len(p.Addrs) == 0 {
			continue
		}

		addrs := make([]string, len(p.Addrs))
		for i, ma := range p.Addrs {
			addrs[i] = ma.String()
		}

		entries = append(entries, PeerEntry{
			ID:    p.ID.String(),
			Addrs: addrs,
			Seen:  time.Now().Unix(),
		})
	}

	return &SignedPeerList{
		Version: 1,
		Peers:   entries,
	}
}

// PruneStale removes entries older than maxAge.
// Per PLAN.md: "ci-aggregate job prunes entries older than --max-age=24h".
func (spl *SignedPeerList) PruneStale(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge).Unix()
	filtered := make([]PeerEntry, 0, len(spl.Peers))

	for _, entry := range spl.Peers {
		if entry.Seen >= cutoff {
			filtered = append(filtered, entry)
		}
	}

	spl.Peers = filtered
}

// canonicalBytes returns a canonical byte representation for signing/verification.
// Includes version, timestamp, and peers, but excludes signature fields.
func (spl *SignedPeerList) canonicalBytes() ([]byte, error) {
	// Create a copy without signature fields
	canonical := struct {
		Version   int         `json:"version"`
		Timestamp int64       `json:"timestamp"`
		Peers     []PeerEntry `json:"peers"`
	}{
		Version:   spl.Version,
		Timestamp: spl.Timestamp,
		Peers:     spl.Peers,
	}

	return json.Marshal(canonical)
}

// encodeBase64 encodes bytes to base64 string (standard encoding).
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// decodeBase64 decodes base64 string to bytes.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
