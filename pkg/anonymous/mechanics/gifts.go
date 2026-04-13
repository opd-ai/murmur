// Package mechanics provides anonymous social interactions.
// Per ANONYMOUS_GAME_MECHANICS.md, mechanics include Phantom Gifts, Specter Marks,
// Territory Drift, and Cipher Puzzles.
package mechanics

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Gift tier constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// GiftTierBasic requires Resonance 25 (Shade milestone).
	GiftTierBasic = 25
	// GiftTierExpanded requires Resonance 50 (Wraith milestone).
	GiftTierExpanded = 50
	// GiftTierPremium requires Resonance 100 (Phantom milestone).
	GiftTierPremium = 100

	// GiftDuration is how long gifts remain visible (7 days).
	GiftDuration = 7 * 24 * time.Hour

	// MaxGiftsPerDay limits gifts per Specter per 24h.
	MaxGiftsPerDay = 3
)

// EffectType identifies a visual effect for Phantom Gifts.
type EffectType uint8

// Basic effects (Resonance 25+) per ANONYMOUS_GAME_MECHANICS.md.
const (
	EffectSoftGlowPulse EffectType = iota + 1
	EffectFaintHaloRing
	EffectGentleParticleDrift
	EffectShimmerOverlay
	EffectWarmthTintShift
)

// Expanded effects (Resonance 50+).
const (
	EffectOrbitingGeometric EffectType = iota + 10
	EffectAuroraColorShift
	EffectCrystallineFracture
	EffectEmberTrails
	EffectRippleDistortion
	EffectStarlightSparkle
	EffectVoidRipple
	EffectPrismShard
	EffectIceFormation
	EffectMistVeil
)

// Premium effects (Resonance 100+).
const (
	EffectMultiParticleSystem EffectType = iota + 30
	EffectFluidSimulation
	EffectGeometricMandala
	EffectVoidGravitation
	EffectPrismaticRefraction
	EffectNebulaeCloud
	EffectElectricArc
	EffectCrystalGrowth
	EffectPhoenixFlame
	EffectShadowWraith
)

// Errors for gift operations.
var (
	ErrInsufficientResonance = errors.New("insufficient resonance for this effect")
	ErrDailyLimitExceeded    = errors.New("daily gift limit exceeded")
	ErrGiftExpired           = errors.New("gift has expired")
	ErrInvalidSignature      = errors.New("invalid gift signature")
	ErrInvalidRecipient      = errors.New("invalid recipient")
	ErrDuplicateGift         = errors.New("duplicate gift")
)

// Gift represents a Phantom Gift from a Specter to any node.
// Per ANONYMOUS_GAME_MECHANICS.md, gifts are one-way gestures of
// generosity or recognition.
type Gift struct {
	ID           [32]byte   // Unique gift ID (BLAKE3 hash of content).
	SenderPubKey [32]byte   // Specter's Curve25519 public key.
	RecipientKey []byte     // Recipient's public key (Ed25519 or Curve25519).
	Effect       EffectType // Visual effect to apply.
	CreatedAt    time.Time  // When the gift was sent.
	ExpiresAt    time.Time  // When the gift fades (7 days).
	Signature    []byte     // Ed25519 signature for verification.
}

// IsExpired returns true if the gift has passed its expiration time.
func (g *Gift) IsExpired() bool {
	return time.Now().After(g.ExpiresAt)
}

// RequiredResonance returns the minimum Resonance needed for this effect.
func RequiredResonance(effect EffectType) int {
	switch {
	case effect >= EffectMultiParticleSystem:
		return GiftTierPremium
	case effect >= EffectOrbitingGeometric:
		return GiftTierExpanded
	default:
		return GiftTierBasic
	}
}

// GiftCatalog provides available effects based on Resonance level.
type GiftCatalog struct{}

// AvailableEffects returns effects available at a given Resonance level.
func (c *GiftCatalog) AvailableEffects(resonance int) []EffectType {
	var effects []EffectType

	// Basic effects (Resonance 25+).
	if resonance >= GiftTierBasic {
		effects = append(effects,
			EffectSoftGlowPulse,
			EffectFaintHaloRing,
			EffectGentleParticleDrift,
			EffectShimmerOverlay,
			EffectWarmthTintShift,
		)
	}

	// Expanded effects (Resonance 50+).
	if resonance >= GiftTierExpanded {
		effects = append(effects,
			EffectOrbitingGeometric,
			EffectAuroraColorShift,
			EffectCrystallineFracture,
			EffectEmberTrails,
			EffectRippleDistortion,
			EffectStarlightSparkle,
			EffectVoidRipple,
			EffectPrismShard,
			EffectIceFormation,
			EffectMistVeil,
		)
	}

	// Premium effects (Resonance 100+).
	if resonance >= GiftTierPremium {
		effects = append(effects,
			EffectMultiParticleSystem,
			EffectFluidSimulation,
			EffectGeometricMandala,
			EffectVoidGravitation,
			EffectPrismaticRefraction,
			EffectNebulaeCloud,
			EffectElectricArc,
			EffectCrystalGrowth,
			EffectPhoenixFlame,
			EffectShadowWraith,
		)
	}

	return effects
}

