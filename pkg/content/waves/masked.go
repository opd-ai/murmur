// Package waves provides Wave creation, signing, and validation.
// This file implements Masked Wave (type 0x07) per WAVES.md.
// Masked Waves are ephemeral messages within Masked Events using single-use keypairs.
package waves

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
)

// MaskedTTL is the fixed TTL for Masked Waves (7 days per WAVES.md).
const MaskedTTL = 7 * 24 * time.Hour

// Metadata keys for Masked Waves.
const (
	MetaMaskedEventID   = "event_id"
	MetaMaskedPseudonym = "masked_pseudonym"
	MetaMaskedKeyHash   = "masked_key_hash"
)

// Errors for Masked Wave operations.
var (
	ErrMissingEventID      = errors.New("event_id is required for Masked Wave")
	ErrInvalidMaskedWave   = errors.New("invalid Masked Wave structure")
	ErrNotMaskedWave       = errors.New("wave is not a Masked Wave")
	ErrMaskedKeyDisposed   = errors.New("masked keypair has been disposed")
	ErrMaskedEventExpired  = errors.New("masked event has expired")
	ErrMaskedWaveMalformed = errors.New("masked wave metadata malformed")
)

// MaskedKeypair is an ephemeral Ed25519 keypair for Masked Event participation.
// The private key is zeroed when Dispose() is called.
type MaskedKeypair struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	pseudonym  string
	keyHash    []byte
	disposed   bool
}

// GenerateMaskedKeypair creates a new single-use keypair for a Masked Event.
func GenerateMaskedKeypair() (*MaskedKeypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	keyHash := blake3.Sum256(pub)
	pseudonym := generateMaskedPseudonym(keyHash[:])

	return &MaskedKeypair{
		publicKey:  pub,
		privateKey: priv,
		pseudonym:  pseudonym,
		keyHash:    keyHash[:],
		disposed:   false,
	}, nil
}

// PublicKey returns the public key of the Masked keypair.
func (mk *MaskedKeypair) PublicKey() ed25519.PublicKey {
	if mk == nil || mk.disposed {
		return nil
	}
	return mk.publicKey
}

// Pseudonym returns the two-word pseudonym derived from the public key hash.
func (mk *MaskedKeypair) Pseudonym() string {
	if mk == nil {
		return ""
	}
	return mk.pseudonym
}

// KeyHash returns the BLAKE3 hash of the public key.
func (mk *MaskedKeypair) KeyHash() []byte {
	if mk == nil {
		return nil
	}
	return mk.keyHash
}

// Sign signs data with the private key. Returns nil if disposed.
func (mk *MaskedKeypair) Sign(data []byte) []byte {
	if mk == nil || mk.disposed || mk.privateKey == nil {
		return nil
	}
	return ed25519.Sign(mk.privateKey, data)
}

// Dispose zeros the private key material and marks the keypair as unusable.
// This should be called when the Masked Event ends.
func (mk *MaskedKeypair) Dispose() {
	if mk == nil || mk.disposed {
		return
	}

	// Zero private key material.
	for i := range mk.privateKey {
		mk.privateKey[i] = 0
	}

	mk.privateKey = nil
	mk.disposed = true
}

// IsDisposed returns true if the keypair has been disposed.
func (mk *MaskedKeypair) IsDisposed() bool {
	if mk == nil {
		return true
	}
	return mk.disposed
}

// MaskedOptions configures Masked Wave creation.
type MaskedOptions struct {
	EventID    string
	TTL        time.Duration
	Difficulty uint8
	ParentHash []byte // For replies within the event
}

// DefaultMaskedOptions returns default options for Masked Wave creation.
func DefaultMaskedOptions(eventID string) MaskedOptions {
	return MaskedOptions{
		EventID:    eventID,
		TTL:        MaskedTTL,
		Difficulty: pow.DefaultDifficulty,
		ParentHash: nil,
	}
}

// CreateMasked creates a new Masked Wave for a Masked Event.
// The wave is signed with the provided single-use Masked keypair.
func CreateMasked(content []byte, mk *MaskedKeypair, opts MaskedOptions) (*pb.Wave, error) {
	if err := validateMaskedParams(content, mk, opts); err != nil {
		return nil, err
	}

	wave := buildMaskedWave(content, mk, opts)
	if err := signMaskedWave(wave, mk, opts.Difficulty); err != nil {
		return nil, err
	}

	return wave, nil
}

