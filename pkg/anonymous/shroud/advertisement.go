// Package shroud - Relay advertisement broadcasting and processing.
// Per SHADOW_GRADIENT.md, relays advertise availability via Beacon Waves
// on /murmur/shroud/1 topic.
package shroud

import (
	"crypto/ed25519"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

// AdvertisementTTL is how long relay advertisements are valid.
const AdvertisementTTL = 15 * time.Minute

// GenerateAdvertisement creates a signed relay advertisement.
// Returns nil if this node is not configured as a relay.
func (b *Beacon) GenerateAdvertisement(ed25519PubKey, ed25519PrivKey []byte, addrs []string) *pb.RelayAdvertisement {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.isRelay || b.selfInfo == nil {
		return nil
	}

	now := time.Now()
	ad := &pb.RelayAdvertisement{
		Curve25519Pubkey: b.publicKey[:],
		Ed25519Pubkey:    ed25519PubKey,
		Addrs:            addrs,
		Roles:            []pb.RelayRole{pb.RelayRole_RELAY_ROLE_ENTRY, pb.RelayRole_RELAY_ROLE_MIDDLE, pb.RelayRole_RELAY_ROLE_EXIT},
		Bandwidth:        b.selfInfo.Bandwidth,
		Timestamp:        now.Unix(),
		ExpiresAt:        now.Add(AdvertisementTTL).Unix(),
	}

	// Sign the advertisement.
	signatureData := advertisementSignatureData(ad)
	ad.Signature = ed25519.Sign(ed25519PrivKey, signatureData)

	return ad
}

// ValidateAdvertisement checks a relay advertisement's signature and expiry.
func ValidateAdvertisement(ad *pb.RelayAdvertisement) error {
	if ad == nil {
		return ErrInvalidPacket
	}

	// Check expiry.
	now := time.Now().Unix()
	if ad.ExpiresAt < now {
		return ErrRelayNotFound // Expired.
	}

	// Check timestamp not too far in future.
	if ad.Timestamp > now+300 {
		return ErrInvalidPacket // Future timestamp.
	}

	// Verify signature.
	if len(ad.Ed25519Pubkey) != ed25519.PublicKeySize {
		return ErrInvalidPacket
	}
	if len(ad.Signature) != ed25519.SignatureSize {
		return ErrInvalidPacket
	}

	signatureData := advertisementSignatureData(ad)
	if !ed25519.Verify(ad.Ed25519Pubkey, signatureData, ad.Signature) {
		return ErrDecryptionFailed
	}

	// Validate Curve25519 public key.
	if len(ad.Curve25519Pubkey) != 32 {
		return ErrInvalidPacket
	}

	return nil
}

// ProcessAdvertisement validates and adds a relay from an advertisement.
// Returns the RelayInfo if successful, or nil if validation fails.
func (b *Beacon) ProcessAdvertisement(ad *pb.RelayAdvertisement, peerID string) *RelayInfo {
	if err := ValidateAdvertisement(ad); err != nil {
		return nil
	}

	var pubKey [32]byte
	copy(pubKey[:], ad.Curve25519Pubkey)

	info := &RelayInfo{
		PeerID:    peerID,
		PublicKey: pubKey,
		Bandwidth: ad.Bandwidth,
		SeenAt:    time.Now(),
	}

	b.AddRelay(info)
	return info
}

// advertisementSignatureData generates the data to sign for an advertisement.
// Signature covers: curve25519_pubkey || ed25519_pubkey || addrs || roles || bandwidth || timestamp || expires_at.
func advertisementSignatureData(ad *pb.RelayAdvertisement) []byte {
	var data []byte
	data = append(data, ad.Curve25519Pubkey...)
	data = append(data, ad.Ed25519Pubkey...)
	for _, addr := range ad.Addrs {
		data = append(data, []byte(addr)...)
	}
	for _, role := range ad.Roles {
		data = append(data, byte(role))
	}
	data = append(data, byte(ad.Bandwidth>>56), byte(ad.Bandwidth>>48), byte(ad.Bandwidth>>40), byte(ad.Bandwidth>>32))
	data = append(data, byte(ad.Bandwidth>>24), byte(ad.Bandwidth>>16), byte(ad.Bandwidth>>8), byte(ad.Bandwidth))
	data = append(data, byte(ad.Timestamp>>56), byte(ad.Timestamp>>48), byte(ad.Timestamp>>40), byte(ad.Timestamp>>32))
	data = append(data, byte(ad.Timestamp>>24), byte(ad.Timestamp>>16), byte(ad.Timestamp>>8), byte(ad.Timestamp))
	data = append(data, byte(ad.ExpiresAt>>56), byte(ad.ExpiresAt>>48), byte(ad.ExpiresAt>>40), byte(ad.ExpiresAt>>32))
	data = append(data, byte(ad.ExpiresAt>>24), byte(ad.ExpiresAt>>16), byte(ad.ExpiresAt>>8), byte(ad.ExpiresAt))
	return data
}

// PruneExpiredRelays removes relays that haven't been seen recently.
func (b *Beacon) PruneExpiredRelays(maxAge time.Duration) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	pruned := 0

	for peerID, info := range b.relays {
		if info.SeenAt.Before(cutoff) {
			delete(b.relays, peerID)
			pruned++
		}
	}

	return pruned
}