// GiftStore manages Phantom Gift storage and rate limiting.
type GiftStore struct {
	mu          sync.RWMutex
	gifts       map[[32]byte]*Gift     // By gift ID.
	byRecipient map[string][]*Gift     // By recipient key (hex).
	bySender    map[string][]*Gift     // By sender key (hex).
	dailyLimits map[string][]time.Time // Gift timestamps by sender (hex).
}

// NewGiftStore creates a new gift store.
func NewGiftStore() *GiftStore {
	return &GiftStore{
		gifts:       make(map[[32]byte]*Gift),
		byRecipient: make(map[string][]*Gift),
		bySender:    make(map[string][]*Gift),
		dailyLimits: make(map[string][]time.Time),
	}
}

// keyToHex converts a public key to a hex string for map keys.
func keyToHex(key []byte) string {
	if len(key) == 0 {
		return ""
	}
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(key)*2)
	for i, v := range key {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

// CanSendGift checks if a Specter can send a gift based on rate limits.
func (s *GiftStore) CanSendGift(senderKey [32]byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hex := keyToHex(senderKey[:])
	timestamps := s.dailyLimits[hex]

	// Count gifts in the last 24 hours.
	cutoff := time.Now().Add(-24 * time.Hour)
	count := 0
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			count++
		}
	}

	return count < MaxGiftsPerDay
}

// CreateGift creates a new Phantom Gift.
// The signingKey is used to sign the gift for verification.
func (s *GiftStore) CreateGift(
	senderKey [32]byte,
	recipientKey []byte,
	effect EffectType,
	resonance int,
	signingKey ed25519.PrivateKey,
) (*Gift, error) {
	// Check resonance requirement.
	required := RequiredResonance(effect)
	if resonance < required {
		return nil, ErrInsufficientResonance
	}

	// Check daily limit.
	if !s.CanSendGift(senderKey) {
		return nil, ErrDailyLimitExceeded
	}

	// Validate recipient.
	if len(recipientKey) == 0 {
		return nil, ErrInvalidRecipient
	}

	now := time.Now()
	gift := &Gift{
		SenderPubKey: senderKey,
		RecipientKey: recipientKey,
		Effect:       effect,
		CreatedAt:    now,
		ExpiresAt:    now.Add(GiftDuration),
	}

	// Generate gift ID as BLAKE3 hash of content.
	h := blake3.New()
	h.Write(senderKey[:])
	h.Write(recipientKey)
	h.Write([]byte{byte(effect)})
	var timestamp [8]byte
	now.UnmarshalBinary(timestamp[:])
	h.Write(timestamp[:])
	copy(gift.ID[:], h.Sum(nil))

	// Sign the gift.
	if signingKey != nil {
		signData := append(gift.ID[:], gift.RecipientKey...)
		signData = append(signData, byte(effect))
		gift.Signature = ed25519.Sign(signingKey, signData)
	}

	// Store the gift.
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gifts[gift.ID] = gift

	senderHex := keyToHex(senderKey[:])
	s.bySender[senderHex] = append(s.bySender[senderHex], gift)
	s.dailyLimits[senderHex] = append(s.dailyLimits[senderHex], now)

	recipientHex := keyToHex(recipientKey)
	s.byRecipient[recipientHex] = append(s.byRecipient[recipientHex], gift)

	return gift, nil
}

// GetGift retrieves a gift by ID.
func (s *GiftStore) GetGift(id [32]byte) (*Gift, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gift, ok := s.gifts[id]
	if !ok {
		return nil, nil
	}

	if gift.IsExpired() {
		return nil, ErrGiftExpired
	}

	return gift, nil
}

// GetGiftsForRecipient returns all active gifts for a recipient.
func (s *GiftStore) GetGiftsForRecipient(recipientKey []byte) []*Gift {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hex := keyToHex(recipientKey)
	all := s.byRecipient[hex]

	var active []*Gift
	for _, g := range all {
		if !g.IsExpired() {
			active = append(active, g)
		}
	}

	return active
}

// GetGiftsBySender returns all active gifts sent by a Specter.
func (s *GiftStore) GetGiftsBySender(senderKey [32]byte) []*Gift {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hex := keyToHex(senderKey[:])
	all := s.bySender[hex]

	var active []*Gift
	for _, g := range all {
		if !g.IsExpired() {
			active = append(active, g)
		}
	}

	return active
}

// VerifyGift verifies a gift's signature.
func VerifyGift(gift *Gift, publicKey ed25519.PublicKey) bool {
	if gift == nil || len(gift.Signature) == 0 {
		return false
	}

	signData := append(gift.ID[:], gift.RecipientKey...)
	signData = append(signData, byte(gift.Effect))

	return ed25519.Verify(publicKey, signData, gift.Signature)
}

