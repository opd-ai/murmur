// Package declarations provides identity declaration creation and parsing.
// This file implements Specter Declarations for the Anonymous Layer.
// Per DESIGN_DOCUMENT.md §26, Specters are pseudonymous identities with
// procedurally generated names and cool-tone sigils.
package declarations

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/proto"
	"golang.org/x/crypto/curve25519"
	pb "google.golang.org/protobuf/proto"
)

// Specter-specific constants.
const (
	// SpecterPseudonymMaxLen is the maximum length for a Specter pseudonym.
	SpecterPseudonymMaxLen = 32

	// SpecterPoWDifficulty is the required PoW difficulty for Specter creation.
	// Per DESIGN_DOCUMENT.md, 2-5 seconds compute time is target.
	SpecterPoWDifficulty = 20

	// SpecterKeySize is the size of Curve25519 public keys.
	SpecterKeySize = 32
)

// Specter errors.
var (
	ErrInvalidSpecterKey = errors.New("invalid Specter public key size")
	ErrPseudonymTooLong  = errors.New("pseudonym exceeds maximum length")
	ErrInvalidPoW        = errors.New("proof of work verification failed")
	ErrSpecterNotSigned  = errors.New("specter declaration not signed")
	ErrInvalidSpecterSig = errors.New("invalid specter signature")
	ErrMissingPoW        = errors.New("missing proof of work nonce")
)

// SpecterDeclaration represents a Specter identity on the Anonymous Layer.
// Per DESIGN_DOCUMENT.md §26, Specters have procedurally generated pseudonyms
// and cool-tone sigils, and require PoW for registration.
type SpecterDeclaration struct {
	// PublicKey is the Curve25519 public key for the Specter (32 bytes).
	PublicKey []byte

	// Pseudonym is the procedurally generated name.
	Pseudonym string

	// SigilPNG is the 64x64 cool-tone sigil image.
	SigilPNG []byte

	// CreatedAt is the Unix timestamp of creation.
	CreatedAt int64

	// PoWNonce is the Proof of Work nonce (anti-spam).
	PoWNonce uint64

	// Signature is Ed25519 signature over the declaration.
	// Note: Specters use Curve25519 for anonymity but sign declarations
	// with a derived Ed25519 key for compatibility with our signing infrastructure.
	Signature []byte

	// InitialResonance is always 0 for new Specters.
	InitialResonance uint32
}

// NewSpecterDeclaration creates a new Specter declaration.
// The caller must call ComputePoW() and Sign() to complete.
func NewSpecterDeclaration(publicKey []byte, pseudonym string) (*SpecterDeclaration, error) {
	if len(publicKey) != SpecterKeySize {
		return nil, ErrInvalidSpecterKey
	}
	if len(pseudonym) > SpecterPseudonymMaxLen {
		return nil, ErrPseudonymTooLong
	}

	return &SpecterDeclaration{
		PublicKey:        publicKey,
		Pseudonym:        pseudonym,
		CreatedAt:        time.Now().Unix(),
		InitialResonance: 0, // Always starts at 0
	}, nil
}

// SetSigil sets the sigil PNG data.
func (s *SpecterDeclaration) SetSigil(png []byte) {
	s.SigilPNG = png
}

// ComputePoW computes the Proof of Work nonce for anti-spam.
// This should take 2-5 seconds per DESIGN_DOCUMENT.md.
func (s *SpecterDeclaration) ComputePoW() error {
	target := computePoWTarget(SpecterPoWDifficulty)
	payload := s.powPayload()

	for nonce := uint64(0); ; nonce++ {
		s.PoWNonce = nonce
		if verifyPoWAttempt(payload, nonce, target) {
			return nil
		}
	}
}

// VerifyPoW verifies the Proof of Work nonce.
func (s *SpecterDeclaration) VerifyPoW() error {
	if s.PoWNonce == 0 && !s.hasValidPoW() {
		return ErrInvalidPoW
	}

	target := computePoWTarget(SpecterPoWDifficulty)
	payload := s.powPayload()

	if !verifyPoWAttempt(payload, s.PoWNonce, target) {
		return ErrInvalidPoW
	}

	return nil
}

