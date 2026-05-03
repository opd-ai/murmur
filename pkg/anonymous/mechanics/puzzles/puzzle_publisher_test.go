package puzzles

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// mockPublisher is a mock Publisher for testing.
type mockPublisher struct {
	published []publishedMessage
}

type publishedMessage struct {
	topic string
	data  []byte
}

func (m *mockPublisher) Publish(_ context.Context, topic string, data []byte) error {
	m.published = append(m.published, publishedMessage{topic: topic, data: data})
	return nil
}

func (m *mockPublisher) lastMessage() (*pb.GossipMessage, error) {
	if len(m.published) == 0 {
		return nil, nil
	}
	msg := &pb.GossipMessage{}
	err := proto.Unmarshal(m.published[len(m.published)-1].data, msg)
	return msg, err
}

func TestPuzzlePublisher_PublishPuzzleCreated(t *testing.T) {
	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, privKey)

	var seed [32]byte
	copy(seed[:], []byte("test-seed-for-puzzle"))
	var initiator [32]byte
	copy(initiator[:], pub)

	puzzle, err := NewPuzzle(PuzzleFragment, seed, 20, PuzzleDuration15Min, initiator)
	if err != nil {
		t.Fatalf("NewPuzzle: %v", err)
	}

	err = publisher.PublishPuzzleCreated(context.Background(), puzzle)
	if err != nil {
		t.Fatalf("PublishPuzzleCreated: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.published))
	}
	if mockPub.published[0].topic != TopicAnonymousMechanics {
		t.Errorf("wrong topic: %s", mockPub.published[0].topic)
	}

	msg, err := mockPub.lastMessage()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	puzzleEvent := msg.GetPuzzleEvent()
	if puzzleEvent == nil {
		t.Fatal("expected puzzle event")
	}
	if puzzleEvent.EventType != pb.PuzzleEventType_PUZZLE_EVENT_CREATED {
		t.Errorf("wrong event type: %v", puzzleEvent.EventType)
	}
	if puzzleEvent.Puzzle == nil {
		t.Error("expected puzzle in event")
	}
	if len(puzzleEvent.Signature) == 0 {
		t.Error("expected signature")
	}
}

func TestPuzzlePublisher_PublishPuzzleSolved(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, privKey)

	var puzzleID [32]byte
	copy(puzzleID[:], []byte("puzzle-id-12345678901234567890"))
	var solverKey [32]byte
	copy(solverKey[:], []byte("solver-key-123456789012345678"))
	solution := []byte("winning-solution")

	err := publisher.PublishPuzzleSolved(context.Background(), puzzleID, solverKey, solution)
	if err != nil {
		t.Fatalf("PublishPuzzleSolved: %v", err)
	}

	msg, err := mockPub.lastMessage()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	event := msg.GetPuzzleEvent()
	if event == nil {
		t.Fatal("expected puzzle event")
	}
	if event.EventType != pb.PuzzleEventType_PUZZLE_EVENT_SOLVED {
		t.Errorf("wrong event type: %v", event.EventType)
	}
	if !bytes.Equal(event.PuzzleId, puzzleID[:]) {
		t.Error("puzzle ID mismatch")
	}
	if !bytes.Equal(event.SolverPubkey, solverKey[:]) {
		t.Error("solver key mismatch")
	}
	if len(event.SolutionHash) != 32 {
		t.Error("expected 32-byte solution hash")
	}
}

func TestPuzzlePublisher_PublishPuzzleExpired(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, privKey)

	var puzzleID [32]byte
	copy(puzzleID[:], []byte("expired-puzzle-id-1234567890"))

	err := publisher.PublishPuzzleExpired(context.Background(), puzzleID)
	if err != nil {
		t.Fatalf("PublishPuzzleExpired: %v", err)
	}

	msg, err := mockPub.lastMessage()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	event := msg.GetPuzzleEvent()
	if event == nil {
		t.Fatal("expected puzzle event")
	}
	if event.EventType != pb.PuzzleEventType_PUZZLE_EVENT_EXPIRED {
		t.Errorf("wrong event type: %v", event.EventType)
	}
}

func TestPuzzlePublisher_PublishMosaicContribution(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, privKey)

	var puzzleID [32]byte
	copy(puzzleID[:], []byte("mosaic-puzzle-id-12345678901"))
	var solverKey [32]byte
	copy(solverKey[:], []byte("contributor-key-123456789012"))
	solution := []byte("fragment-solution")

	err := publisher.PublishMosaicContribution(context.Background(), puzzleID, solverKey, 2, solution)
	if err != nil {
		t.Fatalf("PublishMosaicContribution: %v", err)
	}

	msg, err := mockPub.lastMessage()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	event := msg.GetPuzzleEvent()
	if event == nil {
		t.Fatal("expected puzzle event")
	}
	if event.EventType != pb.PuzzleEventType_PUZZLE_EVENT_CONTRIBUTION {
		t.Errorf("wrong event type: %v", event.EventType)
	}
}

