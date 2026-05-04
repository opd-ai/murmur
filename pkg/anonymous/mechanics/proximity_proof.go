// Package mechanics - DHT-based proximity proofs for Specter Hunts.
// Per ANONYMOUS_GAME_MECHANICS.md, proof-of-proximity is "a signed attestation
// from a peer within 3 hops of the fragment's target node, or a gossip-observable
// proof that the claiming Specter recently exchanged messages with nodes near the target."
package mechanics

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"math/big"
	"time"

	"github.com/zeebo/blake3"
)

// Proximity proof errors.
var (
	ErrProofExpired          = errors.New("proximity proof has expired")
	ErrInvalidAttestation    = errors.New("invalid attestation signature")
	ErrInsufficientProximity = errors.New("attester not close enough to target")
	ErrNoAttestations        = errors.New("no valid attestations provided")
	ErrSelfAttestation       = errors.New("claimer cannot attest their own proximity")
)

// ProximityProofTTL is how long a proximity attestation remains valid.
const ProximityProofTTL = 5 * time.Minute

// HuntClaimProximityHops is the maximum number of hops allowed for hunt claims.
const HuntClaimProximityHops = 3

// ProximityAttestation is a signed statement from a peer that
// they observed the claimer near a target location.
type ProximityAttestation struct {
	AttesterPubKey [32]byte // Ed25519 public key of attester.
	AttesterPeerID string   // libp2p peer ID of attester.
	ClaimerPubKey  [32]byte // Public key of the claimer being attested.
	TargetHash     [32]byte // Fragment location hash being attested.
	Timestamp      int64    // Unix timestamp when attestation was created.
	XORDistance    []byte   // XOR distance from attester to target (big-endian).
	Signature      [64]byte // Ed25519 signature over attestation data.
}

// NewProximityAttestation creates a new signed attestation.
func NewProximityAttestation(
	attesterPrivKey ed25519.PrivateKey,
	attesterPeerID string,
	claimerPubKey [32]byte,
	targetHash [32]byte,
	attesterXORDistance *big.Int,
) *ProximityAttestation {
	att := &ProximityAttestation{
		AttesterPeerID: attesterPeerID,
		ClaimerPubKey:  claimerPubKey,
		TargetHash:     targetHash,
		Timestamp:      time.Now().Unix(),
		XORDistance:    attesterXORDistance.Bytes(),
	}

	// Extract public key from private key.
	copy(att.AttesterPubKey[:], attesterPrivKey.Public().(ed25519.PublicKey))

	// Sign the attestation.
	sigData := att.signatureData()
	sig := ed25519.Sign(attesterPrivKey, sigData)
	copy(att.Signature[:], sig)

	return att
}

// signatureData returns the data that is signed.
func (a *ProximityAttestation) signatureData() []byte {
	h := blake3.New()
	h.Write([]byte("proximity-attestation-v1"))
	h.Write(a.AttesterPubKey[:])
	h.Write([]byte(a.AttesterPeerID))
	h.Write(a.ClaimerPubKey[:])
	h.Write(a.TargetHash[:])
	binary.Write(h, binary.BigEndian, a.Timestamp)
	h.Write(a.XORDistance)
	return h.Sum(nil)
}

// Verify checks if the attestation signature is valid.
func (a *ProximityAttestation) Verify() bool {
	sigData := a.signatureData()
	return ed25519.Verify(a.AttesterPubKey[:], sigData, a.Signature[:])
}

// IsExpired checks if the attestation has exceeded its TTL.
func (a *ProximityAttestation) IsExpired() bool {
	created := time.Unix(a.Timestamp, 0)
	return time.Since(created) > ProximityProofTTL
}

// GetXORDistance returns the XOR distance as a big.Int.
func (a *ProximityAttestation) GetXORDistance() *big.Int {
	return new(big.Int).SetBytes(a.XORDistance)
}

// DHTProximityProof is a proof that the claimer is within
// a certain number of hops from a target location, verified
// using DHT routing table topology.
type DHTProximityProof struct {
	ClaimerPubKey    [32]byte               // Public key of claimer.
	ClaimerPeerID    string                 // libp2p peer ID of claimer.
	TargetHash       [32]byte               // Fragment location being claimed.
	Attestations     []ProximityAttestation // Signed attestations from nearby peers.
	RoutingTableSize int                    // Size of claimer's DHT routing table.
	Timestamp        int64                  // When this proof was created.
}

// NewDHTProximityProof creates a new DHT-based proximity proof.
func NewDHTProximityProof(
	claimerPubKey [32]byte,
	claimerPeerID string,
	targetHash [32]byte,
	routingTableSize int,
) *DHTProximityProof {
	return &DHTProximityProof{
		ClaimerPubKey:    claimerPubKey,
		ClaimerPeerID:    claimerPeerID,
		TargetHash:       targetHash,
		Attestations:     make([]ProximityAttestation, 0),
		RoutingTableSize: routingTableSize,
		Timestamp:        time.Now().Unix(),
	}
}

