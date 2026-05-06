// Package storage provides local Wave caching and garbage collection.
// Per TECHNICAL_IMPLEMENTATION.md §1.5, Waves are stored in Bbolt
// with TTL metadata for expiration.
package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// GCInterval is the interval between garbage collection runs.
// Per TECHNICAL_IMPLEMENTATION.md, GC runs hourly.
const GCInterval = time.Hour

// GCTargetTime is the target maximum duration for garbage collection.
// Per TECHNICAL_IMPLEMENTATION.md, GC should complete in <100ms.
const GCTargetTime = 100 * time.Millisecond

// DefaultCacheSize is the default maximum number of Waves to cache.
const DefaultCacheSize = 10000

// MaxContentWindow is the maximum age of content (30 days).
// Per WAVES.md, Waves older than 30 days are garbage collected.
const MaxContentWindow = 30 * 24 * time.Hour

// Errors for storage operations.
var (
	ErrNotFound    = errors.New("wave not found")
	ErrCacheFull   = errors.New("cache is full")
	ErrStoreClosed = errors.New("store is closed")
	ErrInvalidWave = errors.New("invalid wave data")
	ErrNilStore    = errors.New("store is nil")
)

type waveWithTime struct {
	id   string
	time int64
}

// Cache provides in-memory and persistent Wave storage with TTL enforcement.
type Cache struct {
	mu      sync.RWMutex
	db      *store.DB
	memory  map[string]*pb.Wave // wave ID -> Wave
	maxSize int
	closed  bool

	// Rate tracking for adaptive difficulty adjustment.
	// Per AUDIT.md HIGH finding "PoW difficulty not dynamically adjusted".
	rateWindow     []time.Time // timestamps of recent Wave arrivals (5-minute sliding window)
	lastRateCheck  time.Time   // last time rate was evaluated
	lastAdjustment time.Time   // last time difficulty was adjusted
}

// CacheConfig configures the Wave cache.
type CacheConfig struct {
	MaxSize int
}

// NewCache creates a new Wave cache with the given database.
func NewCache(db *store.DB) (*Cache, error) {
	if db == nil {
		return nil, ErrNilStore
	}

	return &Cache{
		db:             db,
		memory:         make(map[string]*pb.Wave),
		maxSize:        DefaultCacheSize,
		rateWindow:     make([]time.Time, 0, 100),
		lastRateCheck:  time.Now(),
		lastAdjustment: time.Now(),
	}, nil
}

// NewCacheWithConfig creates a cache with custom configuration.
func NewCacheWithConfig(db *store.DB, cfg CacheConfig) (*Cache, error) {
	if db == nil {
		return nil, ErrNilStore
	}

	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = DefaultCacheSize
	}

	return &Cache{
		db:             db,
		memory:         make(map[string]*pb.Wave),
		maxSize:        maxSize,
		rateWindow:     make([]time.Time, 0, 100),
		lastRateCheck:  time.Now(),
		lastAdjustment: time.Now(),
	}, nil
}

// Put stores a Wave in the cache and database.
func (c *Cache) Put(wave *pb.Wave) error {
	if err := validateWave(wave); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrStoreClosed
	}

	if err := c.ensureCapacityLocked(); err != nil {
		return err
	}

	// Track arrival time for rate-based difficulty adjustment.
	c.trackArrivalLocked(time.Now())

	c.memory[string(wave.WaveId)] = wave
	return c.persistWave(wave)
}

// validateWave checks if the wave is valid for storage.
func validateWave(wave *pb.Wave) error {
	if wave == nil || len(wave.WaveId) == 0 {
		return ErrInvalidWave
	}
	return nil
}

// ensureCapacityLocked ensures cache has room for a new wave. Must hold c.mu.
func (c *Cache) ensureCapacityLocked() error {
	if len(c.memory) < c.maxSize {
		return nil
	}

	c.evictExpiredLocked()
	if len(c.memory) >= c.maxSize {
		return ErrCacheFull
	}
	return nil
}

// persistWave serializes and stores the wave in the database.
func (c *Cache) persistWave(wave *pb.Wave) error {
	data, err := proto.Marshal(wave)
	if err != nil {
		return err
	}
	return c.db.Put(store.BucketWaves, wave.WaveId, data)
}

