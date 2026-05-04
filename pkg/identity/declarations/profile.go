// Package declarations provides identity declaration creation and parsing.
// Per DESIGN_DOCUMENT.md, identity declarations are signed announcements
// of identity information broadcast via GossipSub.
package declarations

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/proto"
	pb "google.golang.org/protobuf/proto"
)

// Constants for declaration validation.
const (
	// MaxTimestampSkew is the maximum allowed clock skew (±300 seconds per spec).
	MaxTimestampSkew = 300 * time.Second

	// MaxDisplayNameLen is the maximum display name length.
	MaxDisplayNameLen = 64

	// MaxBioLen is the maximum bio/description length.
	MaxBioLen = 256
)

// Errors for declaration operations.
var (
	ErrInvalidPublicKey  = errors.New("invalid public key size")
	ErrInvalidSignature  = errors.New("signature verification failed")
	ErrTimestampTooOld   = errors.New("timestamp too old")
	ErrTimestampTooNew   = errors.New("timestamp too new")
	ErrDisplayNameTooLon = errors.New("display name too long")
	ErrBioTooLong        = errors.New("bio too long")
	ErrNilKeyPair        = errors.New("nil keypair provided")
)

// Declaration represents a signed identity announcement.
// Per TECHNICAL_IMPLEMENTATION.md §2.1, declarations are broadcast on
// /murmur/identity/1 topic and stored locally for known peers.
type Declaration struct {
	// PublicKey is the Ed25519 public key of the identity.
	PublicKey []byte

	// DisplayName is the optional human-readable name.
	DisplayName string

	// Bio is an optional description/bio.
	Bio string

	// Timestamp is the Unix timestamp of declaration creation.
	Timestamp int64

	// Version is the declaration version (incremented on updates).
	Version uint32

	// Signature is the Ed25519 signature over the declaration fields.
	Signature []byte

	// SigilPNG is the optional 64x64 sigil image (PNG encoded).
	SigilPNG []byte

	// PrivacyMode is the user's current privacy mode.
	PrivacyMode modes.Mode
}

// New creates a new Declaration with the given keypair.
// The declaration is not signed until Sign() is called.
func New(kp *keys.KeyPair, displayName string) (*Declaration, error) {
	if kp == nil {
		return nil, ErrNilKeyPair
	}
	if len(displayName) > MaxDisplayNameLen {
		return nil, ErrDisplayNameTooLon
	}
	return &Declaration{
		PublicKey:   kp.PublicKey,
		DisplayName: displayName,
		Timestamp:   time.Now().Unix(),
		Version:     1,
		PrivacyMode: modes.Open,
	}, nil
}

// Sign signs the declaration using the provided keypair.
// The signature covers all fields except the signature itself.
func (d *Declaration) Sign(kp *keys.KeyPair) error {
	if kp == nil {
		return ErrNilKeyPair
	}
	message := d.signingPayload()
	d.Signature = kp.Sign(message)
	return nil
}

// Verify verifies the declaration signature.
// Returns nil if valid, or an error describing the failure.
func (d *Declaration) Verify() error {
	if len(d.PublicKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}
	message := d.signingPayload()
	if !keys.Verify(d.PublicKey, message, d.Signature) {
		return ErrInvalidSignature
	}
	return nil
}

// ValidateTimestamp checks if the timestamp is within acceptable bounds.
func (d *Declaration) ValidateTimestamp() error {
	now := time.Now().Unix()
	maxSkew := int64(MaxTimestampSkew.Seconds())
	if d.Timestamp < now-maxSkew {
		return ErrTimestampTooOld
	}
	if d.Timestamp > now+maxSkew {
		return ErrTimestampTooNew
	}
	return nil
}

// PoW constants for identity creation anti-spam.
const (
	// IdentityPoWDifficulty is the PoW difficulty for identity creation.
	// Per DESIGN_DOCUMENT.md, 2-5 seconds compute time is target.
	IdentityPoWDifficulty = 18

	// PoWNonceSize is the size of the PoW nonce in bytes.
	PoWNonceSize = 8
)