func TestPuzzlePublisher_PublishCascadeStage(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, privKey)

	var puzzleID [32]byte
	copy(puzzleID[:], []byte("cascade-puzzle-id-1234567890"))
	var solverKey [32]byte
	copy(solverKey[:], []byte("stage-solver-key-12345678901"))
	solution := []byte("stage-1-solution")

	err := publisher.PublishCascadeStage(context.Background(), puzzleID, solverKey, 1, solution)
	if err != nil {
		t.Fatalf("PublishCascadeStage: %v", err)
	}

	msg, err := mockPub.lastMessage()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	event := msg.GetPuzzleEvent()
	if event == nil {
		t.Fatal("expected puzzle event")
	}
	if event.EventType != pb.PuzzleEventType_PUZZLE_EVENT_STAGE {
		t.Errorf("wrong event type: %v", event.EventType)
	}
}

func TestPuzzlePublisher_NoPublisher(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	publisher := NewPuzzlePublisher(nil, privKey)

	var seed [32]byte
	puzzle := &Puzzle{Seed: seed}

	err := publisher.PublishPuzzleCreated(context.Background(), puzzle)
	if err != ErrPublisherNotSet {
		t.Errorf("expected ErrPublisherNotSet, got %v", err)
	}
}

func TestPuzzlePublisher_NoPrivateKey(t *testing.T) {
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, nil)

	var seed [32]byte
	puzzle := &Puzzle{Seed: seed}

	err := publisher.PublishPuzzleCreated(context.Background(), puzzle)
	if err != ErrMissingPrivateKey {
		t.Errorf("expected ErrMissingPrivateKey, got %v", err)
	}
}

func TestPuzzleReceiver_HandlePuzzleCreated(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var puzzleID [32]byte
	copy(puzzleID[:], []byte("new-puzzle-id-1234567890123"))
	var creator [32]byte
	copy(creator[:], pub)

	pbPuzzle := &pb.CipherPuzzle{
		Id:            puzzleID[:],
		CreatorPubkey: creator[:],
		Difficulty:    20,
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     time.Now().Add(15 * time.Minute).Unix(),
		State:         pb.PuzzleState_PUZZLE_STATE_ACTIVE,
	}

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_CREATED,
		Puzzle:    pbPuzzle,
		PuzzleId:  puzzleID[:],
		Timestamp: time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != nil {
		t.Fatalf("HandlePuzzleEvent: %v", err)
	}

	stored := store.GetPuzzle(puzzleID)
	if stored == nil {
		t.Error("puzzle not stored")
	}
}