// Get retrieves a Wave by ID from cache or database.
func (c *Cache) Get(waveID []byte) (*pb.Wave, error) {
	wave, found, err := c.getFromMemory(waveID)
	if err != nil {
		return nil, err
	}
	if found {
		return wave, nil
	}
	return c.getFromDatabase(waveID)
}

// getFromMemory checks the memory cache for a Wave.
func (c *Cache) getFromMemory(waveID []byte) (*pb.Wave, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, false, ErrStoreClosed
	}

	wave, ok := c.memory[string(waveID)]
	return wave, ok, nil
}

// getFromDatabase retrieves a Wave from the database and caches it.
func (c *Cache) getFromDatabase(waveID []byte) (*pb.Wave, error) {
	data, err := c.db.Get(store.BucketWaves, waveID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, ErrNotFound
	}

	wave := &pb.Wave{}
	if err := proto.Unmarshal(data, wave); err != nil {
		return nil, err
	}

	c.cacheWave(waveID, wave)
	return wave, nil
}

// cacheWave adds a Wave to the memory cache if space is available.
func (c *Cache) cacheWave(waveID []byte, wave *pb.Wave) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed && len(c.memory) < c.maxSize {
		c.memory[string(waveID)] = wave
	}
}

// Delete removes a Wave from cache and database.
func (c *Cache) Delete(waveID []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrStoreClosed
	}

	delete(c.memory, string(waveID))
	return c.db.Delete(store.BucketWaves, waveID)
}

// Has checks if a Wave exists in cache or database.
func (c *Cache) Has(waveID []byte) bool {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return false
	}

	if _, ok := c.memory[string(waveID)]; ok {
		c.mu.RUnlock()
		return true
	}
	c.mu.RUnlock()

	data, err := c.db.Get(store.BucketWaves, waveID)
	return err == nil && len(data) > 0
}

// Size returns the number of Waves in the memory cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.memory)
}

// evictExpiredLocked removes expired waves from memory cache.
// Must be called with c.mu held.
func (c *Cache) evictExpiredLocked() int {
	count := 0
	for id, wave := range c.memory {
		if waves.IsExpired(wave) {
			delete(c.memory, id)
			count++
		}
	}
	return count
}

// GarbageCollect removes expired Waves from cache and database.
// Per TECHNICAL_IMPLEMENTATION.md, this should complete in <100ms.
func (c *Cache) GarbageCollect() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, ErrStoreClosed
	}

	// Phase 1: Collect expired IDs from memory cache.
	memoryExpired := c.collectExpiredIDs()
	memoryExpiredSet := make(map[string]struct{}, len(memoryExpired))
	for _, id := range memoryExpired {
		memoryExpiredSet[string(id)] = struct{}{}
	}

	// Phase 2: Scan database for expired waves not already found in memory.
	dbExpired, err := c.collectExpiredFromDatabase(memoryExpiredSet)
	if err != nil {
		// Still clean up memory even if database scan fails.
		c.removeFromMemory(memoryExpired)
		c.removeFromDatabase(memoryExpired)
		return len(memoryExpired), err
	}

	// Phase 3: Remove all expired waves from memory and database.
	c.removeFromMemory(memoryExpired)
	allExpired := append(memoryExpired, dbExpired...)
	c.removeFromDatabase(allExpired)

	return len(allExpired), nil
}

// collectExpiredFromDatabase scans the database for expired waves.
// Returns IDs of waves that have expired and are not in skipSet.
func (c *Cache) collectExpiredFromDatabase(skipSet map[string]struct{}) ([][]byte, error) {
	var expiredIDs [][]byte

	err := c.db.ForEach(store.BucketWaves, func(key, value []byte) error {
		// Skip if already found in memory scan.
		if _, ok := skipSet[string(key)]; ok {
			return nil
		}

		wave := &pb.Wave{}
		if err := proto.Unmarshal(value, wave); err != nil {
			// Skip malformed entries.
			return nil
		}

		if waves.IsExpired(wave) {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			expiredIDs = append(expiredIDs, keyCopy)
		}
		return nil
	})

	return expiredIDs, err
}

// collectExpiredIDs returns IDs of all expired waves in memory.
func (c *Cache) collectExpiredIDs() [][]byte {
	var expiredIDs [][]byte
	for id, wave := range c.memory {
		if waves.IsExpired(wave) {
			expiredIDs = append(expiredIDs, []byte(id))
		}
	}
	return expiredIDs
}