// AddAttestation adds an attestation to the proof.
func (p *DHTProximityProof) AddAttestation(att ProximityAttestation) {
	p.Attestations = append(p.Attestations, att)
}

// Verify validates the proximity proof against the target and max hops.
// Returns true if the proof demonstrates the claimer is within maxHops.
func (p *DHTProximityProof) Verify(targetHash [32]byte, maxHops int) bool {
	if !p.hasValidTarget(targetHash) {
		return false
	}

	threshold := calculateXORThreshold(maxHops)
	validCount := p.countValidAttestations(threshold)

	// Per ANONYMOUS_GAME_MECHANICS.md, need attestation from peer within maxHops.
	return validCount > 0
}

// hasValidTarget checks if the proof's target hash matches and has attestations.
func (p *DHTProximityProof) hasValidTarget(targetHash [32]byte) bool {
	return p.TargetHash == targetHash && len(p.Attestations) > 0
}

// countValidAttestations returns the number of valid attestations within the XOR threshold.
func (p *DHTProximityProof) countValidAttestations(threshold *big.Int) int {
	validAttestations := 0
	for i := range p.Attestations {
		att := &p.Attestations[i]
		if p.isAttestationValid(att, threshold) {
			validAttestations++
		}
	}
	return validAttestations
}

// isAttestationValid checks if an attestation is valid for this proof.
func (p *DHTProximityProof) isAttestationValid(att *ProximityAttestation, threshold *big.Int) bool {
	// Skip self-attestations.
	if att.AttesterPubKey == p.ClaimerPubKey {
		return false
	}

	// Verify signature.
	if !att.Verify() {
		return false
	}

	// Check expiration.
	if att.IsExpired() {
		return false
	}

	// Check attestation matches this proof.
	if att.ClaimerPubKey != p.ClaimerPubKey || att.TargetHash != p.TargetHash {
		return false
	}

	// Check XOR distance is within threshold.
	attDistance := att.GetXORDistance()
	return attDistance.Cmp(threshold) <= 0
}

// calculateXORThreshold computes the maximum XOR distance for the given hop count.
// In Kademlia, each hop roughly doubles the XOR distance coverage.
// More hops = larger threshold = accept more distant attesters.
func calculateXORThreshold(maxHops int) *big.Int {
	// For 256-bit keyspace with ~20-bucket Kademlia, each hop covers ~42 bits.
	// With maxHops hops, we can reach distances up to 2^(hops * 42).
	bitsPerHop := 42
	coverableBits := maxHops * bitsPerHop
	if coverableBits > 256 {
		coverableBits = 256
	}

	threshold := new(big.Int).Lsh(big.NewInt(1), uint(coverableBits))
	return threshold
}

// ComputeXORDistance calculates the XOR distance between two 32-byte hashes.
func ComputeXORDistance(a, b [32]byte) *big.Int {
	var xor [32]byte
	for i := 0; i < 32; i++ {
		xor[i] = a[i] ^ b[i]
	}
	return new(big.Int).SetBytes(xor[:])
}

// PeerIDToHash converts a libp2p peer ID to a 32-byte hash for XOR distance.
func PeerIDToHash(peerID string) [32]byte {
	var hash [32]byte
	h := blake3.New()
	h.Write([]byte(peerID))
	copy(hash[:], h.Sum(nil))
	return hash
}

// ProximityVerifier provides methods to verify proximity proofs
// using local DHT routing table information.
type ProximityVerifier struct {
	localPeerID      string
	localPubKey      [32]byte
	routingTableFunc func() []string // Returns peer IDs in routing table.
}

// NewProximityVerifier creates a new proximity verifier.
func NewProximityVerifier(
	localPeerID string,
	localPubKey [32]byte,
	routingTableFunc func() []string,
) *ProximityVerifier {
	return &ProximityVerifier{
		localPeerID:      localPeerID,
		localPubKey:      localPubKey,
		routingTableFunc: routingTableFunc,
	}
}

// CreateAttestation creates a signed attestation for a claimer.
// The attester (this node) vouches that the claimer is near the target.
func (v *ProximityVerifier) CreateAttestation(
	attesterPrivKey ed25519.PrivateKey,
	claimerPubKey [32]byte,
	targetHash [32]byte,
) *ProximityAttestation {
	// Calculate our XOR distance to the target.
	localHash := PeerIDToHash(v.localPeerID)
	distance := ComputeXORDistance(localHash, targetHash)

	return NewProximityAttestation(
		attesterPrivKey,
		v.localPeerID,
		claimerPubKey,
		targetHash,
		distance,
	)
}

// VerifyProof validates a proximity proof using local knowledge.
func (v *ProximityVerifier) VerifyProof(proof *DHTProximityProof, maxHops int) error {
	// Basic validation.
	if len(proof.Attestations) == 0 {
		return ErrNoAttestations
	}

	// Check proof timestamp.
	created := time.Unix(proof.Timestamp, 0)
	if time.Since(created) > ProximityProofTTL {
		return ErrProofExpired
	}

	// Verify the proof using the standard method.
	if !proof.Verify(proof.TargetHash, maxHops) {
		return ErrInsufficientProximity
	}

	return nil
}

