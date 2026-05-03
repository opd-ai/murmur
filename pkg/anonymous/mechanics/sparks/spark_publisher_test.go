package sparks

import (
"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

func TestNewSparkmechanics.Publisher(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	sp := NewSparkmechanics.Publisher(pub, privKey)
	if sp == nil {
		t.Fatal("NewSparkmechanics.Publisher returned nil")
	}
	if sp.topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("wrong topic: got %s, want %s", sp.topic, mechanics.TopicAnonymousMechanics)
	}
}

func TestPublishSparkCreated(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	spark := &Spark{
		Type:        SparkEchoRace,
		InitiatorID: make([]byte, 32),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SparkDuration),
		State:       SparkActive,
	}
	rand.Read(spark.ID[:])
	rand.Read(spark.InitiatorID)

	err := sp.PublishSparkCreated(context.Background(), spark)
	if err != nil {
		t.Errorf("PublishSparkCreated failed: %v", err)
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.published))
	}

	// Verify the message can be unmarshaled.
	msg := &pb.GossipMessage{}
	if err := proto.Unmarshal(pub.published[0].data, msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	sparkEvent := msg.GetSparkEvent()
	if sparkEvent == nil {
		t.Fatal("expected spark event")
	}
	if sparkEvent.EventType != pb.SparkEventType_SPARK_EVENT_CREATED {
		t.Errorf("wrong event type: got %v", sparkEvent.EventType)
	}
}

func TestPublishSparkCreated_NilSpark(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	err := sp.PublishSparkCreated(context.Background(), nil)
	if err != ErrInvalidSparkPub {
		t.Errorf("expected ErrInvalidSparkPub, got %v", err)
	}
}

func TestPublishSparkCreated_Nilmechanics.Publisher(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(nil, privKey)

	spark := &Spark{
		Type:  SparkEchoRace,
		State: SparkActive,
	}

	err := sp.PublishSparkCreated(context.Background(), spark)
	if err != mechanics.ErrPublisherNotSet {
		t.Errorf("expected mechanics.ErrPublisherNotSet, got %v", err)
	}
}

func TestPublishSparkResponse(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	var sparkID, waveID [32]byte
	responderKey := make([]byte, 32)
	rand.Read(sparkID[:])
	rand.Read(responderKey)
	rand.Read(waveID[:])

	err := sp.PublishSparkResponse(context.Background(), sparkID, responderKey, waveID)
	if err != nil {
		t.Errorf("PublishSparkResponse failed: %v", err)
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.published))
	}

	msg := &pb.GossipMessage{}
	if err := proto.Unmarshal(pub.published[0].data, msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	sparkEvent := msg.GetSparkEvent()
	if sparkEvent == nil {
		t.Fatal("expected spark event")
	}
	if sparkEvent.EventType != pb.SparkEventType_SPARK_EVENT_RESPONSE {
		t.Errorf("wrong event type: got %v", sparkEvent.EventType)
	}
	if sparkEvent.Response == nil {
		t.Fatal("expected response in event")
	}
}

