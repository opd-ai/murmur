// Package specters provides Specter identity creation and name generation.
// Per DESIGN_DOCUMENT.md, Specters are pseudonymous identities with
// procedurally generated names from a wordlist.
package specters

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// Specter status values.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
)

// Errors for Specter operations.
var (
	ErrNilKeyPair       = errors.New("nil keypair")
	ErrSuspended        = errors.New("specter is suspended")
	ErrDeleted          = errors.New("specter is deleted")
	ErrInvalidStatus    = errors.New("invalid status")
	ErrEncryptionFail   = errors.New("encryption failed")
	ErrDecryptionFail   = errors.New("decryption failed")
	ErrAlreadyAnnounced = errors.New("specter already announced")
	ErrNotAnnounced     = errors.New("specter not announced")
	ErrRotationFailed   = errors.New("specter rotation failed")
)

// Specter represents an anonymous identity in the Anonymous Layer.
// Per DESIGN_DOCUMENT.md, Specters use independent Curve25519 keypairs
// with no derivation relationship to Surface identities.
type Specter struct {
	mu           sync.RWMutex
	PrivateKey   [32]byte
	PublicKey    [32]byte
	Name         string
	CreatedAt    time.Time
	Status       string
	Announced    bool     // Whether this Specter has been announced to the network
	RotationFrom [32]byte // Public key of previous identity (for rotation tracking)
	Version      int      // Identity version (increments on rotation)
}

// KeyPair holds a Curve25519 key pair for Shroud circuits.
type KeyPair struct {
	Private [32]byte
	Public  [32]byte
}

// GenerateKeyPair creates a new independent Curve25519 keypair.
// Per SECURITY_PRIVACY.md, Specter keys have no derivation relationship
// to Surface (Ed25519) keys.
func GenerateKeyPair() (*KeyPair, error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, err
	}

	// Clamp the private key per Curve25519 requirements.
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &KeyPair{
		Private: privateKey,
		Public:  publicKey,
	}, nil
}

// NewSpecter creates a new Specter identity with an independent keypair.
// Per SHADOW_GRADIENT.md, the Specter is created locally without network announcement.
// Call Announce() to register the Specter on the Anonymous Layer.
func NewSpecter() (*Specter, error) {
	kp, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	s := &Specter{
		PrivateKey: kp.Private,
		PublicKey:  kp.Public,
		CreatedAt:  time.Now(),
		Status:     StatusActive,
		Announced:  false, // Per SHADOW_GRADIENT.md: created without network announcement
		Version:    1,
	}

	// Generate procedural name from public key.
	s.Name = GenerateName(s.PublicKey[:])

	return s, nil
}

// NewSpecterFromKeyPair creates a Specter from an existing keypair.
// Per SHADOW_GRADIENT.md, the Specter is created locally without network announcement.
func NewSpecterFromKeyPair(kp *KeyPair) (*Specter, error) {
	if kp == nil {
		return nil, ErrNilKeyPair
	}

	s := &Specter{
		PrivateKey: kp.Private,
		PublicKey:  kp.Public,
		CreatedAt:  time.Now(),
		Status:     StatusActive,
		Announced:  false, // Per SHADOW_GRADIENT.md: created without network announcement
		Version:    1,
	}

	s.Name = GenerateName(s.PublicKey[:])

	return s, nil
}

// IsActive returns true if the Specter is active.
func (s *Specter) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Status == StatusActive
}

// Suspend temporarily deactivates the Specter.
func (s *Specter) Suspend() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusDeleted {
		return ErrDeleted
	}

	s.Status = StatusSuspended
	return nil
}

// Activate reactivates a suspended Specter.
func (s *Specter) Activate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusDeleted {
		return ErrDeleted
	}

	s.Status = StatusActive
	return nil
}

// Delete permanently destroys the Specter identity.
// The private key is zeroed for security.
func (s *Specter) Delete() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Zero the private key.
	for i := range s.PrivateKey {
		s.PrivateKey[i] = 0
	}

	s.Status = StatusDeleted
}