// removeFromMemory deletes expired waves from the memory cache.
func (c *Cache) removeFromMemory(expiredIDs [][]byte) {
	for _, id := range expiredIDs {
		delete(c.memory, string(id))
	}
}

// removeFromDatabase deletes expired waves from persistent storage.
func (c *Cache) removeFromDatabase(expiredIDs [][]byte) {
	for _, id := range expiredIDs {
		c.db.Delete(store.BucketWaves, id)
	}
}

// EvictOldest evicts up to count oldest Waves from memory cache.
// Per AUDIT.md HIGH "No memory budget enforcement", this is called
// during memory pressure to free space. Returns number of Waves evicted.
func (c *Cache) EvictOldest(count int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || count <= 0 {
		return 0
	}

	waves := c.collectWavesWithTime()
	sortWavesByTime(waves)
	return c.evictWaves(waves, count)
}

func (c *Cache) collectWavesWithTime() []waveWithTime {
	waves := make([]waveWithTime, 0, len(c.memory))
	for id, wave := range c.memory {
		waves = append(waves, waveWithTime{
			id:   id,
			time: wave.CreatedAt,
		})
	}
	return waves
}

func sortWavesByTime(waves []waveWithTime) {
	for i := 0; i < len(waves); i++ {
		for j := i + 1; j < len(waves); j++ {
			if waves[i].time > waves[j].time {
				waves[i], waves[j] = waves[j], waves[i]
			}
		}
	}
}

func (c *Cache) evictWaves(waves []waveWithTime, count int) int {
	evicted := 0
	for i := 0; i < len(waves) && i < count; i++ {
		delete(c.memory, waves[i].id)
		evicted++
	}
	return evicted
}

// StartGC runs periodic garbage collection.
// Returns a cancel function to stop the GC goroutine.
// Per ROADMAP.md line 836, monitors GC sweep duration (<100ms target).
func (c *Cache) StartGC(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.performGCCycle()
			}
		}
	}()

	return cancel
}

// performGCCycle executes one garbage collection cycle and reports performance.
func (c *Cache) performGCCycle() {
	start := time.Now()
	count, err := c.GarbageCollect()
	duration := time.Since(start)

	c.reportGCPerformance(duration, count, err)
}

// reportGCPerformance logs GC performance if it exceeds target or encounters errors.
func (c *Cache) reportGCPerformance(duration time.Duration, count int, err error) {
	if duration > GCTargetTime {
		durationMs := duration.Milliseconds()
		println("WARNING: GC sweep took", durationMs, "ms (target <100ms), collected", count, "waves")
	}
	if err != nil {
		println("GC sweep error:", err.Error())
	}
}

// List returns up to limit Waves from the cache, sorted by timestamp (newest first).
func (c *Cache) List(limit int) ([]*pb.Wave, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrStoreClosed
	}

	waveList := c.collectNonExpiredWaves()
	sortWavesByNewest(waveList)

	if len(waveList) > limit {
		waveList = waveList[:limit]
	}

	return waveList, nil
}

// collectNonExpiredWaves gathers all non-expired waves from memory.
func (c *Cache) collectNonExpiredWaves() []*pb.Wave {
	waveList := make([]*pb.Wave, 0, len(c.memory))
	for _, wave := range c.memory {
		if !waves.IsExpired(wave) {
			waveList = append(waveList, wave)
		}
	}
	return waveList
}

// sortWavesByNewest sorts waves by timestamp descending (newest first).
func sortWavesByNewest(waveList []*pb.Wave) {
	for i := 0; i < len(waveList); i++ {
		for j := i + 1; j < len(waveList); j++ {
			if waveList[i].CreatedAt < waveList[j].CreatedAt {
				waveList[i], waveList[j] = waveList[j], waveList[i]
			}
		}
	}
}

// Close closes the cache.
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	c.memory = nil
	return nil
}

