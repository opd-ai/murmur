// Package mechanics - Cipher Puzzles implementation.
// Per ANONYMOUS_GAME_MECHANICS.md, puzzles are collaborative/competitive
// cryptographic challenges that Specters solve together or against each other.
package puzzles

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Puzzle constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// PuzzleMinResonance requires Resonance 50 (Wraith milestone) to initiate.
	PuzzleMinResonance = 50

	// Puzzle durations.
	PuzzleDuration15Min = 15 * time.Minute
	PuzzleDuration30Min = 30 * time.Minute
	PuzzleDuration60Min = 60 * time.Minute

	// Default difficulty (leading zero bits required).
	DefaultPuzzleDifficulty = 20

	// Resonance bonus decay period.
	PuzzleBonusDecayDays = 14
)

// PuzzleType identifies the type of puzzle.
type PuzzleType uint8

// Puzzle types per ANONYMOUS_GAME_MECHANICS.md.
const (
	PuzzleFragment PuzzleType = iota + 1 // Competitive - first solver wins.
	PuzzleMosaic                         // Collaborative - multiple contributors.
	PuzzleCascade                        // Sequential - chain of puzzles.
)

// PuzzleState represents the current state of a puzzle.
type PuzzleState uint8

const (
	PuzzlePending PuzzleState = iota // Not yet started.
	PuzzleActive                     // Accepting solutions.
	PuzzleSolved                     // Solution found.
	PuzzleExpired                    // Time limit reached.
)

// Errors for puzzle operations.
var (
	ErrPuzzleNotFound        = errors.New("puzzle not found")
	ErrPuzzleExpired         = errors.New("puzzle has expired")
	ErrPuzzleAlreadySolved   = errors.New("puzzle already solved")
	ErrInvalidSolution       = errors.New("invalid solution")
	ErrInvalidPuzzleType     = errors.New("invalid puzzle type")
	ErrInvalidPuzzleDuration = errors.New("invalid puzzle duration")
	ErrPuzzleInsufficientRes = errors.New("insufficient resonance to initiate puzzle")
)

// Puzzle represents a Cipher Puzzle challenge.
type Puzzle struct {
	ID           [32]byte      // Unique puzzle ID.
	Type         PuzzleType    // Fragment, Mosaic, or Cascade.
	Seed         [32]byte      // Seed for deterministic generation.
	Difficulty   uint8         // Number of leading zero bits required.
	CreatedAt    time.Time     // When the puzzle was announced.
	Duration     time.Duration // Time limit.
	ExpiresAt    time.Time     // When the puzzle expires.
	State        PuzzleState   // Current state.
	InitiatorKey [32]byte      // Specter who initiated the puzzle.

	// For Fragment puzzles (single winner).
	WinnerKey *[32]byte  // Winner's Specter key.
	Solution  []byte     // Winning solution.
	SolvedAt  *time.Time // When solved.

	// For Mosaic puzzles (multiple contributors).
	Fragments     int            // Number of sub-problems.
	Contributions []Contribution // Submitted solutions.

	// For Cascade puzzles (sequential).
	Stages         int        // Number of stages.
	CurrentStage   int        // Current stage (0-indexed).
	StageSolutions [][]byte   // Solutions per stage.
	StageSolvers   [][32]byte // Solver per stage.

	mu sync.RWMutex
}

// Contribution represents a puzzle contribution (for Mosaic puzzles).
type Contribution struct {
	SolverKey     [32]byte  // Contributor's Specter key.
	FragmentIndex int       // Which fragment was solved.
	Solution      []byte    // The sub-solution.
	SubmittedAt   time.Time // When submitted.
}

// PuzzleGenerator defines the interface for puzzle generation and verification.
type PuzzleGenerator interface {
	Generate(seed []byte, epoch uint64) *Puzzle
	Verify(puzzle *Puzzle, solution []byte) bool
}

// FragmentGenerator generates Fragment-type puzzles.
type FragmentGenerator struct {
	Difficulty uint8
}

