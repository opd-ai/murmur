// Package mechanics - Cipher Puzzle network propagation.
// Per ROADMAP.md line 415, publishes puzzle events to /murmur/anonymous/mechanics/1.0.
package mechanics

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// PuzzlePublisher topic.
const TopicAnonymousMechanics = "/murmur/anonymous/mechanics/1.0"

// Publisher provides an interface for publishing to GossipSub.
// This abstracts the networking layer from the mechanics package.
type Publisher interface {
	Publish(ctx context.Context, topicName string, data []byte) error
}

// Publication errors.
var (
	ErrPublisherNotSet   = errors.New("publisher not set")
	ErrInvalidPuzzle     = errors.New("invalid puzzle")
	ErrMissingSignature  = errors.New("missing signature")
	ErrSignatureFailed   = errors.New("signature verification failed")
	ErrMissingPrivateKey = errors.New("private key required for signing")
)

// PuzzlePublisher handles publishing puzzle events to the anonymous mechanics topic.
// Per TECHNICAL_IMPLEMENTATION.md, all mechanics events go to /murmur/anonymous/mechanics/1.0.
type PuzzlePublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewPuzzlePublisher creates a new puzzle publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewPuzzlePublisher(pub Publisher, privateKey ed25519.PrivateKey) *PuzzlePublisher {
	return &PuzzlePublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishPuzzleCreated publishes a puzzle creation event.
// This announces a new puzzle to the network.
func (p *PuzzlePublisher) PublishPuzzleCreated(ctx context.Context, puzzle *Puzzle) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if puzzle == nil {
		return ErrInvalidPuzzle
	}

	pbPuzzle := puzzleToProto(puzzle)
	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_CREATED,
		Puzzle:    pbPuzzle,
		PuzzleId:  puzzle.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishPuzzleSolved publishes a puzzle solved event.
// solution is the raw solution; only its hash is transmitted.
func (p *PuzzlePublisher) PublishPuzzleSolved(
	ctx context.Context,
	puzzleID [32]byte,
	solverKey [32]byte,
	solution []byte,
) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	solutionHash := blake3.Sum256(solution)
	event := &pb.PuzzleEvent{
		EventType:    pb.PuzzleEventType_PUZZLE_EVENT_SOLVED,
		PuzzleId:     puzzleID[:],
		SolverPubkey: solverKey[:],
		SolutionHash: solutionHash[:],
		Timestamp:    time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishPuzzleExpired publishes a puzzle expiration event.
func (p *PuzzlePublisher) PublishPuzzleExpired(ctx context.Context, puzzleID [32]byte) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_EXPIRED,
		PuzzleId:  puzzleID[:],
		Timestamp: time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishMosaicContribution publishes a Mosaic puzzle contribution.
func (p *PuzzlePublisher) PublishMosaicContribution(
	ctx context.Context,
	puzzleID [32]byte,
	solverKey [32]byte,
	fragmentIndex int,
	solution []byte,
) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	solutionHash := blake3.Sum256(solution)
	event := &pb.PuzzleEvent{
		EventType:    pb.PuzzleEventType_PUZZLE_EVENT_CONTRIBUTION,
		PuzzleId:     puzzleID[:],
		SolverPubkey: solverKey[:],
		SolutionHash: solutionHash[:],
		Timestamp:    time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishCascadeStage publishes a Cascade puzzle stage completion.
func (p *PuzzlePublisher) PublishCascadeStage(
	ctx context.Context,
	puzzleID [32]byte,
	solverKey [32]byte,
	stageIndex int,
	solution []byte,
) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	solutionHash := blake3.Sum256(solution)
	event := &pb.PuzzleEvent{
		EventType:    pb.PuzzleEventType_PUZZLE_EVENT_STAGE,
		PuzzleId:     puzzleID[:],
		SolverPubkey: solverKey[:],
		SolutionHash: solutionHash[:],
		Timestamp:    time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it.
func (p *PuzzlePublisher) signAndPublish(ctx context.Context, event *pb.PuzzleEvent) error {
	if p.privateKey == nil {
		return ErrMissingPrivateKey
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(p.privateKey, signedData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_PuzzleEvent{
			PuzzleEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling puzzle event: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// buildPuzzleSignedData builds the data to sign for a puzzle event.
func buildPuzzleSignedData(event *pb.PuzzleEvent) []byte {
	h := blake3.New()
	h.Write([]byte{byte(event.EventType)})
	h.Write(event.PuzzleId)
	h.Write(event.SolverPubkey)
	h.Write(event.SolutionHash)
	var ts [8]byte
	ts[0] = byte(event.Timestamp >> 56)
	ts[1] = byte(event.Timestamp >> 48)
	ts[2] = byte(event.Timestamp >> 40)
	ts[3] = byte(event.Timestamp >> 32)
	ts[4] = byte(event.Timestamp >> 24)
	ts[5] = byte(event.Timestamp >> 16)
	ts[6] = byte(event.Timestamp >> 8)
	ts[7] = byte(event.Timestamp)
	h.Write(ts[:])
	return h.Sum(nil)
}

// PuzzleReceiver handles incoming puzzle events from the network.
type PuzzleReceiver struct {
	store *PuzzleStore
}

// NewPuzzleReceiver creates a new puzzle event receiver.
func NewPuzzleReceiver(store *PuzzleStore) *PuzzleReceiver {
	return &PuzzleReceiver{store: store}
}

// HandlePuzzleEvent processes an incoming puzzle event.
func (r *PuzzleReceiver) HandlePuzzleEvent(
	ctx context.Context,
	event *pb.PuzzleEvent,
	senderPubkey []byte,
) error {
	if err := verifyPuzzleEventSignature(event, senderPubkey); err != nil {
		return err
	}

	switch event.EventType {
	case pb.PuzzleEventType_PUZZLE_EVENT_CREATED:
		return r.handlePuzzleCreated(ctx, event)
	case pb.PuzzleEventType_PUZZLE_EVENT_SOLVED:
		return r.handlePuzzleSolved(ctx, event)
	case pb.PuzzleEventType_PUZZLE_EVENT_EXPIRED:
		return r.handlePuzzleExpired(ctx, event)
	case pb.PuzzleEventType_PUZZLE_EVENT_CONTRIBUTION:
		return r.handleMosaicContribution(ctx, event)
	case pb.PuzzleEventType_PUZZLE_EVENT_STAGE:
		return r.handleCascadeStage(ctx, event)
	default:
		return fmt.Errorf("unknown puzzle event type: %d", event.EventType)
	}
}

// verifyPuzzleEventSignature verifies the event signature.
func verifyPuzzleEventSignature(event *pb.PuzzleEvent, senderPubkey []byte) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}
	if len(senderPubkey) != ed25519.PublicKeySize {
		return ErrSignatureFailed
	}

	signedData := buildPuzzleSignedData(event)
	if !ed25519.Verify(senderPubkey, signedData, event.Signature) {
		return ErrSignatureFailed
	}
	return nil
}

// handlePuzzleCreated processes a puzzle creation event.
func (r *PuzzleReceiver) handlePuzzleCreated(ctx context.Context, event *pb.PuzzleEvent) error {
	if event.Puzzle == nil {
		return ErrInvalidPuzzle
	}

	puzzle := protoToPuzzle(event.Puzzle)
	r.store.AddPuzzle(puzzle)
	return nil
}

// handlePuzzleSolved processes a puzzle solved event.
func (r *PuzzleReceiver) handlePuzzleSolved(ctx context.Context, event *pb.PuzzleEvent) error {
	var puzzleID [32]byte
	copy(puzzleID[:], event.PuzzleId)

	puzzle := r.store.GetPuzzle(puzzleID)
	if puzzle == nil {
		return ErrPuzzleNotFound
	}

	puzzle.mu.Lock()
	defer puzzle.mu.Unlock()

	if puzzle.State == PuzzleSolved {
		return nil
	}

	var solverKey [32]byte
	copy(solverKey[:], event.SolverPubkey)

	now := time.Now()
	puzzle.WinnerKey = &solverKey
	puzzle.SolvedAt = &now
	puzzle.State = PuzzleSolved

	return nil
}

// handlePuzzleExpired processes a puzzle expiration event.
func (r *PuzzleReceiver) handlePuzzleExpired(ctx context.Context, event *pb.PuzzleEvent) error {
	var puzzleID [32]byte
	copy(puzzleID[:], event.PuzzleId)

	puzzle := r.store.GetPuzzle(puzzleID)
	if puzzle == nil {
		return ErrPuzzleNotFound
	}

	puzzle.mu.Lock()
	defer puzzle.mu.Unlock()

	if puzzle.State != PuzzleActive {
		return nil
	}

	puzzle.State = PuzzleExpired
	return nil
}

// handleMosaicContribution processes a Mosaic puzzle contribution.
func (r *PuzzleReceiver) handleMosaicContribution(ctx context.Context, event *pb.PuzzleEvent) error {
	var puzzleID [32]byte
	copy(puzzleID[:], event.PuzzleId)

	puzzle := r.store.GetPuzzle(puzzleID)
	if puzzle == nil {
		return ErrPuzzleNotFound
	}

	puzzle.mu.Lock()
	defer puzzle.mu.Unlock()

	if puzzle.Type != PuzzleMosaic {
		return ErrInvalidPuzzleType
	}
	if puzzle.State != PuzzleActive {
		return ErrPuzzleAlreadySolved
	}

	var solverKey [32]byte
	copy(solverKey[:], event.SolverPubkey)

	contrib := Contribution{
		SolverKey:   solverKey,
		Solution:    event.SolutionHash,
		SubmittedAt: time.Unix(event.Timestamp, 0),
	}
	puzzle.Contributions = append(puzzle.Contributions, contrib)

	if len(puzzle.Contributions) >= puzzle.Fragments {
		puzzle.State = PuzzleSolved
	}

	return nil
}

// handleCascadeStage processes a Cascade puzzle stage completion.
func (r *PuzzleReceiver) handleCascadeStage(ctx context.Context, event *pb.PuzzleEvent) error {
	var puzzleID [32]byte
	copy(puzzleID[:], event.PuzzleId)

	puzzle := r.store.GetPuzzle(puzzleID)
	if puzzle == nil {
		return ErrPuzzleNotFound
	}

	puzzle.mu.Lock()
	defer puzzle.mu.Unlock()

	if puzzle.Type != PuzzleCascade {
		return ErrInvalidPuzzleType
	}
	if puzzle.State != PuzzleActive {
		return ErrPuzzleAlreadySolved
	}
	if puzzle.CurrentStage >= puzzle.Stages {
		return nil
	}

	var solverKey [32]byte
	copy(solverKey[:], event.SolverPubkey)

	puzzle.StageSolutions[puzzle.CurrentStage] = event.SolutionHash
	puzzle.StageSolvers[puzzle.CurrentStage] = solverKey
	puzzle.CurrentStage++

	if puzzle.CurrentStage >= puzzle.Stages {
		puzzle.State = PuzzleSolved
	}

	return nil
}