// MarkAnnounced marks the Specter as having been announced to the network.
// Per SHADOW_GRADIENT.md, a Specter must be announced before it can be used
// for network operations (Specter Waves, anonymous connections, etc.).
func (s *Specter) MarkAnnounced() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusDeleted {
		return ErrDeleted
	}
	if s.Announced {
		return ErrAlreadyAnnounced
	}

	s.Announced = true
	return nil
}

// IsAnnounced returns true if the Specter has been announced to the network.
func (s *Specter) IsAnnounced() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Announced
}

// Rotate creates a new Specter identity and destroys the current one.
// Per SHADOW_GRADIENT.md, rotation is irreversible and generates a completely
// new identity with no cryptographic link to the previous one.
// Returns the new Specter (which needs to be announced separately).
func (s *Specter) Rotate() (*Specter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusDeleted {
		return nil, ErrDeleted
	}

	// Save old public key for tracking.
	var oldPubKey [32]byte
	copy(oldPubKey[:], s.PublicKey[:])
	oldVersion := s.Version

	// Zero the old private key first.
	for i := range s.PrivateKey {
		s.PrivateKey[i] = 0
	}
	s.Status = StatusDeleted

	// Create new Specter.
	newSpecter, err := NewSpecter()
	if err != nil {
		return nil, ErrRotationFailed
	}

	// Track rotation lineage (not cryptographically linked, just for local reference).
	newSpecter.RotationFrom = oldPubKey
	newSpecter.Version = oldVersion + 1

	return newSpecter, nil
}

// DestroyForModeDowngrade completely destroys the Specter for privacy mode downgrade.
// Per SHADOW_GRADIENT.md, when transitioning from a Specter-enabled mode
// (Hybrid/Guarded/Fortress) to Open mode, the Specter keypair must be
// destroyed to prevent identity correlation.
// This is more thorough than Delete() - it also zeros the public key.
func (s *Specter) DestroyForModeDowngrade() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Zero the private key.
	for i := range s.PrivateKey {
		s.PrivateKey[i] = 0
	}

	// Zero the public key too.
	for i := range s.PublicKey {
		s.PublicKey[i] = 0
	}

	// Zero rotation tracking.
	for i := range s.RotationFrom {
		s.RotationFrom[i] = 0
	}

	// Clear identifiable information.
	s.Name = ""
	s.Status = StatusDeleted
	s.Announced = false
}

// PublicKeyCopy returns a copy of the public key.
func (s *Specter) PublicKeyCopy() [32]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var key [32]byte
	copy(key[:], s.PublicKey[:])
	return key
}

// IdentityVersion returns the identity version (1 for original, 2+ for rotated).
func (s *Specter) IdentityVersion() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Version
}

// RotationSource returns the public key of the previous identity, if any.
// Returns zero value if this is the original identity (not rotated).
func (s *Specter) RotationSource() [32]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var key [32]byte
	copy(key[:], s.RotationFrom[:])
	return key
}

// Deprecated: Use PublicKeyCopy instead.
func (s *Specter) GetPublicKey() [32]byte {
	return s.PublicKeyCopy()
}

// Deprecated: Use IdentityVersion instead.
func (s *Specter) GetVersion() int {
	return s.IdentityVersion()
}

// Deprecated: Use RotationSource instead.
func (s *Specter) GetRotationSource() [32]byte {
	return s.RotationSource()
}

// DeriveSharedSecret performs X25519 key exchange.
func (s *Specter) DeriveSharedSecret(peerPublic []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Status == StatusDeleted {
		return nil, ErrDeleted
	}
	if s.Status == StatusSuspended {
		return nil, ErrSuspended
	}

	if len(peerPublic) != 32 {
		return nil, errors.New("invalid peer public key length")
	}

	var peerPub [32]byte
	copy(peerPub[:], peerPublic)

	var shared [32]byte
	curve25519.ScalarMult(&shared, &s.PrivateKey, &peerPub)

	return shared[:], nil
}