// validateMaskedParams checks prerequisites for Masked Wave creation.
func validateMaskedParams(content []byte, mk *MaskedKeypair, opts MaskedOptions) error {
	if mk == nil {
		return ErrNilKeyPair
	}
	if mk.disposed {
		return ErrMaskedKeyDisposed
	}
	if len(content) > MaxContentSize {
		return ErrContentTooLarge
	}
	if opts.EventID == "" {
		return ErrMissingEventID
	}
	if opts.TTL <= 0 {
		return ErrInvalidTTL
	}
	// Masked Waves have a maximum TTL of MaskedTTL (7 days).
	if opts.TTL > MaskedTTL {
		opts.TTL = MaskedTTL
	}
	return nil
}

// buildMaskedWave constructs the protobuf Wave with Masked-specific metadata.
func buildMaskedWave(content []byte, mk *MaskedKeypair, opts MaskedOptions) *pb.Wave {
	now := time.Now()

	// Enforce maximum TTL for Masked Waves.
	ttl := opts.TTL
	if ttl > MaskedTTL {
		ttl = MaskedTTL
	}

	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeMasked),
		Content:      content,
		AuthorPubkey: mk.publicKey,
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(ttl.Seconds()),
		ParentHash:   opts.ParentHash,
		HopCount:     0,
		Metadata:     make(map[string][]byte),
	}

	// Set Masked-specific metadata.
	wave.Metadata[MetaMaskedEventID] = []byte(opts.EventID)
	wave.Metadata[MetaMaskedPseudonym] = []byte(mk.pseudonym)
	wave.Metadata[MetaMaskedKeyHash] = mk.keyHash

	wave.WaveId = computeMaskedWaveID(wave)
	return wave
}

// computeMaskedWaveID generates a BLAKE3 hash including event-specific data.
func computeMaskedWaveID(wave *pb.Wave) []byte {
	h := blake3.New()

	h.Write([]byte{byte(wave.WaveType)})
	h.Write(wave.Content)
	h.Write(wave.AuthorPubkey)

	ts := int64ToBytes(wave.CreatedAt)
	h.Write(ts[:])

	// Include event ID in hash for scoping.
	if eventID, ok := wave.Metadata[MetaMaskedEventID]; ok {
		h.Write(eventID)
	}

	if len(wave.ParentHash) > 0 {
		h.Write(wave.ParentHash)
	}

	return h.Sum(nil)
}

// signMaskedWave signs the wave and computes proof of work.
func signMaskedWave(wave *pb.Wave, mk *MaskedKeypair, difficulty uint8) error {
	sigData := signatureData(wave)
	wave.Signature = mk.Sign(sigData)
	if wave.Signature == nil {
		return ErrMaskedKeyDisposed
	}

	powInput := powData(wave)
	work, err := pow.Compute(powInput, difficulty)
	if err != nil {
		return err
	}
	wave.PowNonce = work.Nonce
	return nil
}

// IsMasked checks if a Wave is a Masked Wave.
func IsMasked(wave *pb.Wave) bool {
	if wave == nil {
		return false
	}
	return wave.WaveType == pb.WaveType(TypeMasked) && wave.Metadata != nil &&
		len(wave.Metadata[MetaMaskedEventID]) > 0
}

// ValidateMasked validates a Masked Wave.
func ValidateMasked(wave *pb.Wave, difficulty uint8) error {
	if wave == nil {
		return ErrInvalidMaskedWave
	}
	if wave.WaveType != pb.WaveType(TypeMasked) {
		return ErrNotMaskedWave
	}

	// Validate metadata presence.
	if wave.Metadata == nil || len(wave.Metadata[MetaMaskedEventID]) == 0 {
		return ErrMissingEventID
	}

	// Check TTL is within limits.
	maxTTL := int64(MaskedTTL.Seconds())
	if wave.TtlSeconds > maxTTL {
		return ErrTTLTooLong
	}

	// Use standard validation for common checks.
	return Validate(wave, difficulty)
}

// GetMaskedEventID returns the event ID from a Masked Wave.
func GetMaskedEventID(wave *pb.Wave) string {
	if wave == nil || wave.Metadata == nil {
		return ""
	}
	return string(wave.Metadata[MetaMaskedEventID])
}

// GetMaskedPseudonym returns the masked pseudonym from a Masked Wave.
func GetMaskedPseudonym(wave *pb.Wave) string {
	if wave == nil || wave.Metadata == nil {
		return ""
	}
	return string(wave.Metadata[MetaMaskedPseudonym])
}

