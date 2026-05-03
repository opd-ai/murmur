// Package mechanics - Surface Spark network propagation.
// Per ROADMAP.md line 558, publishes spark events to /murmur/anonymous/mechanics/1.0.
package sparks

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

// Spark publication errors.
var (
	ErrInvalidSparkPub    = errors.New("invalid spark")
	ErrSparkNotActivePub  = errors.New("spark is not active")
	ErrSparkResponseEmpty = errors.New("spark response empty")
)

// SparkPublisher handles publishing spark events to the anonymous mechanics topic.
// Per TECHNICAL_IMPLEMENTATION.md, all mechanics events go to /murmur/anonymous/mechanics/1.0.
type SparkPublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewSparkPublisher creates a new spark publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewSparkPublisher(pub Publisher, privateKey ed25519.PrivateKey) *SparkPublisher {
	return &SparkPublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishSparkCreated publishes a spark creation event.
// This announces a new Surface Spark to the network.
func (p *SparkPublisher) PublishSparkCreated(ctx context.Context, spark *Spark) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if spark == nil {
		return ErrInvalidSparkPub
	}

	pbSpark := sparkToProto(spark)
	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_CREATED,
		Spark:     pbSpark,
		SparkId:   spark.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishSparkResponse publishes a response to a spark challenge.
func (p *SparkPublisher) PublishSparkResponse(
	ctx context.Context,
	sparkID [32]byte,
	responderKey []byte,
	waveID [32]byte,
) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	response := &pb.SparkResponse{
		SparkId:         sparkID[:],
		ResponderPubkey: responderKey,
		WaveId:          waveID[:],
		RespondedAt:     time.Now().Unix(),
	}

	// Sign the response.
	responseData := buildSparkResponseSignedData(response)
	response.Signature = ed25519.Sign(p.privateKey, responseData)

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_RESPONSE,
		SparkId:   sparkID[:],
		Response:  response,
		Timestamp: time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishSparkCompleted publishes a spark completion event with winner.
func (p *SparkPublisher) PublishSparkCompleted(
	ctx context.Context,
	sparkID [32]byte,
	winnerKey []byte,
) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.SparkEvent{
		EventType:    pb.SparkEventType_SPARK_EVENT_COMPLETED,
		SparkId:      sparkID[:],
		WinnerPubkey: winnerKey,
		Timestamp:    time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishSparkExpired publishes a spark expiration event.
func (p *SparkPublisher) PublishSparkExpired(ctx context.Context, sparkID [32]byte) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_EXPIRED,
		SparkId:   sparkID[:],
		Timestamp: time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// PublishSparkCancelled publishes a spark cancellation event.
func (p *SparkPublisher) PublishSparkCancelled(ctx context.Context, sparkID [32]byte) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_CANCELLED,
		SparkId:   sparkID[:],
		Timestamp: time.Now().Unix(),
	}

	return p.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it.
func (p *SparkPublisher) signAndPublish(ctx context.Context, event *pb.SparkEvent) error {
	if p.privateKey == nil {
		return ErrMissingPrivateKey
	}

	signedData := buildSparkSignedData(event)
	event.Signature = ed25519.Sign(p.privateKey, signedData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_SparkEvent{
			SparkEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling spark event: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// buildSparkSignedData builds the data to sign for a spark event.
func buildSparkSignedData(event *pb.SparkEvent) []byte {
	h := blake3.New()
	h.Write([]byte{byte(event.EventType)})
	h.Write(event.SparkId)
	h.Write(event.WinnerPubkey)
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

// buildSparkResponseSignedData builds the data to sign for a spark response.
func buildSparkResponseSignedData(response *pb.SparkResponse) []byte {
	h := blake3.New()
	h.Write(response.SparkId)
	h.Write(response.ResponderPubkey)
	h.Write(response.WaveId)
	var ts [8]byte
	ts[0] = byte(response.RespondedAt >> 56)
	ts[1] = byte(response.RespondedAt >> 48)
	ts[2] = byte(response.RespondedAt >> 40)
	ts[3] = byte(response.RespondedAt >> 32)
	ts[4] = byte(response.RespondedAt >> 24)
	ts[5] = byte(response.RespondedAt >> 16)
	ts[6] = byte(response.RespondedAt >> 8)
	ts[7] = byte(response.RespondedAt)
	h.Write(ts[:])
	return h.Sum(nil)
}

// sparkToProto converts a Spark to its protobuf representation.
func sparkToProto(s *Spark) *pb.SurfaceSpark {
	pbSpark := &pb.SurfaceSpark{
		Id:              s.ID[:],
		SparkType:       pb.SparkType(s.Type),
		InitiatorPubkey: s.InitiatorID,
		Prompt:          s.Prompt,
		CreatedAt:       s.CreatedAt.Unix(),
		ExpiresAt:       s.ExpiresAt.Unix(),
		State:           pb.SparkState(s.State),
	}
	if len(s.WinnerID) > 0 {
		pbSpark.WinnerPubkey = s.WinnerID
	}
	if !s.WinnerTime.IsZero() {
		pbSpark.CompletedAt = s.WinnerTime.Unix()
	}
	return pbSpark
}

// protoToSpark converts a protobuf SurfaceSpark to a Spark.
func protoToSpark(p *pb.SurfaceSpark) *Spark {
	s := &Spark{
		Type:      SparkType(p.SparkType),
		Prompt:    p.Prompt,
		CreatedAt: time.Unix(p.CreatedAt, 0),
		ExpiresAt: time.Unix(p.ExpiresAt, 0),
		State:     SparkState(p.State),
	}
	copy(s.ID[:], p.Id)
	s.InitiatorID = make([]byte, len(p.InitiatorPubkey))
	copy(s.InitiatorID, p.InitiatorPubkey)
	if len(p.WinnerPubkey) > 0 {
		s.WinnerID = make([]byte, len(p.WinnerPubkey))
		copy(s.WinnerID, p.WinnerPubkey)
	}
	if p.CompletedAt > 0 {
		s.WinnerTime = time.Unix(p.CompletedAt, 0)
	}
	return s
}

// SparkReceiver handles incoming spark events from the network.
type SparkReceiver struct {
	store *SparkStore
}

// NewSparkReceiver creates a new spark event receiver.
func NewSparkReceiver(store *SparkStore) *SparkReceiver {
	return &SparkReceiver{store: store}
}

// HandleSparkEvent processes an incoming spark event.
func (r *SparkReceiver) HandleSparkEvent(
	ctx context.Context,
	event *pb.SparkEvent,
	senderPubkey []byte,
) error {
	if err := verifySparkEventSignature(event, senderPubkey); err != nil {
		return err
	}

	switch event.EventType {
	case pb.SparkEventType_SPARK_EVENT_CREATED:
		return r.handleSparkCreated(ctx, event)
	case pb.SparkEventType_SPARK_EVENT_RESPONSE:
		return r.handleSparkResponse(ctx, event)
	case pb.SparkEventType_SPARK_EVENT_COMPLETED:
		return r.handleSparkCompleted(ctx, event)
	case pb.SparkEventType_SPARK_EVENT_EXPIRED:
		return r.handleSparkExpired(ctx, event)
	case pb.SparkEventType_SPARK_EVENT_CANCELLED:
		return r.handleSparkCancelled(ctx, event)
	default:
		return fmt.Errorf("unknown spark event type: %d", event.EventType)
	}
}

// verifySparkEventSignature verifies the event signature.
func verifySparkEventSignature(event *pb.SparkEvent, senderPubkey []byte) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}
	if len(senderPubkey) != ed25519.PublicKeySize {
		return ErrSignatureFailed
	}

	signedData := buildSparkSignedData(event)
	if !ed25519.Verify(senderPubkey, signedData, event.Signature) {
		return ErrSignatureFailed
	}
	return nil
}

// verifySparkResponseSignature verifies a response signature.
func verifySparkResponseSignature(response *pb.SparkResponse) error {
	if len(response.Signature) == 0 {
		return ErrMissingSignature
	}
	if len(response.ResponderPubkey) != ed25519.PublicKeySize {
		return ErrSignatureFailed
	}

	signedData := buildSparkResponseSignedData(response)
	if !ed25519.Verify(response.ResponderPubkey, signedData, response.Signature) {
		return ErrSignatureFailed
	}
	return nil
}

// handleSparkCreated processes a spark creation event.
func (r *SparkReceiver) handleSparkCreated(ctx context.Context, event *pb.SparkEvent) error {
	if event.Spark == nil {
		return ErrInvalidSparkPub
	}

	spark := protoToSpark(event.Spark)
	return r.store.AddSpark(spark)
}

// handleSparkResponse processes a spark response event.
func (r *SparkReceiver) handleSparkResponse(ctx context.Context, event *pb.SparkEvent) error {
	if event.Response == nil {
		return ErrSparkResponseEmpty
	}

	// Verify the response signature.
	if err := verifySparkResponseSignature(event.Response); err != nil {
		return err
	}

	var sparkID [32]byte
	copy(sparkID[:], event.SparkId)

	var waveID [32]byte
	copy(waveID[:], event.Response.WaveId)

	// Use the store's RespondToSpark to handle the response.
	_, err := r.store.RespondToSpark(
		sparkID,
		event.Response.ResponderPubkey,
		waveID,
		nil, // No private key needed for network reception.
	)
	return err
}

// handleSparkCompleted processes a spark completion event.
func (r *SparkReceiver) handleSparkCompleted(ctx context.Context, event *pb.SparkEvent) error {
	var sparkID [32]byte
	copy(sparkID[:], event.SparkId)

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	spark, ok := r.store.sparks[sparkID]
	if !ok {
		return ErrSparkNotFound
	}

	if spark.State != SparkActive {
		return nil // Already completed.
	}

	spark.WinnerID = make([]byte, len(event.WinnerPubkey))
	copy(spark.WinnerID, event.WinnerPubkey)
	spark.WinnerTime = time.Unix(event.Timestamp, 0)
	spark.State = SparkCompleted

	return nil
}

// handleSparkExpired processes a spark expiration event.
func (r *SparkReceiver) handleSparkExpired(ctx context.Context, event *pb.SparkEvent) error {
	var sparkID [32]byte
	copy(sparkID[:], event.SparkId)

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	spark, ok := r.store.sparks[sparkID]
	if !ok {
		return ErrSparkNotFound
	}

	if spark.State != SparkActive {
		return nil
	}

	spark.State = SparkExpired
	return nil
}

// handleSparkCancelled processes a spark cancellation event.
func (r *SparkReceiver) handleSparkCancelled(ctx context.Context, event *pb.SparkEvent) error {
	var sparkID [32]byte
	copy(sparkID[:], event.SparkId)

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	spark, ok := r.store.sparks[sparkID]
	if !ok {
		return ErrSparkNotFound
	}

	if spark.State != SparkActive {
		return nil
	}

	spark.State = SparkCancelled
	return nil
}
