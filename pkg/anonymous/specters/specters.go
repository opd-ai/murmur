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
	ErrNilKeyPair     = errors.New("nil keypair")
	ErrSuspended      = errors.New("specter is suspended")
	ErrDeleted        = errors.New("specter is deleted")
	ErrInvalidStatus  = errors.New("invalid status")
	ErrEncryptionFail = errors.New("encryption failed")
	ErrDecryptionFail = errors.New("decryption failed")
)

// Specter represents an anonymous identity in the Anonymous Layer.
// Per DESIGN_DOCUMENT.md, Specters use independent Curve25519 keypairs
// with no derivation relationship to Surface identities.
type Specter struct {
	mu         sync.RWMutex
	PrivateKey [32]byte
	PublicKey  [32]byte
	Name       string
	CreatedAt  time.Time
	Status     string
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
	}

	// Generate procedural name from public key.
	s.Name = GenerateName(s.PublicKey[:])

	return s, nil
}

// NewSpecterFromKeyPair creates a Specter from an existing keypair.
func NewSpecterFromKeyPair(kp *KeyPair) (*Specter, error) {
	if kp == nil {
		return nil, ErrNilKeyPair
	}

	s := &Specter{
		PrivateKey: kp.Private,
		PublicKey:  kp.Public,
		CreatedAt:  time.Now(),
		Status:     StatusActive,
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

// Encrypt encrypts data using XChaCha20-Poly1305 with a derived key.
func (s *Specter) Encrypt(plaintext, recipientPublic []byte) ([]byte, error) {
	shared, err := s.DeriveSharedSecret(recipientPublic)
	if err != nil {
		return nil, err
	}

	// Derive encryption key from shared secret.
	h := blake3.New()
	h.Write(shared)
	h.Write([]byte("murmur-specter-encrypt"))
	key := h.Sum(nil)[:32]

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

	// Derive encryption key from shared secret.
	h := blake3.New()
	h.Write(shared)
	h.Write([]byte("murmur-specter-encrypt"))
	key := h.Sum(nil)[:32]

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