// Generate creates a new Fragment puzzle from a seed.
func (g *FragmentGenerator) Generate(seed []byte, epoch uint64) *Puzzle {
	var puzzleID [32]byte
	h := blake3.New()
	h.Write(seed)
	binary.Write(h, binary.BigEndian, epoch)
	copy(puzzleID[:], h.Sum(nil))

	var seedArr [32]byte
	copy(seedArr[:], seed)

	difficulty := g.Difficulty
	if difficulty == 0 {
		difficulty = DefaultPuzzleDifficulty
	}

	return &Puzzle{
		ID:         puzzleID,
		Type:       PuzzleFragment,
		Seed:       seedArr,
		Difficulty: difficulty,
		State:      PuzzlePending,
	}
}

// Verify checks if a solution is valid for a Fragment puzzle.
// Per ANONYMOUS_GAME_MECHANICS.md: Find nonce such that
// SHA-256(puzzle_seed || nonce) starts with N zero bits.
func (g *FragmentGenerator) Verify(puzzle *Puzzle, solution []byte) bool {
	if puzzle == nil || len(solution) == 0 {
		return false
	}

	// Compute SHA-256(seed || nonce).
	h := sha256.New()
	h.Write(puzzle.Seed[:])
	h.Write(solution)
	hash := h.Sum(nil)

	// Check for required leading zero bits.
	return hasLeadingZeros(hash, int(puzzle.Difficulty))
}

// hasLeadingZeros checks if a hash has at least n leading zero bits.
func hasLeadingZeros(hash []byte, n int) bool {
	fullBytes := n / 8
	remainingBits := n % 8

	// Check full zero bytes.
	for i := 0; i < fullBytes && i < len(hash); i++ {
		if hash[i] != 0 {
			return false
		}
	}

	// Check remaining bits.
	if remainingBits > 0 && fullBytes < len(hash) {
		mask := byte(0xFF) << (8 - remainingBits)
		if hash[fullBytes]&mask != 0 {
			return false
		}
	}

	return true
}

// NewPuzzle creates a new puzzle with the given parameters.
func NewPuzzle(
	puzzleType PuzzleType,
	seed [32]byte,
	difficulty uint8,
	duration time.Duration,
	initiatorKey [32]byte,
) (*Puzzle, error) {
	if err := validatePuzzleParams(puzzleType, duration); err != nil {
		return nil, err
	}

	difficulty = normalizeDifficulty(difficulty)
	now := time.Now()

	puzzle := &Puzzle{
		ID:           computePuzzleID(seed, initiatorKey, now),
		Type:         puzzleType,
		Seed:         seed,
		Difficulty:   difficulty,
		CreatedAt:    now,
		Duration:     duration,
		ExpiresAt:    now.Add(duration),
		State:        PuzzleActive,
		InitiatorKey: initiatorKey,
	}

	initPuzzleTypeFields(puzzle, puzzleType)
	return puzzle, nil
}

// NewPuzzleGated creates a puzzle with Resonance gating enforcement.
// Per ROADMAP.md line 414, only Specters with Resonance >= 50 can create puzzles.
func NewPuzzleGated(
	puzzleType PuzzleType,
	seed [32]byte,
	difficulty uint8,
	duration time.Duration,
	initiatorKey [32]byte,
	gate ResonanceGate,
) (*Puzzle, error) {
	if err := CheckResonanceGate(gate, initiatorKey, PuzzleMinResonance); err != nil {
		return nil, ErrPuzzleInsufficientRes
	}
	return NewPuzzle(puzzleType, seed, difficulty, duration, initiatorKey)
}

// validatePuzzleParams validates puzzle type and duration.
func validatePuzzleParams(puzzleType PuzzleType, duration time.Duration) error {
	if puzzleType < PuzzleFragment || puzzleType > PuzzleCascade {
		return ErrInvalidPuzzleType
	}
	if !isValidDuration(duration) {
		return ErrInvalidPuzzleDuration
	}
	return nil
}