// IsNearTarget checks if a peer is within maxHops of a target.
// This uses XOR distance calculation based on Kademlia routing.
func (v *ProximityVerifier) IsNearTarget(peerID string, targetHash [32]byte, maxHops int) bool {
	peerHash := PeerIDToHash(peerID)
	distance := ComputeXORDistance(peerHash, targetHash)
	threshold := calculateXORThreshold(maxHops)
	return distance.Cmp(threshold) <= 0
}

// GetNearbyPeers returns peers from the routing table that are
// within maxHops of the target.
func (v *ProximityVerifier) GetNearbyPeers(targetHash [32]byte, maxHops int) []string {
	if v.routingTableFunc == nil {
		return nil
	}

	peers := v.routingTableFunc()
	var nearby []string

	threshold := calculateXORThreshold(maxHops)
	for _, peerID := range peers {
		peerHash := PeerIDToHash(peerID)
		distance := ComputeXORDistance(peerHash, targetHash)
		if distance.Cmp(threshold) <= 0 {
			nearby = append(nearby, peerID)
		}
	}

	return nearby
}

// LegacyProximityProofAdapter adapts DHTProximityProof to the existing
// ProximityProof is a legacy format for proximity verification used by Specter Hunts.
// This is the simple format that predates DHTProximityProof.
type ProximityProof struct {
	ClaimerPeerID  string   // Claimer's peer ID.
	ConnectedPeers []string // Peers the claimer is connected to.
	HopDistances   []int    // Hop distances from target.
}

// Verify checks if the proximity proof is valid.
func (p ProximityProof) Verify(targetHash [32]byte, maxHops int) bool {
	// Simplified verification: check if any connected peer
	// is within maxHops of the target based on XOR distance.
	// In production, this would use DHT routing to verify proximity.
	if len(p.HopDistances) == 0 {
		return false
	}
	for _, hops := range p.HopDistances {
		if hops <= maxHops {
			return true
		}
	}
	return false
}

// ProximityProof interface used by Hunt.ClaimFragment.
type LegacyProximityProofAdapter struct {
	DHTProof *DHTProximityProof
}

// ToLegacyProof converts a DHTProximityProof to the legacy ProximityProof format.
// This allows gradual migration while maintaining backward compatibility.
func (p *DHTProximityProof) ToLegacyProof() ProximityProof {
	// Extract hop distances from attestations.
	hopDistances := make([]int, 0, len(p.Attestations))
	connectedPeers := make([]string, 0, len(p.Attestations))

	for i := range p.Attestations {
		att := &p.Attestations[i]
		if !att.Verify() || att.IsExpired() {
			continue
		}

		connectedPeers = append(connectedPeers, att.AttesterPeerID)

		// Estimate hop distance from XOR distance.
		distance := att.GetXORDistance()
		hops := estimateHopsFromXOR(distance)
		hopDistances = append(hopDistances, hops)
	}

	return ProximityProof{
		ClaimerPeerID:  p.ClaimerPeerID,
		ConnectedPeers: connectedPeers,
		HopDistances:   hopDistances,
	}
}

// estimateHopsFromXOR estimates the number of hops based on XOR distance.
// In Kademlia, smaller XOR distance = closer = fewer hops.
func estimateHopsFromXOR(distance *big.Int) int {
	if distance == nil {
		return HuntClaimProximityHops + 1 // Too far.
	}

	// Distance 0 or 1 = same bucket = 0 hops.
	bits := distance.BitLen()
	if bits <= 1 {
		return 0 // Same node or adjacent.
	}

	// Each Kademlia bucket covers a certain bit range.
	// More bits in distance = further away = more hops needed.
	// Rough estimate: each hop covers ~42 bits of address space.
	const bitsPerHop = 42
	hops := (bits + bitsPerHop - 1) / bitsPerHop // Round up.
	if hops < 0 {
		hops = 0
	}

	return hops
}

// CreateDHTProofFromRoutingTable creates a proximity proof using
// the local DHT routing table to find attesters.
func CreateDHTProofFromRoutingTable(
	claimerPubKey [32]byte,
	claimerPeerID string,
	targetHash [32]byte,
	routingTable []string,
	attesters map[string]ed25519.PrivateKey, // PeerID -> PrivKey for testing.
) *DHTProximityProof {
	proof := NewDHTProximityProof(
		claimerPubKey,
		claimerPeerID,
		targetHash,
		len(routingTable),
	)

	// Find peers close to target and collect attestations.
	threshold := calculateXORThreshold(HuntClaimProximityHops)
	for _, peerID := range routingTable {
		peerHash := PeerIDToHash(peerID)
		distance := ComputeXORDistance(peerHash, targetHash)

		if distance.Cmp(threshold) <= 0 {
			// This peer is close enough to attest.
			if privKey, ok := attesters[peerID]; ok {
				att := NewProximityAttestation(
					privKey,
					peerID,
					claimerPubKey,
					targetHash,
					distance,
				)
				proof.AddAttestation(*att)
			}
		}
	}

	return proof
}