func TestPublishSparkCompleted(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	var sparkID [32]byte
	winnerKey := make([]byte, 32)
	rand.Read(sparkID[:])
	rand.Read(winnerKey)

	err := sp.PublishSparkCompleted(context.Background(), sparkID, winnerKey)
	if err != nil {
		t.Errorf("PublishSparkCompleted failed: %v", err)
	}

	msg := &pb.GossipMessage{}
	if err := proto.Unmarshal(pub.published[0].data, msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	sparkEvent := msg.GetSparkEvent()
	if sparkEvent.EventType != pb.SparkEventType_SPARK_EVENT_COMPLETED {
		t.Errorf("wrong event type: got %v", sparkEvent.EventType)
	}
}

func TestPublishSparkExpired(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	var sparkID [32]byte
	rand.Read(sparkID[:])

	err := sp.PublishSparkExpired(context.Background(), sparkID)
	if err != nil {
		t.Errorf("PublishSparkExpired failed: %v", err)
	}

	msg := &pb.GossipMessage{}
	if err := proto.Unmarshal(pub.published[0].data, msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	sparkEvent := msg.GetSparkEvent()
	if sparkEvent.EventType != pb.SparkEventType_SPARK_EVENT_EXPIRED {
		t.Errorf("wrong event type: got %v", sparkEvent.EventType)
	}
}

func TestPublishSparkCancelled(t *testing.T) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	var sparkID [32]byte
	rand.Read(sparkID[:])

	err := sp.PublishSparkCancelled(context.Background(), sparkID)
	if err != nil {
		t.Errorf("PublishSparkCancelled failed: %v", err)
	}

	msg := &pb.GossipMessage{}
	if err := proto.Unmarshal(pub.published[0].data, msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	sparkEvent := msg.GetSparkEvent()
	if sparkEvent.EventType != pb.SparkEventType_SPARK_EVENT_CANCELLED {
		t.Errorf("wrong event type: got %v", sparkEvent.EventType)
	}
}

func TestSparkToProto(t *testing.T) {
	spark := &Spark{
		Type:        SparkEchoRace,
		InitiatorID: make([]byte, 32),
		Prompt:      "test prompt",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SparkDuration),
		State:       SparkActive,
	}
	rand.Read(spark.ID[:])
	rand.Read(spark.InitiatorID)

	pbSpark := sparkToProto(spark)
	if pbSpark == nil {
		t.Fatal("sparkToProto returned nil")
	}
	if pbSpark.SparkType != pb.SparkType(SparkEchoRace) {
		t.Errorf("wrong type: got %v", pbSpark.SparkType)
	}
	if pbSpark.Prompt != "test prompt" {
		t.Errorf("wrong prompt: got %s", pbSpark.Prompt)
	}
}

func TestProtoToSpark(t *testing.T) {
	initiator := make([]byte, 32)
	rand.Read(initiator)

	pbSpark := &pb.SurfaceSpark{
		Id:              make([]byte, 32),
		SparkType:       pb.SparkType_SPARK_TYPE_WAVE_RELAY,
		InitiatorPubkey: initiator,
		Prompt:          "relay challenge",
		CreatedAt:       time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
		State:           pb.SparkState_SPARK_STATE_ACTIVE,
	}
	rand.Read(pbSpark.Id)

	spark := protoToSpark(pbSpark)
	if spark == nil {
		t.Fatal("protoToSpark returned nil")
	}
	if spark.Type != SparkWaveRelay {
		t.Errorf("wrong type: got %v", spark.Type)
	}
	if spark.Prompt != "relay challenge" {
		t.Errorf("wrong prompt: got %s", spark.Prompt)
	}
}

func TestSparkReceiver_HandleSparkEvent_Created(t *testing.T) {
	store := NewSparkStore()
	receiver := NewSparkReceiver(store)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	initiator := make([]byte, 32)
	rand.Read(initiator)

	pbSpark := &pb.SurfaceSpark{
		Id:              make([]byte, 32),
		SparkType:       pb.SparkType_SPARK_TYPE_ECHO_RACE,
		InitiatorPubkey: initiator,
		CreatedAt:       time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
		State:           pb.SparkState_SPARK_STATE_ACTIVE,
	}
	rand.Read(pbSpark.Id)

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_CREATED,
		Spark:     pbSpark,
		SparkId:   pbSpark.Id,
		Timestamp: time.Now().Unix(),
	}

	// Sign the event.
	signedData := buildSparkSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandleSparkEvent(context.Background(), event, pubKey)
	if err != nil {
		t.Errorf("HandleSparkEvent failed: %v", err)
	}

	// Verify spark was stored.
	var sparkID [32]byte
	copy(sparkID[:], pbSpark.Id)
	_, err = store.GetSpark(sparkID)
	if err != nil {
		t.Errorf("spark not found in store: %v", err)
	}
}

func TestSparkReceiver_HandleSparkEvent_InvalidSignature(t *testing.T) {
	store := NewSparkStore()
	receiver := NewSparkReceiver(store)

	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)
	_, otherPrivKey, _ := ed25519.GenerateKey(rand.Reader)

	pbSpark := &pb.SurfaceSpark{
		Id:              make([]byte, 32),
		SparkType:       pb.SparkType_SPARK_TYPE_ECHO_RACE,
		InitiatorPubkey: make([]byte, 32),
		CreatedAt:       time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
		State:           pb.SparkState_SPARK_STATE_ACTIVE,
	}
	rand.Read(pbSpark.Id)

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_CREATED,
		Spark:     pbSpark,
		SparkId:   pbSpark.Id,
		Timestamp: time.Now().Unix(),
	}

	// Sign with wrong key.
	signedData := buildSparkSignedData(event)
	event.Signature = ed25519.Sign(otherPrivKey, signedData)

	err := receiver.HandleSparkEvent(context.Background(), event, pubKey)
	if err != mechanics.ErrSignatureFailed {
		t.Errorf("expected mechanics.ErrSignatureFailed, got %v", err)
	}
}

func TestSparkReceiver_HandleSparkEvent_Expired(t *testing.T) {
	store := NewSparkStore()
	receiver := NewSparkReceiver(store)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	// Create a spark first.
	initiator := make([]byte, 32)
	rand.Read(initiator)
	spark, _ := store.CreateSpark(SparkEchoRace, initiator, "", nil)

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_EXPIRED,
		SparkId:   spark.ID[:],
		Timestamp: time.Now().Unix(),
	}

	signedData := buildSparkSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandleSparkEvent(context.Background(), event, pubKey)
	if err != nil {
		t.Errorf("HandleSparkEvent failed: %v", err)
	}

	// Verify spark state changed.
	stored, _ := store.GetSpark(spark.ID)
	if stored.State != SparkExpired {
		t.Errorf("expected SparkExpired, got %v", stored.State)
	}
}