// isValidDuration checks if duration is one of the allowed values.
func isValidDuration(duration time.Duration) bool {
	return duration == PuzzleDuration15Min ||
		duration == PuzzleDuration30Min ||
		duration == PuzzleDuration60Min
}

// normalizeDifficulty returns default difficulty if zero.
func normalizeDifficulty(difficulty uint8) uint8 {
	if difficulty == 0 {
		return DefaultPuzzleDifficulty
	}
	return difficulty
}

// computePuzzleID generates a BLAKE3 hash for the puzzle.
func computePuzzleID(seed, initiatorKey [32]byte, createdAt time.Time) [32]byte {
	h := blake3.New()
	h.Write(seed[:])
	h.Write(initiatorKey[:])
	binary.Write(h, binary.BigEndian, createdAt.Unix())
	var id [32]byte
	copy(id[:], h.Sum(nil))
	return id
}

// initPuzzleTypeFields initializes type-specific puzzle fields.
func initPuzzleTypeFields(puzzle *Puzzle, puzzleType PuzzleType) {
	switch puzzleType {
	case PuzzleMosaic:
		puzzle.Fragments = 5
	case PuzzleCascade:
		puzzle.Stages = 3
		puzzle.CurrentStage = 0
		puzzle.StageSolutions = make([][]byte, 3)
		puzzle.StageSolvers = make([][32]byte, 3)
	}
}

// IsExpired returns true if the puzzle has passed its time limit.
func (p *Puzzle) IsExpired() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return time.Now().After(p.ExpiresAt)
}

// IsSolved returns true if the puzzle has been solved.
func (p *Puzzle) IsSolved() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.State == PuzzleSolved
}

// SubmitSolution attempts to submit a solution to the puzzle.
func (p *Puzzle) SubmitSolution(solverKey [32]byte, solution []byte, generator PuzzleGenerator) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.checkPuzzleState(); err != nil {
		return err
	}

	if !generator.Verify(p, solution) {
		return ErrInvalidSolution
	}

	return p.recordSolution(solverKey, solution)
}

// checkPuzzleState validates the puzzle can accept solutions.
func (p *Puzzle) checkPuzzleState() error {
	if p.State == PuzzleExpired || time.Now().After(p.ExpiresAt) {
		p.State = PuzzleExpired
		return ErrPuzzleExpired
	}
	if p.State == PuzzleSolved && p.Type == PuzzleFragment {
		return ErrPuzzleAlreadySolved
	}
	return nil
}

// recordSolution records a valid solution based on puzzle type.
func (p *Puzzle) recordSolution(solverKey [32]byte, solution []byte) error {
	switch p.Type {
	case PuzzleFragment:
		p.recordFragmentSolution(solverKey, solution)
	case PuzzleMosaic:
		p.recordMosaicContribution(solverKey, solution)
	case PuzzleCascade:
		p.recordCascadeStage(solverKey, solution)
	}
	return nil
}

// recordFragmentSolution records the winning solution for a Fragment puzzle.
func (p *Puzzle) recordFragmentSolution(solverKey [32]byte, solution []byte) {
	now := time.Now()
	p.WinnerKey = &solverKey
	p.Solution = solution
	p.SolvedAt = &now
	p.State = PuzzleSolved
}

// recordMosaicContribution adds a contribution to a Mosaic puzzle.
func (p *Puzzle) recordMosaicContribution(solverKey [32]byte, solution []byte) {
	contrib := Contribution{
		SolverKey:   solverKey,
		Solution:    solution,
		SubmittedAt: time.Now(),
	}
	p.Contributions = append(p.Contributions, contrib)

	if len(p.Contributions) >= p.Fragments {
		p.State = PuzzleSolved
	}
}

