// Package waves provides Wave creation, signing, and validation.
// Abyssal Waves are the most deeply anonymous Wave type, available only to
// Fortress-mode Specters. They use one-time keypairs derived from the Specter's
// keypair using: abyssal_key = Ed25519_keygen(SHA-256(specter_private_key || abyssal_nonce))
// Per WAVES.md, each Abyssal Wave has a unique, disposable author key.
package waves

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
)

// Abyssal Wave constants per WAVES.md.
const (
	// AbyssalMinResonance is the minimum Specter Resonance for Abyssal Waves.
	AbyssalMinResonance = 50

	// AbyssalNonceSize is the size of the abyssal nonce (32 bytes).
	AbyssalNonceSize = 32

	// AbyssalHigherPoWDifficulty uses higher difficulty than standard Waves.
	// Per WAVES.md, Beacon Waves have 24 bits; Abyssal also elevated.
	AbyssalHigherPoWDifficulty = 22
)

// Abyssal Wave errors.
var (
	ErrAbyssalInsufficientResonance = errors.New(
		"insufficient resonance for abyssal wave")
	ErrAbyssalNotFortress = errors.New("abyssal waves require fortress mode")
	ErrAbyssalInvalidKey  = errors.New("invalid abyssal key derivation")
)

// AbyssalKeyPair represents a one-time keypair for Abyssal Waves.
type AbyssalKeyPair struct {
	// PublicKey is the derived one-time public key.
	PublicKey []byte

	// PrivateKey is the derived one-time private key.
	PrivateKey []byte

	// Nonce is the random nonce used for derivation.
	Nonce [32]byte

	// SpecterPubKey is the parent Specter's public key (for local records).
	SpecterPubKey []byte
}

// DeriveAbyssalKeyPair derives a one-time keypair for an Abyssal Wave.
// Per WAVES.md: abyssal_key = Ed25519_keygen(SHA-256(specter_private_key || abyssal_nonce))
func DeriveAbyssalKeyPair(specterPrivateKey []byte) (*AbyssalKeyPair, error) {
	if len(specterPrivateKey) != ed25519.PrivateKeySize {
		return nil, ErrAbyssalInvalidKey
	}

	// Generate random nonce.
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	// Derive seed: SHA-256(specter_private_key || abyssal_nonce)
	h := sha256.New()
	h.Write(specterPrivateKey)
	h.Write(nonce[:])
	seed := h.Sum(nil)

	// Generate Ed25519 keypair from seed.
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Extract parent Specter public key from private key.
	specterPubKey := specterPrivateKey[32:]

	return &AbyssalKeyPair{
		PublicKey:     publicKey,
		PrivateKey:    privateKey,
		Nonce:         nonce,
		SpecterPubKey: specterPubKey,
	}, nil
}

// Sign signs data with the Abyssal key.
func (akp *AbyssalKeyPair) Sign(data []byte) []byte {
	return ed25519.Sign(akp.PrivateKey, data)
}

// AbyssalWave represents an Abyssal Wave with its one-time key context.
type AbyssalWave struct {
	// Wave is the underlying protobuf Wave.
	Wave *pb.Wave

	// KeyPair is the one-time key used for this Wave.
	KeyPair *AbyssalKeyPair

	// ZKProof is the ZK proof of Fortress mode and Resonance threshold.
	ZKProof []byte
}

// CreateAbyssalOptions configures Abyssal Wave creation.
type CreateAbyssalOptions struct {
	// TTL is the time-to-live.
	TTL time.Duration

	// Difficulty is the PoW difficulty.
	Difficulty uint8

	// ZKProof is the ZK proof of Fortress mode and minimum Resonance.
	ZKProof []byte
}

// DefaultAbyssalOptions returns default options for Abyssal Wave creation.
func DefaultAbyssalOptions() CreateAbyssalOptions {
	return CreateAbyssalOptions{
		TTL:        DefaultTTL,
		Difficulty: AbyssalHigherPoWDifficulty,
		ZKProof:    nil, // Caller must provide.
	}
}