func TestSparkReceiver_HandleSparkEvent_Cancelled(t *testing.T) {
	store := NewSparkStore()
	receiver := NewSparkReceiver(store)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	// Create a spark first.
	initiator := make([]byte, 32)
	rand.Read(initiator)
	spark, _ := store.CreateSpark(SparkEchoRace, initiator, "", nil)

	event := &pb.SparkEvent{
		EventType: pb.SparkEventType_SPARK_EVENT_CANCELLED,
		SparkId:   spark.ID[:],
		Timestamp: time.Now().Unix(),
	}

	signedData := buildSparkSignedData(event)
	event.Signature = ed25519.Sign(privKey, signedData)

	err := receiver.HandleSparkEvent(context.Background(), event, pubKey)
	if err != nil {
		t.Errorf("HandleSparkEvent failed: %v", err)
	}

	// Verify spark state changed.
	stored, _ := store.GetSpark(spark.ID)
	if stored.State != SparkCancelled {
		t.Errorf("expected SparkCancelled, got %v", stored.State)
	}
}

func TestSparkAddSpark(t *testing.T) {
	store := NewSparkStore()

	spark := &Spark{
		Type:        SparkEchoRace,
		InitiatorID: make([]byte, 32),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SparkDuration),
		State:       SparkActive,
	}
	rand.Read(spark.ID[:])
	rand.Read(spark.InitiatorID)

	err := store.AddSpark(spark)
	if err != nil {
		t.Errorf("AddSpark failed: %v", err)
	}

	// Verify retrieval.
	retrieved, err := store.GetSpark(spark.ID)
	if err != nil {
		t.Errorf("GetSpark failed: %v", err)
	}
	if retrieved.Type != SparkEchoRace {
		t.Errorf("wrong type: got %v", retrieved.Type)
	}
}

func TestSparkAddSpark_Idempotent(t *testing.T) {
	store := NewSparkStore()

	spark := &Spark{
		Type:        SparkEchoRace,
		InitiatorID: make([]byte, 32),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SparkDuration),
		State:       SparkActive,
	}
	rand.Read(spark.ID[:])
	rand.Read(spark.InitiatorID)

	err1 := store.AddSpark(spark)
	err2 := store.AddSpark(spark)

	if err1 != nil || err2 != nil {
		t.Errorf("AddSpark should be idempotent: err1=%v, err2=%v", err1, err2)
	}

	// Should only have one spark.
	active := store.GetActiveSparks()
	if len(active) != 1 {
		t.Errorf("expected 1 spark, got %d", len(active))
	}
}

func BenchmarkSparkPublisher_PublishSparkCreated(b *testing.B) {
	pub := &mockmechanics.Publisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sp := NewSparkmechanics.Publisher(pub, privKey)

	spark := &Spark{
		Type:        SparkEchoRace,
		InitiatorID: make([]byte, 32),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SparkDuration),
		State:       SparkActive,
	}
	rand.Read(spark.ID[:])
	rand.Read(spark.InitiatorID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sp.PublishSparkCreated(context.Background(), spark)
	}
}

func BenchmarkSparkReceiver_HandleSparkEvent(b *testing.B) {
	store := NewSparkStore()
	receiver := NewSparkReceiver(store)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	initiator := make([]byte, 32)
	rand.Read(initiator)

	events := make([]*pb.SparkEvent, b.N)
	for i := 0; i < b.N; i++ {
		pbSpark := &pb.SurfaceSpark{
			Id:              make([]byte, 32),
			SparkType:       pb.SparkType_SPARK_TYPE_ECHO_RACE,
			InitiatorPubkey: initiator,
			CreatedAt:       time.Now().Unix(),
			ExpiresAt:       time.Now().Add(time.Hour).Unix(),
			State:           pb.SparkState_SPARK_STATE_ACTIVE,
		}
		rand.Read(pbSpark.Id)

		event := &pb.SparkEvent{
			EventType: pb.SparkEventType_SPARK_EVENT_CREATED,
			Spark:     pbSpark,
			SparkId:   pbSpark.Id,
			Timestamp: time.Now().Unix(),
		}

		signedData := buildSparkSignedData(event)
		event.Signature = ed25519.Sign(privKey, signedData)
		events[i] = event
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		receiver.HandleSparkEvent(context.Background(), events[i], pubKey)
	}
}