// recordCascadeStage progresses a Cascade puzzle to the next stage.
func (p *Puzzle) recordCascadeStage(solverKey [32]byte, solution []byte) {
	if p.CurrentStage < p.Stages {
		p.StageSolutions[p.CurrentStage] = solution
		p.StageSolvers[p.CurrentStage] = solverKey
		p.CurrentStage++

		if p.CurrentStage >= p.Stages {
			p.State = PuzzleSolved
		}
	}
}

// ComputeResonanceBonus calculates the Resonance bonus for puzzle participation.
// Per ANONYMOUS_GAME_MECHANICS.md:
// puzzle_bonus = 4 * ln(1 + difficulty_factor * participation_count).
func ComputePuzzleBonus(difficulty, participantCount int) float64 {
	difficultyFactor := float64(difficulty) / float64(DefaultPuzzleDifficulty)
	return 4.0 * math.Log1p(difficultyFactor*float64(participantCount))
}

// PuzzleStore manages active and historical puzzles.
type PuzzleStore struct {
	mu      sync.RWMutex
	puzzles map[[32]byte]*Puzzle // By puzzle ID.
	active  []*Puzzle            // Active puzzles.
	history []*Puzzle            // Completed/expired puzzles.
}

// NewPuzzleStore creates a new puzzle store.
func NewPuzzleStore() *PuzzleStore {
	return &PuzzleStore{
		puzzles: make(map[[32]byte]*Puzzle),
	}
}

// AddPuzzle adds a new puzzle to the store.
func (s *PuzzleStore) AddPuzzle(p *Puzzle) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.puzzles[p.ID] = p
	s.active = append(s.active, p)
}

// GetPuzzle retrieves a puzzle by ID.
func (s *PuzzleStore) GetPuzzle(id [32]byte) *Puzzle {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.puzzles[id]
}

// GetActivePuzzles returns all active puzzles.
func (s *PuzzleStore) GetActivePuzzles() []*Puzzle {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*Puzzle
	for _, p := range s.active {
		if !p.IsExpired() && !p.IsSolved() {
			active = append(active, p)
		}
	}
	return active
}

// UpdatePuzzleState processes expired puzzles and moves them to history.
func (s *PuzzleStore) UpdatePuzzleState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stillActive []*Puzzle
	now := time.Now()

	for _, p := range s.active {
		p.mu.Lock()
		if now.After(p.ExpiresAt) && p.State == PuzzleActive {
			p.State = PuzzleExpired
		}

		if p.State == PuzzleSolved || p.State == PuzzleExpired {
			s.history = append(s.history, p)
		} else {
			stillActive = append(stillActive, p)
		}
		p.mu.Unlock()
	}

	s.active = stillActive
}

// GetPuzzleHistory returns completed/expired puzzles.
func (s *PuzzleStore) GetPuzzleHistory(limit int) []*Puzzle {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	// Return most recent first.
	result := make([]*Puzzle, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.history[len(s.history)-1-i]
	}
	return result
}

// Count returns the number of active puzzles.
func (s *PuzzleStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, p := range s.active {
		if !p.IsExpired() && !p.IsSolved() {
			count++
		}
	}
	return count
}

// GarbageCollect removes old history entries.
func (s *PuzzleStore) GarbageCollect(maxHistory int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var removed int
	s.history, removed = GarbageCollectHistory(s.history, s.puzzles, maxHistory, func(p *Puzzle) [32]byte { return p.ID })
	return removed
}

// PuzzleTypeString returns the human-readable name of a puzzle type.
func PuzzleTypeString(t PuzzleType) string {
	switch t {
	case PuzzleFragment:
		return "Fragment"
	case PuzzleMosaic:
		return "Mosaic"
	case PuzzleCascade:
		return "Cascade"
	default:
		return "Unknown"
	}
}

// PuzzleStateString returns the human-readable name of a puzzle state.
func PuzzleStateString(s PuzzleState) string {
	switch s {
	case PuzzlePending:
		return "Pending"
	case PuzzleActive:
		return "Active"
	case PuzzleSolved:
		return "Solved"
	case PuzzleExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}