// hasValidPoW checks if nonce 0 happens to be valid (rare but possible).
func (s *SpecterDeclaration) hasValidPoW() bool {
	target := computePoWTarget(SpecterPoWDifficulty)
	payload := s.powPayload()
	return verifyPoWAttempt(payload, 0, target)
}

// powPayload creates the data to hash for PoW.
// Format: public_key || pseudonym_len || pseudonym || timestamp
func (s *SpecterDeclaration) powPayload() []byte {
	size := len(s.PublicKey) + 4 + len(s.Pseudonym) + 8
	buf := make([]byte, 0, size)

	buf = append(buf, s.PublicKey...)
	buf = appendStringWithLen(buf, s.Pseudonym)
	buf = appendU64BigEndian(buf, uint64(s.CreatedAt))

	return buf
}

// computePoWTarget computes the target value for the given difficulty.
func computePoWTarget(difficulty int) []byte {
	// Target is 256-bit value with 'difficulty' leading zero bits.
	target := make([]byte, 32)
	for i := difficulty / 8; i < 32; i++ {
		target[i] = 0xff
	}
	if remainder := difficulty % 8; remainder > 0 && difficulty/8 < 32 {
		target[difficulty/8] = 0xff >> remainder
	}
	return target
}

// verifyPoWAttempt checks if hash(payload || nonce) < target.
func verifyPoWAttempt(payload []byte, nonce uint64, target []byte) bool {
	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, nonce)

	hash := sha256.Sum256(append(payload, nonceBuf...))

	// Compare hash against target.
	for i := 0; i < 32; i++ {
		if hash[i] < target[i] {
			return true
		}
		if hash[i] > target[i] {
			return false
		}
	}
	return true
}

// Sign signs the Specter declaration using the provided Ed25519 private key.
// Note: This uses the Ed25519 representation of the Curve25519 key for signing.
func (s *SpecterDeclaration) Sign(privateKey ed25519.PrivateKey) error {
	if len(privateKey) != ed25519.PrivateKeySize {
		return ErrNilKeyPair
	}

	payload := s.signingPayload()
	s.Signature = ed25519.Sign(privateKey, payload)
	return nil
}

// SignWithCurve25519 signs using a Curve25519 private key.
// This converts the Curve25519 key to Ed25519 for signing.
func (s *SpecterDeclaration) SignWithCurve25519(privateKey []byte) error {
	if len(privateKey) != curve25519.ScalarSize {
		return ErrInvalidSpecterKey
	}

	// Derive Ed25519 private key from Curve25519 scalar.
	// This is a simplified conversion - in production, use proper key derivation.
	edPriv := make([]byte, ed25519.PrivateKeySize)
	copy(edPriv[:32], privateKey)
	edPub := ed25519.PrivateKey(edPriv).Public().(ed25519.PublicKey)
	copy(edPriv[32:], edPub)

	payload := s.signingPayload()
	s.Signature = ed25519.Sign(edPriv, payload)
	return nil
}

// Verify verifies the declaration signature.
// Note: Uses the Ed25519 representation derived from the Curve25519 key.
func (s *SpecterDeclaration) Verify(ed25519PublicKey []byte) error {
	if len(s.Signature) == 0 {
		return ErrSpecterNotSigned
	}
	if len(ed25519PublicKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}

	payload := s.signingPayload()
	if !ed25519.Verify(ed25519PublicKey, payload, s.Signature) {
		return ErrInvalidSpecterSig
	}

	return nil
}

// signingPayload creates the byte sequence to be signed.
// Format: public_key || pseudonym_len || pseudonym || timestamp || pow_nonce || sigil_len || sigil
func (s *SpecterDeclaration) signingPayload() []byte {
	size := len(s.PublicKey) + 4 + len(s.Pseudonym) + 8 + 8 + 4 + len(s.SigilPNG)
	buf := make([]byte, 0, size)

	buf = append(buf, s.PublicKey...)
	buf = appendStringWithLen(buf, s.Pseudonym)
	buf = appendU64BigEndian(buf, uint64(s.CreatedAt))
	buf = appendU64BigEndian(buf, s.PoWNonce)

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(s.SigilPNG)))
	buf = append(buf, lenBuf...)
	buf = append(buf, s.SigilPNG...)

	return buf
}