// CreateAbyssal creates a new Abyssal Wave with a one-time keypair.
// The specterPrivateKey is used to derive the one-time key but is not
// exposed in the resulting Wave.
func CreateAbyssal(
	content []byte,
	specterPrivateKey []byte,
	opts CreateAbyssalOptions,
) (*AbyssalWave, error) {
	if len(content) > MaxContentSize {
		return nil, ErrContentTooLarge
	}

	if opts.TTL <= 0 {
		return nil, ErrInvalidTTL
	}

	if opts.TTL > MaxTTL {
		return nil, ErrTTLTooLong
	}

	// Derive one-time keypair.
	keyPair, err := DeriveAbyssalKeyPair(specterPrivateKey)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Create Wave with one-time public key as author.
	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeAbyssal),
		Content:      content,
		AuthorPubkey: keyPair.PublicKey, // One-time key, not Specter key.
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(opts.TTL.Seconds()),
		ParentHash:   nil, // Abyssal Waves cannot be replies.
		HopCount:     0,
	}

	// Compute Wave ID.
	wave.WaveId = computeWaveID(wave)

	// Sign with one-time key.
	sigData := signatureData(wave)
	wave.Signature = keyPair.Sign(sigData)

	// Compute PoW with elevated difficulty.
	powData := powData(wave)
	work, err := pow.Compute(powData, opts.Difficulty)
	if err != nil {
		return nil, err
	}
	wave.PowNonce = work.Nonce

	return &AbyssalWave{
		Wave:    wave,
		KeyPair: keyPair,
		ZKProof: opts.ZKProof,
	}, nil
}

// ValidateAbyssal validates an Abyssal Wave.
// Note: The ZK proof should be verified separately as it requires
// access to the ZK verification primitives.
func ValidateAbyssal(wave *pb.Wave, difficulty uint8) error {
	// Check wave type first.
	if wave != nil && pb.WaveType(TypeAbyssal) != wave.WaveType {
		return errors.New("not an abyssal wave")
	}

	// Perform common validation (content size, expiration, pubkey size, PoW).
	if err := validateCommon(wave, difficulty); err != nil {
		return err
	}

	// Verify signature with one-time key.
	sigData := signatureData(wave)
	if !ed25519.Verify(wave.AuthorPubkey, sigData, wave.Signature) {
		return ErrInvalidSig
	}

	return nil
}

// CanProveAuthorship checks if the holder of the abyssal keypair can prove
// authorship of a specific Wave. This requires the nonce and Specter private key.
func CanProveAuthorship(
	wave *pb.Wave,
	specterPrivateKey []byte,
	nonce [32]byte,
) bool {
	if wave == nil || len(specterPrivateKey) != ed25519.PrivateKeySize {
		return false
	}

	// Re-derive the key.
	h := sha256.New()
	h.Write(specterPrivateKey)
	h.Write(nonce[:])
	seed := h.Sum(nil)

	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Check if derived public key matches Wave author.
	if len(wave.AuthorPubkey) != len(publicKey) {
		return false
	}

	for i := range publicKey {
		if publicKey[i] != wave.AuthorPubkey[i] {
			return false
		}
	}

	return true
}

// AbyssalWaveID computes the ID of an Abyssal Wave.
func AbyssalWaveID(wave *pb.Wave) []byte {
	if wave == nil {
		return nil
	}

	h := blake3.New()

	// Include type and content.
	h.Write([]byte{byte(TypeAbyssal)})
	h.Write(wave.Content)

	// Include one-time author key.
	h.Write(wave.AuthorPubkey)

	// Include creation timestamp.
	var ts [8]byte
	ts[0] = byte(wave.CreatedAt >> 56)
	ts[1] = byte(wave.CreatedAt >> 48)
	ts[2] = byte(wave.CreatedAt >> 40)
	ts[3] = byte(wave.CreatedAt >> 32)
	ts[4] = byte(wave.CreatedAt >> 24)
	ts[5] = byte(wave.CreatedAt >> 16)
	ts[6] = byte(wave.CreatedAt >> 8)
	ts[7] = byte(wave.CreatedAt)
	h.Write(ts[:])

	return h.Sum(nil)
}

// AbyssalStore manages local records of authored Abyssal Waves.
// This allows the author to prove authorship if needed, while keeping
// the nonces private.
type AbyssalStore struct {
	// records maps Wave ID to the nonce used for derivation.
	records map[string][32]byte
}

// NewAbyssalStore creates a new AbyssalStore.
func NewAbyssalStore() *AbyssalStore {
	return &AbyssalStore{
		records: make(map[string][32]byte),
	}
}

// StoreNonce records the nonce used for an Abyssal Wave.
func (s *AbyssalStore) StoreNonce(waveID []byte, nonce [32]byte) {
	if s.records == nil {
		s.records = make(map[string][32]byte)
	}
	s.records[string(waveID)] = nonce
}

// GetNonce retrieves the nonce for an Abyssal Wave.
func (s *AbyssalStore) GetNonce(waveID []byte) ([32]byte, bool) {
	nonce, found := s.records[string(waveID)]
	return nonce, found
}

// RemoveNonce removes the nonce record for an Abyssal Wave.
func (s *AbyssalStore) RemoveNonce(waveID []byte) {
	delete(s.records, string(waveID))
}

// Count returns the number of stored nonces.
func (s *AbyssalStore) Count() int {
	return len(s.records)
}