// Errors for PoW operations.
var (
	ErrInvalidIdentityPoW = errors.New("invalid identity proof of work")
)

// DeclarationWithPoW extends Declaration with Proof of Work for anti-spam.
// Per DESIGN_DOCUMENT.md, identity creation requires PoW to prevent spam.
type DeclarationWithPoW struct {
	*Declaration

	// PoWNonce is the Proof of Work nonce.
	PoWNonce uint64
}

// NewWithPoW creates a new Declaration that will require PoW.
func NewWithPoW(kp *keys.KeyPair, displayName string) (*DeclarationWithPoW, error) {
	decl, err := New(kp, displayName)
	if err != nil {
		return nil, err
	}
	return &DeclarationWithPoW{Declaration: decl}, nil
}

// ComputePoW computes the Proof of Work nonce for the declaration.
// This should take 2-5 seconds per DESIGN_DOCUMENT.md.
func (d *DeclarationWithPoW) ComputePoW() error {
	payload := d.powPayload()
	target := computeIdentityPoWTarget(IdentityPoWDifficulty)

	for nonce := uint64(0); ; nonce++ {
		d.PoWNonce = nonce
		if verifyIdentityPoWAttempt(payload, nonce, target) {
			return nil
		}
	}
}

// VerifyPoW verifies the Proof of Work nonce.
func (d *DeclarationWithPoW) VerifyPoW() error {
	payload := d.powPayload()
	target := computeIdentityPoWTarget(IdentityPoWDifficulty)

	if !verifyIdentityPoWAttempt(payload, d.PoWNonce, target) {
		return ErrInvalidIdentityPoW
	}
	return nil
}

// powPayload creates the data to hash for PoW.
// Format: public_key || display_name || timestamp
func (d *DeclarationWithPoW) powPayload() []byte {
	size := len(d.PublicKey) + len(d.DisplayName) + 8
	buf := make([]byte, 0, size)

	buf = append(buf, d.PublicKey...)
	buf = append(buf, []byte(d.DisplayName)...)

	tsBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBuf, uint64(d.Timestamp))
	buf = append(buf, tsBuf...)

	return buf
}

// computeIdentityPoWTarget computes the target value for the given difficulty.
func computeIdentityPoWTarget(difficulty int) []byte {
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

// verifyIdentityPoWAttempt checks if sha256(payload || nonce) < target.
func verifyIdentityPoWAttempt(payload []byte, nonce uint64, target []byte) bool {
	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, nonce)

	hash := sha256Hash(append(payload, nonceBuf...))

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

// sha256Hash computes SHA-256 hash.
func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// ValidateWithPoW performs full validation including PoW.
func (d *DeclarationWithPoW) ValidateWithPoW() error {
	if err := d.Validate(); err != nil {
		return err
	}
	if err := d.VerifyPoW(); err != nil {
		return err
	}
	return nil
}

// Validate performs full validation of the declaration.
func (d *Declaration) Validate() error {
	if err := d.Verify(); err != nil {
		return err
	}
	if err := d.ValidateTimestamp(); err != nil {
		return err
	}
	if len(d.DisplayName) > MaxDisplayNameLen {
		return ErrDisplayNameTooLon
	}
	if len(d.Bio) > MaxBioLen {
		return ErrBioTooLong
	}
	return nil
}

// signingPayload creates the byte sequence to be signed.
// Format: public_key || display_name_len || display_name || bio_len || bio ||
// timestamp || version || sigil_len || sigil || privacy_mode
func (d *Declaration) signingPayload() []byte {
	size := len(d.PublicKey) + 4 + len(d.DisplayName) + 4 + len(d.Bio) + 8 + 4 + 4 + len(d.SigilPNG) + 4
	buf := make([]byte, 0, size)

	buf = append(buf, d.PublicKey...)
	buf = appendStringWithLen(buf, d.DisplayName)
	buf = appendStringWithLen(buf, d.Bio)
	buf = appendU64BigEndian(buf, uint64(d.Timestamp))

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, d.Version)
	buf = append(buf, lenBuf...)

	binary.BigEndian.PutUint32(lenBuf, uint32(len(d.SigilPNG)))
	buf = append(buf, lenBuf...)
	buf = append(buf, d.SigilPNG...)

	binary.BigEndian.PutUint32(lenBuf, uint32(d.PrivacyMode))
	buf = append(buf, lenBuf...)

	return buf
}