// trackArrivalLocked records a Wave arrival timestamp and checks for difficulty adjustment.
// Must be called with c.mu held.
// Per AUDIT.md HIGH finding: "PoW difficulty not dynamically adjusted".
func (c *Cache) trackArrivalLocked(arrivalTime time.Time) {
	// Add current arrival to window.
	c.rateWindow = append(c.rateWindow, arrivalTime)

	// Check rate every 30 seconds to avoid constant recalculation.
	timeSinceLastCheck := arrivalTime.Sub(c.lastRateCheck)
	if timeSinceLastCheck < 30*time.Second {
		return
	}
	c.lastRateCheck = arrivalTime

	// Prune window to last 5 minutes.
	windowStart := arrivalTime.Add(-5 * time.Minute)
	validIdx := 0
	for i, t := range c.rateWindow {
		if t.After(windowStart) {
			validIdx = i
			break
		}
	}
	c.rateWindow = c.rateWindow[validIdx:]

	// Calculate rate (Waves per minute).
	if len(c.rateWindow) == 0 {
		return
	}
	duration := arrivalTime.Sub(c.rateWindow[0]).Minutes()
	if duration < 0.1 { // avoid division by near-zero
		return
	}
	rate := float64(len(c.rateWindow)) / duration

	// Adjust difficulty if needed (but not more often than every 5 minutes).
	timeSinceLastAdjustment := arrivalTime.Sub(c.lastAdjustment)
	if timeSinceLastAdjustment < 5*time.Minute {
		return
	}

	c.adjustDifficultyLocked(rate, arrivalTime)
}

// adjustDifficultyLocked modifies PoW difficulty based on incoming Wave rate.
// Per AUDIT.md remediation: "If rate exceeds 100 Waves/min, increment difficulty by 1 bit.
// If rate drops below 20 Waves/min for 10 minutes, decrement by 1 (min 16)."
// Must be called with c.mu held.
func (c *Cache) adjustDifficultyLocked(ratePerMinute float64, currentTime time.Time) {
	cfg := pow.GetGlobalConfig()
	currentDifficulty := cfg.GetStandard()

	newDifficulty, adjusted := c.computeNewDifficulty(ratePerMinute, currentTime, currentDifficulty, cfg)

	if adjusted && cfg.SetStandard(newDifficulty) {
		c.lastAdjustment = currentTime
		c.persistDifficulty(newDifficulty)
	}
}

// computeNewDifficulty calculates the new difficulty level based on rate.
func (c *Cache) computeNewDifficulty(ratePerMinute float64, currentTime time.Time, currentDifficulty uint8, cfg *pow.DifficultyConfig) (uint8, bool) {
	if ratePerMinute > 100 {
		return c.increaseDifficulty(currentDifficulty, cfg)
	}
	if ratePerMinute < 20 {
		return c.decreaseDifficulty(currentTime, currentDifficulty, cfg)
	}
	return currentDifficulty, false
}

// increaseDifficulty increments difficulty to throttle high rate.
func (c *Cache) increaseDifficulty(currentDifficulty uint8, cfg *pow.DifficultyConfig) (uint8, bool) {
	newDifficulty := currentDifficulty + 1
	if newDifficulty > cfg.GetMaxAcceptable() {
		newDifficulty = cfg.GetMaxAcceptable()
	}
	return newDifficulty, newDifficulty != currentDifficulty
}

// decreaseDifficulty decrements difficulty if low rate sustained for 10+ minutes.
func (c *Cache) decreaseDifficulty(currentTime time.Time, currentDifficulty uint8, cfg *pow.DifficultyConfig) (uint8, bool) {
	timeSinceLastAdjustment := currentTime.Sub(c.lastAdjustment)
	if timeSinceLastAdjustment > 10*time.Minute && currentDifficulty > cfg.GetMinAcceptable() {
		return currentDifficulty - 1, true
	}
	return currentDifficulty, false
}

// persistDifficulty stores the current difficulty in the config bucket for restart persistence.
func (c *Cache) persistDifficulty(difficulty uint8) {
	key := []byte("pow_difficulty_standard")
	value := []byte{difficulty}
	// Ignore errors; difficulty will reset to default on restart if persistence fails.
	c.db.Put(store.BucketConfig, key, value)
}

// LoadPersistedDifficulty restores difficulty from the config bucket on startup.
// Returns the persisted difficulty, or 0 if not found.
func (c *Cache) LoadPersistedDifficulty() uint8 {
	key := []byte("pow_difficulty_standard")
	value, err := c.db.Get(store.BucketConfig, key)
	if err != nil || len(value) == 0 {
		return 0
	}
	return value[0]
}

// waveExpirationKey generates a key for expiration index.
func waveExpirationKey(expiresAt time.Time, waveID []byte) []byte {
	key := make([]byte, 8+len(waveID))
	binary.BigEndian.PutUint64(key[:8], uint64(expiresAt.Unix()))
	copy(key[8:], waveID)
	return key
}