// deriveEncryptionKey derives a 32-byte encryption key from a shared secret.
// Uses BLAKE3 with domain separation per SECURITY_PRIVACY.md.
func deriveEncryptionKey(sharedSecret []byte) []byte {
	h := blake3.New()
	h.Write(sharedSecret)
	h.Write([]byte("murmur-specter-encrypt"))
	return h.Sum(nil)[:32]
}

// Encrypt encrypts data using XChaCha20-Poly1305 with a derived key.
func (s *Specter) Encrypt(plaintext, recipientPublic []byte) ([]byte, error) {
	shared, err := s.DeriveSharedSecret(recipientPublic)
	if err != nil {
		return nil, err
	}

	key := deriveEncryptionKey(shared)
	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, ErrEncryptionFail
	}

	// Generate random nonce.
	nonce := make([]byte, cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce.
	ciphertext := cipher.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// Decrypt decrypts data using XChaCha20-Poly1305 with a derived key.
func (s *Specter) Decrypt(ciphertext, senderPublic []byte) ([]byte, error) {
	shared, err := s.DeriveSharedSecret(senderPublic)
	if err != nil {
		return nil, err
	}

	key := deriveEncryptionKey(shared)
	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, ErrDecryptionFail
	}

	if len(ciphertext) < cipher.NonceSize() {
		return nil, ErrDecryptionFail
	}

	nonce := ciphertext[:cipher.NonceSize()]
	ciphertext = ciphertext[cipher.NonceSize():]

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFail
	}

	return plaintext, nil
}

// GenerateName creates a procedural two-word pseudonym from public key.
// Per DESIGN_DOCUMENT.md, format is "Adjective Noun" using BLAKE3 hash.
func GenerateName(publicKey []byte) string {
	h := blake3.Sum256(publicKey)

	// Use first 8 bytes for adjective index, next 8 for noun index.
	adjIdx := binary.BigEndian.Uint64(h[:8]) % uint64(len(adjectives))
	nounIdx := binary.BigEndian.Uint64(h[8:16]) % uint64(len(nouns))

	return adjectives[adjIdx] + " " + nouns[nounIdx]
}

// GenerateNameWithPrefix generates a name with collision avoidance.
// If the base name collides, it appends a numeric suffix.
func GenerateNameWithPrefix(publicKey []byte, existingNames map[string]bool) string {
	baseName := GenerateName(publicKey)

	if !existingNames[baseName] {
		return baseName
	}

	// Try variations.
	for i := 1; i < 1000; i++ {
		h := blake3.New()
		h.Write(publicKey)
		var suffix [4]byte
		binary.BigEndian.PutUint32(suffix[:], uint32(i))
		h.Write(suffix[:])
		sum := h.Sum(nil)

		adjIdx := binary.BigEndian.Uint64(sum[:8]) % uint64(len(adjectives))
		nounIdx := binary.BigEndian.Uint64(sum[8:16]) % uint64(len(nouns))

		name := adjectives[adjIdx] + " " + nouns[nounIdx]
		if !existingNames[name] {
			return name
		}
	}

	// Fallback: use hex suffix.
	h := blake3.Sum256(publicKey)
	return baseName + "-" + string(h[:4])
}

