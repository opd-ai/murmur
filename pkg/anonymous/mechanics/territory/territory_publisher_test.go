// Package mechanics - Territory publisher tests.
// Per ROADMAP.md line 444, tests for influence claims and territory state changes.
package territory

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// TestNewTerritoryPublisher tests TerritoryPublisher creation.
func TestNewTerritoryPublisher(t *testing.T) {
	pub := NewTerritoryPublisher(nil, nil)
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
	if pub.topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("expected topic %s, got %s", mechanics.TopicAnonymousMechanics, pub.topic)
	}
}

// TestTerritoryPublisher_PublishInfluenceClaim tests influence claim publishing.
func TestTerritoryPublisher_PublishInfluenceClaim(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(mockPub, privKey)

	var specterKey [32]byte
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	ctx := context.Background()
	err := pub.PublishInfluenceClaim(ctx, "territory-1", specterKey, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetTerritoryEvent()
	if event == nil {
		t.Fatal("expected territory event")
	}
	if event.EventType != pb.TerritoryEventType_TERRITORY_EVENT_INFLUENCE {
		t.Errorf("expected INFLUENCE event type, got %v", event.EventType)
	}
	if string(event.TerritoryId) != "territory-1" {
		t.Errorf("expected territory-1, got %s", event.TerritoryId)
	}
	if event.InfluenceAmount != 100 {
		t.Errorf("expected influence 100, got %d", event.InfluenceAmount)
	}
}

// TestTerritoryPublisher_PublishInfluenceClaim_NoPublisher tests without publisher.
func TestTerritoryPublisher_PublishInfluenceClaim_NoPublisher(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(nil, privKey)

	var specterKey [32]byte
	err := pub.PublishInfluenceClaim(context.Background(), "test", specterKey, 10)
	if err != mechanics.ErrPublisherNotSet {
		t.Errorf("expected mechanics.ErrPublisherNotSet, got %v", err)
	}
}

// TestTerritoryPublisher_PublishInfluenceClaim_NoPrivateKey tests without private key.
func TestTerritoryPublisher_PublishInfluenceClaim_NoPrivateKey(t *testing.T) {
	mockPub := &mockPublisher{}
	pub := NewTerritoryPublisher(mockPub, nil)

	var specterKey [32]byte
	err := pub.PublishInfluenceClaim(context.Background(), "test", specterKey, 10)
	if err != mechanics.ErrMissingPrivateKey {
		t.Errorf("expected mechanics.ErrMissingPrivateKey, got %v", err)
	}
}

// TestTerritoryPublisher_PublishControlChange tests control change publishing.
func TestTerritoryPublisher_PublishControlChange(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(mockPub, privKey)

	territory := NewTerritory("territory-2", 100.0, 200.0)
	var controller [32]byte
	copy(controller[:], privKey.Public().(ed25519.PublicKey))
	territory.Controller = &controller
	territory.State = TerritoryControlled

	ctx := context.Background()
	err := pub.PublishControlChange(ctx, territory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetTerritoryEvent()
	if event == nil {
		t.Fatal("expected territory event")
	}
	if event.EventType != pb.TerritoryEventType_TERRITORY_EVENT_CONTROL {
		t.Errorf("expected CONTROL event type, got %v", event.EventType)
	}
	if event.Territory == nil {
		t.Fatal("expected territory in event")
	}
}

// TestTerritoryPublisher_PublishControlChange_NilTerritory tests with nil territory.
func TestTerritoryPublisher_PublishControlChange_NilTerritory(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(mockPub, privKey)

	err := pub.PublishControlChange(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil territory")
	}
}

// TestTerritoryPublisher_PublishTerritoryDrift tests drift event publishing.
func TestTerritoryPublisher_PublishTerritoryDrift(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(mockPub, privKey)

	territory := NewTerritory("territory-3", 150.0, 250.0)

	ctx := context.Background()
	err := pub.PublishTerritoryDrift(ctx, territory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetTerritoryEvent()
	if event == nil {
		t.Fatal("expected territory event")
	}
	if event.EventType != pb.TerritoryEventType_TERRITORY_EVENT_DRIFT {
		t.Errorf("expected DRIFT event type, got %v", event.EventType)
	}
}

// TestTerritoryPublisher_PublishTerritoryDrift_NilTerritory tests with nil territory.
func TestTerritoryPublisher_PublishTerritoryDrift_NilTerritory(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(mockPub, privKey)

	err := pub.PublishTerritoryDrift(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil territory")
	}
}

// TestNewTerritoryReceiver tests TerritoryReceiver creation.
func TestNewTerritoryReceiver(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)
	if receiver == nil {
		t.Fatal("expected non-nil receiver")
	}
	if receiver.territoryStore != store {
		t.Error("store not set correctly")
	}
}

// TestTerritoryReceiver_HandleInfluenceClaim tests influence claim handling.
func TestTerritoryReceiver_HandleInfluenceClaim(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	// Create and publish an influence claim.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewTerritoryPublisher(mockPub, privKey)

	var specterKey [32]byte
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	err := publisher.PublishInfluenceClaim(context.Background(), "territory-test", specterKey, 50)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify territory was created and influence applied.
	territory := store.GetTerritory("territory-test")
	if territory == nil {
		t.Fatal("expected territory to be created")
	}

	influence := territory.GetInfluence(specterKey)
	if influence == 0 {
		t.Error("expected non-zero influence")
	}
}

// TestTerritoryReceiver_HandleControlChange tests control change handling.
func TestTerritoryReceiver_HandleControlChange(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	// Create a territory with a controller.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewTerritoryPublisher(mockPub, privKey)

	territory := NewTerritory("territory-ctrl", 100.0, 200.0)
	var controller [32]byte
	copy(controller[:], privKey.Public().(ed25519.PublicKey))
	territory.Controller = &controller
	territory.State = TerritoryControlled

	err := publisher.PublishControlChange(context.Background(), territory)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify territory was stored with controller.
	stored := store.GetTerritory("territory-ctrl")
	if stored == nil {
		t.Fatal("expected territory to be stored")
	}
	if stored.Controller == nil {
		t.Error("expected controller to be set")
	}
}

// TestTerritoryReceiver_HandleTerritoryDrift tests drift event handling.
func TestTerritoryReceiver_HandleTerritoryDrift(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	// Create a territory.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewTerritoryPublisher(mockPub, privKey)

	territory := NewTerritory("territory-drift", 300.0, 400.0)

	err := publisher.PublishTerritoryDrift(context.Background(), territory)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify territory was stored.
	stored := store.GetTerritory("territory-drift")
	if stored == nil {
		t.Fatal("expected territory to be stored")
	}
}

// TestTerritoryReceiver_HandleMessage_InvalidData tests with invalid data.
func TestTerritoryReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	err := receiver.HandleMessage([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestTerritoryReceiver_HandleMessage_NonTerritoryEvent tests with non-territory event.
func TestTerritoryReceiver_HandleMessage_NonTerritoryEvent(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	// Create a gossip message without a territory event.
	gossipMsg := &pb.GossipMessage{}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != nil {
		t.Errorf("expected nil for non-territory event, got %v", err)
	}
}

// TestTerritoryReceiver_HandleMessage_MissingSignature tests with missing signature.
func TestTerritoryReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	// Create an unsigned territory event.
	event := &pb.TerritoryEvent{
		EventType:   pb.TerritoryEventType_TERRITORY_EVENT_INFLUENCE,
		TerritoryId: []byte("test"),
	}
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_TerritoryEvent{
			TerritoryEvent: event,
		},
	}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != mechanics.ErrMissingSignature {
		t.Errorf("expected mechanics.ErrMissingSignature, got %v", err)
	}
}

// TestTerritoryStore tests TerritoryStore operations.
func TestTerritoryStore(t *testing.T) {
	store := NewTerritoryStore()

	// Test empty store.
	if store.GetTerritory("nonexistent") != nil {
		t.Error("expected nil for nonexistent territory")
	}

	// Test add.
	territory := NewTerritory("test-1", 10.0, 20.0)
	store.AddTerritory(territory)

	retrieved := store.GetTerritory("test-1")
	if retrieved == nil {
		t.Fatal("expected to retrieve territory")
	}
	if retrieved.ID != "test-1" {
		t.Errorf("expected test-1, got %s", retrieved.ID)
	}

	// Test list.
	territory2 := NewTerritory("test-2", 30.0, 40.0)
	store.AddTerritory(territory2)

	list := store.ListTerritories()
	if len(list) != 2 {
		t.Errorf("expected 2 territories, got %d", len(list))
	}

	// Test update.
	territory.CentroidX = 50.0
	store.UpdateTerritory(territory)

	updated := store.GetTerritory("test-1")
	if updated.CentroidX != 50.0 {
		t.Errorf("expected CentroidX 50.0, got %f", updated.CentroidX)
	}

	// Test remove.
	store.RemoveTerritory("test-1")
	if store.GetTerritory("test-1") != nil {
		t.Error("expected nil after removal")
	}
}

// TestTerritoryToProto tests territory to protobuf conversion.
func TestTerritoryToProto(t *testing.T) {
	territory := NewTerritory("proto-test", 100.0, 200.0)

	// Add some influence.
	var specter1 [32]byte
	specter1[0] = 1
	territory.AddInfluence(specter1, InfluenceWaveAmplified, 10.0)
	territory.ComputeInfluence()

	pbTerritory := territoryToProto(territory)
	if pbTerritory == nil {
		t.Fatal("expected non-nil proto")
	}
	if string(pbTerritory.Id) != "proto-test" {
		t.Errorf("expected proto-test, got %s", pbTerritory.Id)
	}
	if len(pbTerritory.Contenders) == 0 {
		t.Error("expected contenders")
	}
}

// TestProtoToTerritory tests protobuf to territory conversion.
func TestProtoToTerritory(t *testing.T) {
	pbTerritory := &pb.Territory{
		Id:   []byte("proto-test"),
		Name: "Proto Test",
	}

	territory := protoToTerritory(pbTerritory)
	if territory == nil {
		t.Fatal("expected non-nil territory")
	}
	if territory.ID != "proto-test" {
		t.Errorf("expected proto-test, got %s", territory.ID)
	}
}

// TestProtoToTerritory_WithController tests conversion with controller.
func TestProtoToTerritory_WithController(t *testing.T) {
	var controllerKey [32]byte
	controllerKey[0] = 42

	pbTerritory := &pb.Territory{
		Id:               []byte("ctrl-test"),
		ControllerPubkey: controllerKey[:],
	}

	territory := protoToTerritory(pbTerritory)
	if territory.Controller == nil {
		t.Fatal("expected controller to be set")
	}
	if (*territory.Controller)[0] != 42 {
		t.Error("controller key mismatch")
	}
	if territory.State != TerritoryControlled {
		t.Errorf("expected TerritoryControlled, got %v", territory.State)
	}
}

// TestProtoToTerritory_Contested tests contested state detection.
func TestProtoToTerritory_Contested(t *testing.T) {
	var specter1, specter2 [32]byte
	specter1[0] = 1
	specter2[0] = 2

	pbTerritory := &pb.Territory{
		Id: []byte("contested-test"),
		Contenders: []*pb.TerritoryContender{
			{SpecterPubkey: specter1[:], Influence: 100},
			{SpecterPubkey: specter2[:], Influence: 95}, // Within 20% threshold.
		},
	}

	territory := protoToTerritory(pbTerritory)
	if territory.State != TerritoryContested {
		t.Errorf("expected TerritoryContested, got %v", territory.State)
	}
}

// TestTerritoryEventSignatureRoundTrip tests signature verification.
func TestTerritoryEventSignatureRoundTrip(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewTerritoryPublisher(mockPub, privKey)

	var specterKey [32]byte
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	// Publish an influence claim.
	err := publisher.PublishInfluenceClaim(context.Background(), "sig-test", specterKey, 25)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Unmarshal and verify signature manually.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetTerritoryEvent()
	if event == nil {
		t.Fatal("expected territory event")
	}
	if len(event.Signature) == 0 {
		t.Fatal("expected signature")
	}

	// Verify the signature.
	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)
	err = receiver.verifyEventSignature(event)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

// BenchmarkTerritoryPublisher_PublishInfluenceClaim benchmarks influence publishing.
func BenchmarkTerritoryPublisher_PublishInfluenceClaim(b *testing.B) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewTerritoryPublisher(mockPub, privKey)

	var specterKey [32]byte
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPub.published = nil
		pub.PublishInfluenceClaim(ctx, "bench-territory", specterKey, 100)
	}
}

// BenchmarkTerritoryReceiver_HandleMessage benchmarks message handling.
func BenchmarkTerritoryReceiver_HandleMessage(b *testing.B) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewTerritoryPublisher(mockPub, privKey)

	var specterKey [32]byte
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	publisher.PublishInfluenceClaim(context.Background(), "bench-territory", specterKey, 50)
	data := mockPub.published[0].data

	store := NewTerritoryStore()
	receiver := NewTerritoryReceiver(store)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		receiver.HandleMessage(data)
	}
}