func TestPuzzleReceiver_HandlePuzzleSolved(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var seed [32]byte
	var initiator [32]byte
	copy(initiator[:], pub)

	puzzle, _ := NewPuzzle(PuzzleFragment, seed, 20, PuzzleDuration15Min, initiator)
	store.AddPuzzle(puzzle)

	var solverKey [32]byte
	copy(solverKey[:], []byte("solver-key-123456789012345678"))

	event := &pb.PuzzleEvent{
		EventType:    pb.PuzzleEventType_PUZZLE_EVENT_SOLVED,
		PuzzleId:     puzzle.ID[:],
		SolverPubkey: solverKey[:],
		SolutionHash: make([]byte, 32),
		Timestamp:    time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != nil {
		t.Fatalf("HandlePuzzleEvent: %v", err)
	}

	if puzzle.State != PuzzleSolved {
		t.Error("puzzle not marked solved")
	}
	if puzzle.WinnerKey == nil {
		t.Error("winner not set")
	}
}

func TestPuzzleReceiver_HandlePuzzleExpired(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var seed [32]byte
	var initiator [32]byte
	copy(initiator[:], pub)

	puzzle, _ := NewPuzzle(PuzzleFragment, seed, 20, PuzzleDuration15Min, initiator)
	store.AddPuzzle(puzzle)

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_EXPIRED,
		PuzzleId:  puzzle.ID[:],
		Timestamp: time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != nil {
		t.Fatalf("HandlePuzzleEvent: %v", err)
	}

	if puzzle.State != PuzzleExpired {
		t.Error("puzzle not marked expired")
	}
}

func TestPuzzleReceiver_HandleMosaicContribution(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var seed [32]byte
	var initiator [32]byte
	copy(initiator[:], pub)

	puzzle, _ := NewPuzzle(PuzzleMosaic, seed, 20, PuzzleDuration15Min, initiator)
	store.AddPuzzle(puzzle)

	var solverKey [32]byte
	copy(solverKey[:], []byte("contributor-key-123456789012"))

	event := &pb.PuzzleEvent{
		EventType:    pb.PuzzleEventType_PUZZLE_EVENT_CONTRIBUTION,
		PuzzleId:     puzzle.ID[:],
		SolverPubkey: solverKey[:],
		SolutionHash: make([]byte, 32),
		Timestamp:    time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != nil {
		t.Fatalf("HandlePuzzleEvent: %v", err)
	}

	if len(puzzle.Contributions) != 1 {
		t.Errorf("expected 1 contribution, got %d", len(puzzle.Contributions))
	}
}

func TestPuzzleReceiver_HandleCascadeStage(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var seed [32]byte
	var initiator [32]byte
	copy(initiator[:], pub)

	puzzle, _ := NewPuzzle(PuzzleCascade, seed, 20, PuzzleDuration15Min, initiator)
	store.AddPuzzle(puzzle)

	var solverKey [32]byte
	copy(solverKey[:], []byte("stage-solver-key-12345678901"))

	event := &pb.PuzzleEvent{
		EventType:    pb.PuzzleEventType_PUZZLE_EVENT_STAGE,
		PuzzleId:     puzzle.ID[:],
		SolverPubkey: solverKey[:],
		SolutionHash: make([]byte, 32),
		Timestamp:    time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != nil {
		t.Fatalf("HandlePuzzleEvent: %v", err)
	}

	if puzzle.CurrentStage != 1 {
		t.Errorf("expected stage 1, got %d", puzzle.CurrentStage)
	}
}

func TestPuzzleReceiver_InvalidSignature(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, wrongKey, _ := ed25519.GenerateKey(rand.Reader)

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_EXPIRED,
		PuzzleId:  make([]byte, 32),
		Timestamp: time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(wrongKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != ErrSignatureFailed {
		t.Errorf("expected ErrSignatureFailed, got %v", err)
	}
}

func TestPuzzleReceiver_MissingSignature(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, _, _ := ed25519.GenerateKey(rand.Reader)

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_EXPIRED,
		PuzzleId:  make([]byte, 32),
		Timestamp: time.Now().Unix(),
	}

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != ErrMissingSignature {
		t.Errorf("expected ErrMissingSignature, got %v", err)
	}
}

func TestPuzzleReceiver_PuzzleNotFound(t *testing.T) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var puzzleID [32]byte
	copy(puzzleID[:], []byte("nonexistent-puzzle-id-123456"))

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_SOLVED,
		PuzzleId:  puzzleID[:],
		Timestamp: time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandlePuzzleEvent(context.Background(), event, pub)
	if err != ErrPuzzleNotFound {
		t.Errorf("expected ErrPuzzleNotFound, got %v", err)
	}
}

func TestPuzzleToProtoRoundTrip(t *testing.T) {
	var seed [32]byte
	copy(seed[:], []byte("test-seed-for-roundtrip-test"))
	var initiator [32]byte
	copy(initiator[:], []byte("initiator-key-1234567890123"))

	original, err := NewPuzzle(PuzzleFragment, seed, 25, PuzzleDuration30Min, initiator)
	if err != nil {
		t.Fatalf("NewPuzzle: %v", err)
	}

	pb := puzzleToProto(original)
	reconstructed := protoToPuzzle(pb)

	if reconstructed.ID != original.ID {
		t.Error("ID mismatch")
	}
	if reconstructed.Difficulty != original.Difficulty {
		t.Error("Difficulty mismatch")
	}
	if reconstructed.InitiatorKey != original.InitiatorKey {
		t.Error("InitiatorKey mismatch")
	}
	if reconstructed.State != original.State {
		t.Error("State mismatch")
	}
}

func BenchmarkPuzzlePublish(b *testing.B) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewPuzzlePublisher(mockPub, privKey)

	var seed [32]byte
	var initiator [32]byte
	puzzle, _ := NewPuzzle(PuzzleFragment, seed, 20, PuzzleDuration15Min, initiator)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = publisher.PublishPuzzleCreated(ctx, puzzle)
	}
}

func BenchmarkPuzzleReceive(b *testing.B) {
	store := NewPuzzleStore()
	receiver := NewPuzzleReceiver(store)

	pub, privKey, _ := ed25519.GenerateKey(rand.Reader)

	var puzzleID [32]byte
	var creator [32]byte
	copy(creator[:], pub)

	pbPuzzle := &pb.CipherPuzzle{
		Id:            puzzleID[:],
		CreatorPubkey: creator[:],
		Difficulty:    20,
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     time.Now().Add(15 * time.Minute).Unix(),
		State:         pb.PuzzleState_PUZZLE_STATE_ACTIVE,
	}

	event := &pb.PuzzleEvent{
		EventType: pb.PuzzleEventType_PUZZLE_EVENT_CREATED,
		Puzzle:    pbPuzzle,
		PuzzleId:  puzzleID[:],
		Timestamp: time.Now().Unix(),
	}

	signedData := buildPuzzleSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.PuzzleId[0] = byte(i)
		pbPuzzle.Id[0] = byte(i)
		signedData := buildPuzzleSignedData(event)
		event.Signature = ed25519.Sign(privKey, signedData)
		_ = receiver.HandlePuzzleEvent(ctx, event, pub)
	}
}