// GetMaskedKeyHash returns the key hash from a Masked Wave.
func GetMaskedKeyHash(wave *pb.Wave) []byte {
	if wave == nil || wave.Metadata == nil {
		return nil
	}
	return wave.Metadata[MetaMaskedKeyHash]
}

// generateMaskedPseudonym generates a two-word pseudonym from a key hash.
// Uses a different wordlist than Specters per WAVES.md to prevent confusion.
func generateMaskedPseudonym(keyHash []byte) string {
	if len(keyHash) < 4 {
		return "Unknown Mask"
	}

	// Use first 2 bytes for adjective, next 2 for noun.
	adjIdx := (int(keyHash[0]) << 8) | int(keyHash[1])
	nounIdx := (int(keyHash[2]) << 8) | int(keyHash[3])

	adjective := maskedAdjectives[adjIdx%len(maskedAdjectives)]
	noun := maskedNouns[nounIdx%len(maskedNouns)]

	return adjective + " " + noun
}

// Wordlists for Masked pseudonyms (different from Specter wordlist).
// These are intentionally different to prevent confusion between Specters and Masked identities.
var maskedAdjectives = []string{
	"Veiled", "Hidden", "Cloaked", "Shrouded", "Obscured",
	"Masked", "Secret", "Covert", "Faded", "Dim",
	"Muted", "Hushed", "Silent", "Quiet", "Subtle",
	"Elusive", "Fleeting", "Brief", "Swift", "Passing",
	"Unseen", "Invisible", "Phantom", "Ghostly", "Spectral",
	"Misty", "Hazy", "Foggy", "Clouded", "Murky",
	"Shadow", "Dusk", "Twilight", "Night", "Dark",
}

var maskedNouns = []string{
	"Visitor", "Guest", "Stranger", "Wanderer", "Traveler",
	"Figure", "Form", "Shape", "Outline", "Silhouette",
	"Whisper", "Echo", "Voice", "Sound", "Murmur",
	"Dream", "Vision", "Glimpse", "Glance", "Moment",
	"Breeze", "Wind", "Gust", "Draft", "Current",
	"Spark", "Flicker", "Glow", "Light", "Flame",
	"Tide", "Wave", "Ripple", "Flow", "Stream",
}

// CreateMaskedReply creates a reply to another Wave within the same Masked Event.
func CreateMaskedReply(content, parentHash []byte, mk *MaskedKeypair, eventID string) (*pb.Wave, error) {
	opts := DefaultMaskedOptions(eventID)
	opts.ParentHash = parentHash
	return CreateMasked(content, mk, opts)
}

// MaskedEventParticipant tracks a participant in a Masked Event.
type MaskedEventParticipant struct {
	Keypair      *MaskedKeypair
	EventID      string
	JoinedAt     time.Time
	EventEndTime time.Time
}

// NewMaskedEventParticipant creates a new participant for a Masked Event.
func NewMaskedEventParticipant(eventID string, eventEndTime time.Time) (*MaskedEventParticipant, error) {
	kp, err := GenerateMaskedKeypair()
	if err != nil {
		return nil, err
	}

	return &MaskedEventParticipant{
		Keypair:      kp,
		EventID:      eventID,
		JoinedAt:     time.Now(),
		EventEndTime: eventEndTime,
	}, nil
}

// CreateWave creates a Masked Wave as this participant.
func (p *MaskedEventParticipant) CreateWave(content []byte) (*pb.Wave, error) {
	if p.IsEventExpired() {
		return nil, ErrMaskedEventExpired
	}
	return CreateMasked(content, p.Keypair, DefaultMaskedOptions(p.EventID))
}

// CreateReply creates a reply Wave as this participant.
func (p *MaskedEventParticipant) CreateReply(content, parentHash []byte) (*pb.Wave, error) {
	if p.IsEventExpired() {
		return nil, ErrMaskedEventExpired
	}
	return CreateMaskedReply(content, parentHash, p.Keypair, p.EventID)
}

// IsEventExpired returns true if the Masked Event has ended.
func (p *MaskedEventParticipant) IsEventExpired() bool {
	if p == nil {
		return true
	}
	return time.Now().After(p.EventEndTime)
}

// LeaveEvent disposes the keypair, making further participation impossible.
func (p *MaskedEventParticipant) LeaveEvent() {
	if p == nil || p.Keypair == nil {
		return
	}
	p.Keypair.Dispose()
}

// Pseudonym returns the participant's masked pseudonym.
func (p *MaskedEventParticipant) Pseudonym() string {
	if p == nil || p.Keypair == nil {
		return ""
	}
	return p.Keypair.Pseudonym()
}
