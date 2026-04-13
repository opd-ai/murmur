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

// deriveAbyssalKeypairFromNonce derives an Ed25519 keypair from a Specter private key and nonce.
// This is the core derivation: Ed25519_keygen(SHA-256(specter_private_key || abyssal_nonce))
func deriveAbyssalKeypairFromNonce(specterPrivateKey []byte, nonce [32]byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	h := sha256.New()
	h.Write(specterPrivateKey)
	h.Write(nonce[:])
	seed := h.Sum(nil)

	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return publicKey, privateKey
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

	publicKey, privateKey := deriveAbyssalKeypairFromNonce(specterPrivateKey, nonce)

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
	if err := validateAbyssalContent(content, opts); err != nil {
		return nil, err
	}

	keyPair, err := DeriveAbyssalKeyPair(specterPrivateKey)
	if err != nil {
		return nil, err
	}

	wave := buildAbyssalWave(content, keyPair, opts)
	if err := signAndComputeAbyssalPoW(wave, keyPair, opts.Difficulty); err != nil {
		return nil, err
	}

	return &AbyssalWave{
		Wave:    wave,
		KeyPair: keyPair,
		ZKProof: opts.ZKProof,
	}, nil
}

// validateAbyssalContent validates content and TTL for Abyssal Waves.
func validateAbyssalContent(content []byte, opts CreateAbyssalOptions) error {
	if len(content) > MaxContentSize {
		return ErrContentTooLarge
	}
	if opts.TTL <= 0 {
		return ErrInvalidTTL
	}
	if opts.TTL > MaxTTL {
		return ErrTTLTooLong
	}
	return nil
}

// buildAbyssalWave constructs the protobuf Wave structure.
func buildAbyssalWave(content []byte, keyPair *AbyssalKeyPair, opts CreateAbyssalOptions) *pb.Wave {
	now := time.Now()
	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeAbyssal),
		Content:      content,
		AuthorPubkey: keyPair.PublicKey,
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(opts.TTL.Seconds()),
		ParentHash:   nil,
		HopCount:     0,
	}
	wave.WaveId = computeWaveID(wave)
	return wave
}

// signAndComputeAbyssalPoW signs the wave and computes proof of work for Abyssal Waves.
// This delegates to the shared signWaveAndComputePoW function.
func signAndComputeAbyssalPoW(wave *pb.Wave, keyPair *AbyssalKeyPair, difficulty uint8) error {
	return signWaveAndComputePoW(wave, keyPair, difficulty)
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

	// Re-derive the key using the shared derivation function.
	publicKey, _ := deriveAbyssalKeypairFromNonce(specterPrivateKey, nonce)

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
	ts := int64ToBytes(wave.CreatedAt)
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