// GarbageCollect removes expired gifts.
func (s *GiftStore) GarbageCollect() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := s.removeExpiredGifts()
	s.cleanDailyLimits()
	s.rebuildIndexes()

	return removed
}

// removeExpiredGifts deletes expired gifts and returns count removed.
func (s *GiftStore) removeExpiredGifts() int {
	removed := 0
	for id, gift := range s.gifts {
		if gift.IsExpired() {
			delete(s.gifts, id)
			removed++
		}
	}
	return removed
}

// cleanDailyLimits removes rate limit entries older than 24 hours.
func (s *GiftStore) cleanDailyLimits() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for sender, timestamps := range s.dailyLimits {
		recent := filterRecentTimestamps(timestamps, cutoff)
		if len(recent) == 0 {
			delete(s.dailyLimits, sender)
		} else {
			s.dailyLimits[sender] = recent
		}
	}
}

// filterRecentTimestamps returns timestamps after the cutoff.
func filterRecentTimestamps(timestamps []time.Time, cutoff time.Time) []time.Time {
	var recent []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}
	return recent
}

// rebuildIndexes rebuilds recipient and sender indexes.
func (s *GiftStore) rebuildIndexes() {
	s.byRecipient = make(map[string][]*Gift)
	s.bySender = make(map[string][]*Gift)

	for _, gift := range s.gifts {
		senderHex := keyToHex(gift.SenderPubKey[:])
		s.bySender[senderHex] = append(s.bySender[senderHex], gift)

		recipientHex := keyToHex(gift.RecipientKey)
		s.byRecipient[recipientHex] = append(s.byRecipient[recipientHex], gift)
	}
}

// Count returns the number of active (non-expired) gifts.
func (s *GiftStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, gift := range s.gifts {
		if !gift.IsExpired() {
			count++
		}
	}
	return count
}

// GenerateGiftID creates a unique gift ID from random bytes.
func GenerateGiftID() ([32]byte, error) {
	var id [32]byte
	_, err := rand.Read(id[:])
	return id, err
}

// EffectName returns the human-readable name of an effect.
func EffectName(effect EffectType) string {
	switch effect {
	// Basic effects.
	case EffectSoftGlowPulse:
		return "Soft Glow Pulse"
	case EffectFaintHaloRing:
		return "Faint Halo Ring"
	case EffectGentleParticleDrift:
		return "Gentle Particle Drift"
	case EffectShimmerOverlay:
		return "Shimmer Overlay"
	case EffectWarmthTintShift:
		return "Warmth Tint Shift"
	// Expanded effects.
	case EffectOrbitingGeometric:
		return "Orbiting Geometric"
	case EffectAuroraColorShift:
		return "Aurora Color Shift"
	case EffectCrystallineFracture:
		return "Crystalline Fracture"
	case EffectEmberTrails:
		return "Ember Trails"
	case EffectRippleDistortion:
		return "Ripple Distortion"
	case EffectStarlightSparkle:
		return "Starlight Sparkle"
	case EffectVoidRipple:
		return "Void Ripple"
	case EffectPrismShard:
		return "Prism Shard"
	case EffectIceFormation:
		return "Ice Formation"
	case EffectMistVeil:
		return "Mist Veil"
	// Premium effects.
	case EffectMultiParticleSystem:
		return "Multi-Particle System"
	case EffectFluidSimulation:
		return "Fluid Simulation"
	case EffectGeometricMandala:
		return "Geometric Mandala"
	case EffectVoidGravitation:
		return "Void Gravitation"
	case EffectPrismaticRefraction:
		return "Prismatic Refraction"
	case EffectNebulaeCloud:
		return "Nebulae Cloud"
	case EffectElectricArc:
		return "Electric Arc"
	case EffectCrystalGrowth:
		return "Crystal Growth"
	case EffectPhoenixFlame:
		return "Phoenix Flame"
	case EffectShadowWraith:
		return "Shadow Wraith"
	default:
		return "Unknown Effect"
	}
}

// EffectDescription returns a description of the visual effect.
func EffectDescription(effect EffectType) string {
	switch effect {
	case EffectSoftGlowPulse:
		return "A subtle pulsing glow around the node"
	case EffectFaintHaloRing:
		return "A faint ring of light encircling the node"
	case EffectGentleParticleDrift:
		return "Small particles drifting gently outward"
	case EffectShimmerOverlay:
		return "A shimmering overlay effect"
	case EffectWarmthTintShift:
		return "A warm color tint shift"
	case EffectOrbitingGeometric:
		return "Geometric shapes orbiting the node"
	case EffectAuroraColorShift:
		return "Aurora-like color shifting effect"
	case EffectCrystallineFracture:
		return "Crystalline fracture patterns"
	case EffectEmberTrails:
		return "Glowing ember trails"
	case EffectRippleDistortion:
		return "Rippling distortion field"
	default:
		return "Visual effect applied to the node"
	}
}

// CompareGiftIDs compares two gift IDs for equality.
func CompareGiftIDs(a, b [32]byte) bool {
	return bytes.Equal(a[:], b[:])
}