// Marshal serializes the declaration to protobuf wire format.
func (d *Declaration) Marshal() ([]byte, error) {
	pbDecl := &proto.IdentityDeclaration{
		PublicKey:   d.PublicKey,
		DisplayName: d.DisplayName,
		Bio:         d.Bio,
		CreatedAt:   d.Timestamp,
		Version:     d.Version,
		Signature:   d.Signature,
		SigilPng:    d.SigilPNG,
		PrivacyMode: modeToProto(d.PrivacyMode),
	}
	return pb.Marshal(pbDecl)
}

// Unmarshal deserializes a declaration from protobuf wire format.
func Unmarshal(data []byte) (*Declaration, error) {
	pbDecl := &proto.IdentityDeclaration{}
	if err := pb.Unmarshal(data, pbDecl); err != nil {
		return nil, fmt.Errorf("unmarshaling declaration: %w", err)
	}
	return &Declaration{
		PublicKey:   pbDecl.PublicKey,
		DisplayName: pbDecl.DisplayName,
		Bio:         pbDecl.Bio,
		Timestamp:   pbDecl.CreatedAt,
		Version:     pbDecl.Version,
		Signature:   pbDecl.Signature,
		SigilPNG:    pbDecl.SigilPng,
		PrivacyMode: protoToMode(pbDecl.PrivacyMode),
	}, nil
}

// modeToProto converts a modes.Mode to proto.PrivacyMode.
func modeToProto(m modes.Mode) proto.PrivacyMode {
	switch m {
	case modes.Open:
		return proto.PrivacyMode_PRIVACY_MODE_OPEN
	case modes.Hybrid:
		return proto.PrivacyMode_PRIVACY_MODE_HYBRID
	case modes.Guarded:
		return proto.PrivacyMode_PRIVACY_MODE_GUARDED
	case modes.Fortress:
		return proto.PrivacyMode_PRIVACY_MODE_FORTRESS
	default:
		return proto.PrivacyMode_PRIVACY_MODE_UNSPECIFIED
	}
}

// protoToMode converts a proto.PrivacyMode to modes.Mode.
func protoToMode(pm proto.PrivacyMode) modes.Mode {
	switch pm {
	case proto.PrivacyMode_PRIVACY_MODE_OPEN:
		return modes.Open
	case proto.PrivacyMode_PRIVACY_MODE_HYBRID:
		return modes.Hybrid
	case proto.PrivacyMode_PRIVACY_MODE_GUARDED:
		return modes.Guarded
	case proto.PrivacyMode_PRIVACY_MODE_FORTRESS:
		return modes.Fortress
	default:
		return modes.Open
	}
}

// SetBio sets the bio field with validation.
func (d *Declaration) SetBio(bio string) error {
	if len(bio) > MaxBioLen {
		return ErrBioTooLong
	}
	d.Bio = bio
	return nil
}

// SetSigil sets the sigil PNG data.
func (d *Declaration) SetSigil(png []byte) {
	d.SigilPNG = png
}

// SetPrivacyMode sets the privacy mode.
func (d *Declaration) SetPrivacyMode(mode modes.Mode) {
	d.PrivacyMode = mode
}

// IncrementVersion increments the declaration version for updates.
func (d *Declaration) IncrementVersion() {
	d.Version++
	d.Timestamp = time.Now().Unix()
}

// Update creates an updated declaration with incremented version.
// The caller must call Sign() on the returned declaration.
func (d *Declaration) Update() *Declaration {
	updated := &Declaration{
		PublicKey:   d.PublicKey,
		DisplayName: d.DisplayName,
		Bio:         d.Bio,
		Timestamp:   time.Now().Unix(),
		Version:     d.Version + 1,
		SigilPNG:    d.SigilPNG,
		PrivacyMode: d.PrivacyMode,
	}
	return updated
}
