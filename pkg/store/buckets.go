package store

// Bucket names per TECHNICAL_IMPLEMENTATION.md §1.5.
// These are platform-agnostic and used by both bbolt-based and browser stores.
var (
	BucketIdentity  = []byte("identity")
	BucketPeers     = []byte("peers")
	BucketWaves     = []byte("waves")
	BucketThreads   = []byte("threads")
	BucketShroud    = []byte("shroud")
	BucketResonance = []byte("resonance")
	BucketConfig    = []byte("config")

	// Continuity chain storage per docs/KEY_ROTATION.md §Continuity Chain Management.
	BucketContinuityChains = []byte("continuity_chains")
	BucketDevices          = []byte("devices")
	BucketMaskedEvents     = []byte("masked_events")
	BucketDailyLimits      = []byte("daily_limits")

	// Mechanics buckets for Anonymous Layer game state persistence.
	BucketCouncils    = []byte("councils")
	BucketShadowPlay  = []byte("shadowplay")
	BucketForge       = []byte("forge")
	BucketOracles     = []byte("oracles")
	BucketTerritories = []byte("territories")
	BucketHunts       = []byte("hunts")
	BucketPuzzles     = []byte("puzzles")
	BucketMarks       = []byte("marks")
	BucketGifts       = []byte("gifts")
)