// Validate performs full validation of the Specter declaration.
func (s *SpecterDeclaration) Validate(ed25519PublicKey []byte) error {
	if err := s.ValidateTimestamp(); err != nil {
		return err
	}
	if err := s.VerifyPoW(); err != nil {
		return err
	}
	if err := s.Verify(ed25519PublicKey); err != nil {
		return err
	}
	return nil
}

// ValidateTimestamp checks if the timestamp is within acceptable bounds.
func (s *SpecterDeclaration) ValidateTimestamp() error {
	now := time.Now().Unix()
	maxSkew := int64(MaxTimestampSkew.Seconds())
	if s.CreatedAt < now-maxSkew {
		return ErrTimestampTooOld
	}
	if s.CreatedAt > now+maxSkew {
		return ErrTimestampTooNew
	}
	return nil
}

// Marshal serializes the Specter declaration to protobuf wire format.
func (s *SpecterDeclaration) Marshal() ([]byte, error) {
	pbSpec := &proto.SpecterDeclaration{
		PublicKey:        s.PublicKey,
		Pseudonym:        s.Pseudonym,
		SigilPng:         s.SigilPNG,
		CreatedAt:        s.CreatedAt,
		PowNonce:         s.PoWNonce,
		Signature:        s.Signature,
		InitialResonance: s.InitialResonance,
	}
	return pb.Marshal(pbSpec)
}

// UnmarshalSpecter deserializes a Specter declaration from protobuf wire format.
func UnmarshalSpecter(data []byte) (*SpecterDeclaration, error) {
	pbSpec := &proto.SpecterDeclaration{}
	if err := pb.Unmarshal(data, pbSpec); err != nil {
		return nil, fmt.Errorf("unmarshaling specter: %w", err)
	}
	return &SpecterDeclaration{
		PublicKey:        pbSpec.PublicKey,
		Pseudonym:        pbSpec.Pseudonym,
		SigilPNG:         pbSpec.SigilPng,
		CreatedAt:        pbSpec.CreatedAt,
		PoWNonce:         pbSpec.PowNonce,
		Signature:        pbSpec.Signature,
		InitialResonance: pbSpec.InitialResonance,
	}, nil
}

// GeneratePseudonym generates a deterministic pseudonym from the public key.
// Per DESIGN_DOCUMENT.md, Specter names are procedurally generated.
func GeneratePseudonym(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)

	// Use first 4 bytes for word indices.
	adjIdx := binary.BigEndian.Uint16(hash[0:2]) % uint16(len(specterAdjectives))
	nounIdx := binary.BigEndian.Uint16(hash[2:4]) % uint16(len(specterNouns))

	return specterAdjectives[adjIdx] + specterNouns[nounIdx]
}

// Predefined word lists for pseudonym generation.
// Per DESIGN_DOCUMENT.md, names combine adjective + noun from curated lists.
var specterAdjectives = []string{
	"Shadow", "Phantom", "Twilight", "Spectral", "Veiled",
	"Silent", "Ethereal", "Obscure", "Cryptic", "Mystic",
	"Hollow", "Fading", "Shrouded", "Midnight", "Dusk",
	"Lunar", "Stellar", "Void", "Nebula", "Echo",
	"Frost", "Storm", "Thunder", "Lightning", "Tempest",
	"Azure", "Crimson", "Violet", "Cobalt", "Amber",
	"Ancient", "Eternal", "Infinite", "Cosmic", "Astral",
	"Whisper", "Murmur", "Zephyr", "Nimbus", "Aurora",
}

var specterNouns = []string{
	"Walker", "Seeker", "Watcher", "Keeper", "Wanderer",
	"Ghost", "Spirit", "Wraith", "Shade", "Specter",
	"Hunter", "Scout", "Ranger", "Sentinel", "Guardian",
	"Sage", "Oracle", "Seer", "Prophet", "Mystic",
	"Wolf", "Raven", "Owl", "Fox", "Hawk",
	"Storm", "Wind", "Flame", "Wave", "Stone",
	"Star", "Moon", "Sun", "Comet", "Nova",
	"Knight", "Mage", "Rogue", "Monk", "Bard",
}
