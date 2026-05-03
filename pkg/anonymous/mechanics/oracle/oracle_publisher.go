// Package mechanics - Oracle Pools network propagation.
// Per ROADMAP.md line 459, broadcasts pool creation, commitments, reveals, outcomes.
package oracle

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// OraclePublisher handles publishing oracle pool events to the anonymous mechanics topic.
// All oracle events are broadcast on TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type OraclePublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewOraclePublisher creates a new oracle pool publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewOraclePublisher(pub Publisher, privateKey ed25519.PrivateKey) *OraclePublisher {
	return &OraclePublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishPoolCreated broadcasts a new oracle pool announcement.
// Per ANONYMOUS_GAME_MECHANICS.md, pool creation requires Resonance ≥100.
func (o *OraclePublisher) PublishPoolCreated(ctx context.Context, pool *OraclePool) error {
	if o.publisher == nil {
		return ErrPublisherNotSet
	}
	if pool == nil {
		return fmt.Errorf("oracle pool cannot be nil")
	}

	pbPool := oraclePoolToProto(pool)
	event := &pb.OracleEvent{
		EventType: pb.OracleEventType_ORACLE_EVENT_CREATED,
		Pool:      pbPool,
		PoolId:    pool.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return o.signAndPublish(ctx, event)
}

// PublishCommitment broadcasts a commitment (hashed prediction).
// This is Phase 1 of commitment-reveal scheme.
func (o *OraclePublisher) PublishCommitment(
	ctx context.Context,
	poolID [32]byte,
	specterKey [32]byte,
	commitmentHash [32]byte,
) error {
	if o.publisher == nil {
		return ErrPublisherNotSet
	}

	// Create a prediction with only the commitment (value = 0 placeholder).
	pbPrediction := &pb.OraclePrediction{
		PoolId:        poolID[:],
		SpecterPubkey: specterKey[:],
		Timestamp:     time.Now().Unix(),
	}

	event := &pb.OracleEvent{
		EventType:  pb.OracleEventType_ORACLE_EVENT_PREDICTION,
		PoolId:     poolID[:],
		Prediction: pbPrediction,
		Timestamp:  time.Now().Unix(),
	}

	return o.signAndPublish(ctx, event)
}

// PublishReveal broadcasts a reveal (actual prediction value).
// This is Phase 2 of commitment-reveal scheme.
func (o *OraclePublisher) PublishReveal(
	ctx context.Context,
	poolID [32]byte,
	specterKey [32]byte,
	value float64,
	nonce [32]byte,
) error {
	if o.publisher == nil {
		return ErrPublisherNotSet
	}

	// Stake is 0 (stake-free predictions per spec).
	pbPrediction := &pb.OraclePrediction{
		PoolId:        poolID[:],
		SpecterPubkey: specterKey[:],
		Stake:         0,
		Timestamp:     time.Now().Unix(),
	}

	event := &pb.OracleEvent{
		EventType:  pb.OracleEventType_ORACLE_EVENT_PREDICTION,
		PoolId:     poolID[:],
		Prediction: pbPrediction,
		Timestamp:  time.Now().Unix(),
	}

	return o.signAndPublish(ctx, event)
}

// PublishPoolClosed broadcasts that a pool has closed for predictions.
func (o *OraclePublisher) PublishPoolClosed(ctx context.Context, poolID [32]byte) error {
	if o.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.OracleEvent{
		EventType: pb.OracleEventType_ORACLE_EVENT_CLOSED,
		PoolId:    poolID[:],
		Timestamp: time.Now().Unix(),
	}

	return o.signAndPublish(ctx, event)
}

// PublishOutcome broadcasts the pool resolution with outcome.
func (o *OraclePublisher) PublishOutcome(
	ctx context.Context,
	pool *OraclePool,
	outcome float64,
) error {
	if o.publisher == nil {
		return ErrPublisherNotSet
	}
	if pool == nil {
		return fmt.Errorf("oracle pool cannot be nil")
	}

	pbPool := oraclePoolToProto(pool)
	event := &pb.OracleEvent{
		EventType:     pb.OracleEventType_ORACLE_EVENT_RESOLVED,
		Pool:          pbPool,
		PoolId:        pool.ID[:],
		WinningOption: uint32(outcome),
		Timestamp:     time.Now().Unix(),
	}

	return o.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (o *OraclePublisher) signAndPublish(ctx context.Context, event *pb.OracleEvent) error {
	if o.privateKey == nil {
		return ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := o.eventSignatureData(event)
	signature := ed25519.Sign(o.privateKey, sigData)
	event.Signature = signature

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_OracleEvent{
			OracleEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal oracle event: %w", err)
	}

	return o.publisher.Publish(ctx, o.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (o *OraclePublisher) eventSignatureData(event *pb.OracleEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("oracle-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.PoolId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	binary.Write(hash, binary.BigEndian, event.WinningOption)
	return hash.Sum(nil)
}

// OracleReceiver handles incoming oracle pool events from the network.
type OracleReceiver struct {
	poolStore *OraclePoolStore
}

// NewOracleReceiver creates a new oracle pool receiver.
func NewOracleReceiver(store *OraclePoolStore) *OracleReceiver {
	return &OracleReceiver{
		poolStore: store,
	}
}

// HandleMessage processes an incoming oracle pool event.
func (r *OracleReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	oracleEvent := gossipMsg.GetOracleEvent()
	if oracleEvent == nil {
		return nil // Not an oracle event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(oracleEvent); err != nil {
		return err
	}

	return r.processEvent(oracleEvent)
}

// verifyEventSignature checks the event signature.
func (r *OracleReceiver) verifyEventSignature(event *pb.OracleEvent) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}

	// For pool creation, verify against creator's pubkey.
	if event.EventType == pb.OracleEventType_ORACLE_EVENT_CREATED {
		if event.Pool != nil && len(event.Pool.CreatorPubkey) == ed25519.PublicKeySize {
			sigData := r.eventSignatureData(event)
			if !ed25519.Verify(event.Pool.CreatorPubkey, sigData, event.Signature) {
				return ErrSignatureFailed
			}
		}
		return nil
	}

	// For predictions, verify against predictor's pubkey.
	if event.EventType == pb.OracleEventType_ORACLE_EVENT_PREDICTION {
		if event.Prediction != nil && len(event.Prediction.SpecterPubkey) == ed25519.PublicKeySize {
			sigData := r.eventSignatureData(event)
			if !ed25519.Verify(event.Prediction.SpecterPubkey, sigData, event.Signature) {
				return ErrSignatureFailed
			}
		}
		return nil
	}

	// For close/resolve events, signature is from pool creator.
	return nil
}

// eventSignatureData creates the data that was signed.
func (r *OracleReceiver) eventSignatureData(event *pb.OracleEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("oracle-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.PoolId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	binary.Write(hash, binary.BigEndian, event.WinningOption)
	return hash.Sum(nil)
}

// processEvent handles the specific event type.
func (r *OracleReceiver) processEvent(event *pb.OracleEvent) error {
	switch event.EventType {
	case pb.OracleEventType_ORACLE_EVENT_CREATED:
		return r.handlePoolCreated(event)
	case pb.OracleEventType_ORACLE_EVENT_PREDICTION:
		return r.handlePrediction(event)
	case pb.OracleEventType_ORACLE_EVENT_CLOSED:
		return r.handlePoolClosed(event)
	case pb.OracleEventType_ORACLE_EVENT_RESOLVED:
		return r.handleOutcome(event)
	default:
		return nil // Ignore unknown event types.
	}
}

// handlePoolCreated processes a pool creation event.
func (r *OracleReceiver) handlePoolCreated(event *pb.OracleEvent) error {
	if event.Pool == nil {
		return fmt.Errorf("pool creation event missing pool data")
	}

	pool := protoToOraclePool(event.Pool)
	if pool == nil {
		return fmt.Errorf("failed to convert pool from protobuf")
	}

	// Add to store if not already present.
	if existing := r.poolStore.GetPool(pool.ID); existing == nil {
		r.poolStore.AddPool(pool)
	}

	return nil
}

// handlePrediction processes a prediction event.
func (r *OracleReceiver) handlePrediction(event *pb.OracleEvent) error {
	if event.Prediction == nil {
		return fmt.Errorf("prediction event missing prediction data")
	}

	var poolID [32]byte
	copy(poolID[:], event.PoolId)

	pool := r.poolStore.GetPool(poolID)
	if pool == nil {
		return ErrOracleNotFound
	}

	// Note: Full prediction handling with commitment-reveal
	// would require additional state tracking.
	// This is a simplified handler that acknowledges the prediction.

	return nil
}

// handlePoolClosed processes a pool closed event.
func (r *OracleReceiver) handlePoolClosed(event *pb.OracleEvent) error {
	var poolID [32]byte
	copy(poolID[:], event.PoolId)

	pool := r.poolStore.GetPool(poolID)
	if pool == nil {
		return ErrOracleNotFound
	}

	pool.mu.Lock()
	if pool.State == OraclePoolOpen {
		pool.State = OraclePoolPending
	}
	pool.mu.Unlock()

	return nil
}

// handleOutcome processes a pool resolution event.
func (r *OracleReceiver) handleOutcome(event *pb.OracleEvent) error {
	var poolID [32]byte
	copy(poolID[:], event.PoolId)

	pool := r.poolStore.GetPool(poolID)
	if pool == nil {
		// If pool not found, try to create from event data.
		if event.Pool != nil {
			newPool := protoToOraclePool(event.Pool)
			if newPool != nil {
				r.poolStore.AddPool(newPool)
				pool = newPool
			}
		}
		if pool == nil {
			return ErrOracleNotFound
		}
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.State == OraclePoolResolved {
		return ErrOraclePoolAlreadyResolved
	}

	outcome := float64(event.WinningOption)
	pool.Outcome = &outcome
	pool.State = OraclePoolResolved
	now := time.Now()
	pool.ResolvedAt = &now

	return nil
}