// Curated wordlists for procedural name generation.
// Per spec, should have 65,536 entries each; using smaller sets for now.
var adjectives = []string{
	"Ancient", "Arcane", "Astral", "Azure", "Bitter", "Blazing", "Bright",
	"Broken", "Burning", "Celestial", "Chaos", "Cinder", "Cosmic", "Crimson",
	"Crystal", "Cursed", "Dark", "Dawn", "Deep", "Dire", "Distant", "Divine",
	"Dread", "Dusk", "Dying", "Echo", "Elder", "Ember", "Endless", "Eternal",
	"Fading", "Fallen", "Feral", "Fierce", "Final", "Fire", "First", "Flame",
	"Flickering", "Floating", "Fog", "Forgotten", "Forsaken", "Frost", "Frozen",
	"Ghost", "Gilded", "Gloom", "Golden", "Grand", "Grave", "Gray", "Grim",
	"Hallowed", "Haunted", "Hidden", "Hollow", "Holy", "Howling", "Icy", "Idle",
	"Ill", "Immortal", "Infernal", "Iron", "Jade", "Keen", "Last", "Lone",
	"Lost", "Lunar", "Mad", "Masked", "Midnight", "Mist", "Moon", "Muted",
	"Night", "Noble", "Obsidian", "Old", "Omen", "Pale", "Phantom", "Prime",
	"Primal", "Quiet", "Radiant", "Raging", "Raven", "Red", "Restless", "Rising",
	"Ruined", "Sacred", "Savage", "Scarlet", "Secret", "Serene", "Shadow",
	"Shattered", "Silent", "Silver", "Sleeping", "Solar", "Solemn", "Somber",
	"Sorrow", "Spectral", "Spirit", "Starless", "Steel", "Stone", "Storm",
	"Strange", "Swift", "Tainted", "Thunder", "Twilight", "Undying", "Unseen",
	"Veil", "Vengeful", "Venom", "Void", "Wandering", "Waning", "Waxing",
	"Weeping", "Wicked", "Wild", "Winter", "Wistful", "Wrath", "Wraith",
}

var nouns = []string{
	"Abyss", "Adept", "Arbiter", "Archer", "Ash", "Aspect", "Baron", "Beacon",
	"Bear", "Beast", "Blade", "Blaze", "Blood", "Bone", "Bringer", "Caller",
	"Carver", "Chant", "Chaser", "Child", "Cipher", "Claim", "Claw", "Cloud",
	"Cobra", "Coil", "Conjurer", "Crest", "Crow", "Crown", "Curse", "Dancer",
	"Dagger", "Dawn", "Death", "Delver", "Depth", "Doom", "Dragon", "Dream",
	"Drift", "Dusk", "Dust", "Eagle", "Echo", "Edge", "Elder", "Ember", "End",
	"Eye", "Fable", "Face", "Falcon", "Fang", "Fate", "Fear", "Feather", "Fern",
	"Fire", "Flame", "Flight", "Flower", "Fog", "Forest", "Forge", "Fox", "Frost",
	"Gate", "Ghost", "Gleam", "Gloom", "Glory", "Glyph", "Grace", "Grave", "Grove",
	"Guard", "Guide", "Hallow", "Hand", "Harbinger", "Harvest", "Haven", "Hawk",
	"Heart", "Herald", "Hollow", "Hood", "Hope", "Horn", "Hound", "Hunter", "Ice",
	"Judge", "Keeper", "Key", "King", "Knight", "Lake", "Lance", "Light", "Lion",
	"Lore", "Lotus", "Lynx", "Mage", "Maker", "Mantle", "Mark", "Marsh", "Mask",
	"Master", "Meadow", "Mind", "Mist", "Moon", "Mountain", "Mourner", "Myth",
	"Night", "Oath", "Oracle", "Owl", "Path", "Peak", "Phantom", "Phoenix", "Pilgrim",
	"Pyre", "Queen", "Quill", "Raven", "Reaper", "Realm", "Ridge", "Rift", "River",
	"Root", "Rose", "Rune", "Sage", "Sand", "Scale", "Scar", "Seeker", "Sentinel",
	"Shade", "Shadow", "Shard", "Shell", "Shield", "Shore", "Shrine", "Sign",
	"Singer", "Sky", "Slayer", "Snow", "Song", "Soul", "Spark", "Specter", "Spirit",
	"Staff", "Star", "Steel", "Stone", "Storm", "Stream", "Sun", "Sword", "Tale",
	"Talon", "Tempest", "Thorn", "Throne", "Thunder", "Tiger", "Tomb", "Tower",
	"Tracker", "Tree", "Vale", "Veil", "Venom", "Vigor", "Vision", "Voice", "Void",
	"Walker", "Warden", "Watcher", "Wave", "Weaver", "Well", "Whisper", "Wind",
	"Wing", "Winter", "Witness", "Wolf", "Wood", "Wrath", "Wraith", "Wyrm",
}
