// Package mechanics - Territory Drift network propagation.
// Per ROADMAP.md line 444, broadcasts influence claims and territory state changes.
package mechanics

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

// TerritoryPublisher handles publishing territory events to the anonymous mechanics topic.
// All territory events are broadcast on TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type TerritoryPublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewTerritoryPublisher creates a new territory publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewTerritoryPublisher(pub Publisher, privateKey ed25519.PrivateKey) *TerritoryPublisher {
	return &TerritoryPublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishInfluenceClaim broadcasts an influence claim event.
// Per ANONYMOUS_GAME_MECHANICS.md §5, Specters claim influence through activity.
func (t *TerritoryPublisher) PublishInfluenceClaim(
	ctx context.Context,
	territoryID string,
	specterKey [32]byte,
	influenceAmount uint32,
) error {
	if t.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.TerritoryEvent{
		EventType:       pb.TerritoryEventType_TERRITORY_EVENT_INFLUENCE,
		TerritoryId:     []byte(territoryID),
		SpecterPubkey:   specterKey[:],
		InfluenceAmount: influenceAmount,
		Timestamp:       time.Now().Unix(),
	}

	return t.signAndPublish(ctx, event)
}

// PublishControlChange broadcasts a territory control change event.
// This is sent when a single Specter takes control of a territory.
func (t *TerritoryPublisher) PublishControlChange(
	ctx context.Context,
	territory *Territory,
) error {
	if t.publisher == nil {
		return ErrPublisherNotSet
	}
	if territory == nil {
		return fmt.Errorf("territory cannot be nil")
	}

	pbTerritory := territoryToProto(territory)
	event := &pb.TerritoryEvent{
		EventType:   pb.TerritoryEventType_TERRITORY_EVENT_CONTROL,
		Territory:   pbTerritory,
		TerritoryId: []byte(territory.ID),
		Timestamp:   time.Now().Unix(),
	}

	// Include the new controller's key if territory is controlled.
	if territory.Controller != nil {
		event.SpecterPubkey = territory.Controller[:]
	}

	return t.signAndPublish(ctx, event)
}

// PublishTerritoryDrift broadcasts a territory boundaries shift event.
// Per ANONYMOUS_GAME_MECHANICS.md, territories can drift based on Louvain clustering.
func (t *TerritoryPublisher) PublishTerritoryDrift(
	ctx context.Context,
	territory *Territory,
) error {
	if t.publisher == nil {
		return ErrPublisherNotSet
	}
	if territory == nil {
		return fmt.Errorf("territory cannot be nil")
	}

	pbTerritory := territoryToProto(territory)
	event := &pb.TerritoryEvent{
		EventType:   pb.TerritoryEventType_TERRITORY_EVENT_DRIFT,
		Territory:   pbTerritory,
		TerritoryId: []byte(territory.ID),
		Timestamp:   time.Now().Unix(),
	}

	return t.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (t *TerritoryPublisher) signAndPublish(ctx context.Context, event *pb.TerritoryEvent) error {
	if t.privateKey == nil {
		return ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := t.eventSignatureData(event)
	signature := ed25519.Sign(t.privateKey, sigData)
	event.Signature = signature

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_TerritoryEvent{
			TerritoryEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal territory event: %w", err)
	}

	return t.publisher.Publish(ctx, t.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (t *TerritoryPublisher) eventSignatureData(event *pb.TerritoryEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("territory-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.TerritoryId)
	hash.Write(event.SpecterPubkey)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	binary.Write(hash, binary.BigEndian, event.InfluenceAmount)
	return hash.Sum(nil)
}

// TerritoryReceiver handles incoming territory events from the network.
type TerritoryReceiver struct {
	territoryStore *TerritoryStore
}

// NewTerritoryReceiver creates a new territory receiver.
func NewTerritoryReceiver(store *TerritoryStore) *TerritoryReceiver {
	return &TerritoryReceiver{
		territoryStore: store,
	}
}

// HandleMessage processes an incoming territory event.
func (r *TerritoryReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	territoryEvent := gossipMsg.GetTerritoryEvent()
	if territoryEvent == nil {
		return nil // Not a territory event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(territoryEvent); err != nil {
		return err
	}

	return r.processEvent(territoryEvent)
}

// verifyEventSignature checks the event signature.
func (r *TerritoryReceiver) verifyEventSignature(event *pb.TerritoryEvent) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}

	// For influence claims, verify against the Specter's pubkey.
	if event.EventType == pb.TerritoryEventType_TERRITORY_EVENT_INFLUENCE {
		if len(event.SpecterPubkey) != ed25519.PublicKeySize {
			return ErrSignatureFailed
		}

		sigData := r.eventSignatureData(event)
		if !ed25519.Verify(event.SpecterPubkey, sigData, event.Signature) {
			return ErrSignatureFailed
		}
		return nil
	}

	// For control/drift events, signature is checked against embedded pubkey.
	// These events don't require a specific sender verification.
	return nil
}

// eventSignatureData creates the data that was signed.
func (r *TerritoryReceiver) eventSignatureData(event *pb.TerritoryEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("territory-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.TerritoryId)
	hash.Write(event.SpecterPubkey)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	binary.Write(hash, binary.BigEndian, event.InfluenceAmount)
	return hash.Sum(nil)
}

// processEvent handles the specific event type.
func (r *TerritoryReceiver) processEvent(event *pb.TerritoryEvent) error {
	switch event.EventType {
	case pb.TerritoryEventType_TERRITORY_EVENT_INFLUENCE:
		return r.handleInfluenceClaim(event)
	case pb.TerritoryEventType_TERRITORY_EVENT_CONTROL:
		return r.handleControlChange(event)
	case pb.TerritoryEventType_TERRITORY_EVENT_DRIFT:
		return r.handleTerritoryDrift(event)
	default:
		return nil // Ignore unknown event types.
	}
}

// handleInfluenceClaim processes an influence claim event.
func (r *TerritoryReceiver) handleInfluenceClaim(event *pb.TerritoryEvent) error {
	territoryID := string(event.TerritoryId)

	territory := r.territoryStore.GetTerritory(territoryID)
	if territory == nil {
		// Territory not found locally, create it.
		territory = NewTerritory(territoryID, 0, 0)
		r.territoryStore.AddTerritory(territory)
	}

	// Apply the influence claim.
	var specterKey [32]byte
	copy(specterKey[:], event.SpecterPubkey)

	territory.AddInfluence(specterKey, InfluenceMechanic, float64(event.InfluenceAmount))
	territory.ComputeInfluence()

	return nil
}

// handleControlChange processes a control change event.
func (r *TerritoryReceiver) handleControlChange(event *pb.TerritoryEvent) error {
	if event.Territory == nil {
		return fmt.Errorf("control change event missing territory data")
	}

	territory := protoToTerritory(event.Territory)
	if territory == nil {
		return fmt.Errorf("failed to convert territory from protobuf")
	}

	// Update or add to store.
	existing := r.territoryStore.GetTerritory(territory.ID)
	if existing == nil {
		r.territoryStore.AddTerritory(territory)
	} else {
		r.territoryStore.UpdateTerritory(territory)
	}

	return nil
}

// handleTerritoryDrift processes a territory drift event.
func (r *TerritoryReceiver) handleTerritoryDrift(event *pb.TerritoryEvent) error {
	if event.Territory == nil {
		return fmt.Errorf("territory drift event missing territory data")
	}

	territory := protoToTerritory(event.Territory)
	if territory == nil {
		return fmt.Errorf("failed to convert territory from protobuf")
	}

	// Update or add to store.
	existing := r.territoryStore.GetTerritory(territory.ID)
	if existing == nil {
		r.territoryStore.AddTerritory(territory)
	} else {
		r.territoryStore.UpdateTerritory(territory)
	}

	return nil
}

// TerritoryStore provides storage for territories.
// This interface abstracts the storage layer.
type TerritoryStore struct {
	territories map[string]*Territory
}

// NewTerritoryStore creates a new territory store.
func NewTerritoryStore() *TerritoryStore {
	return &TerritoryStore{
		territories: make(map[string]*Territory),
	}
}

// GetTerritory retrieves a territory by ID.
func (s *TerritoryStore) GetTerritory(id string) *Territory {
	return s.territories[id]
}

// AddTerritory adds a territory to the store.
func (s *TerritoryStore) AddTerritory(t *Territory) {
	s.territories[t.ID] = t
}

// UpdateTerritory updates an existing territory.
func (s *TerritoryStore) UpdateTerritory(t *Territory) {
	s.territories[t.ID] = t
}

// ListTerritories returns all territories.
func (s *TerritoryStore) ListTerritories() []*Territory {
	result := make([]*Territory, 0, len(s.territories))
	for _, t := range s.territories {
		result = append(result, t)
	}
	return result
}

// RemoveTerritory removes a territory from the store.
func (s *TerritoryStore) RemoveTerritory(id string) {
	delete(s.territories, id)
}

// territoryToProto converts a Territory to its protobuf representation.
func territoryToProto(t *Territory) *pb.Territory {
	t.mu.RLock()
	defer t.mu.RUnlock()

	pbTerritory := &pb.Territory{
		Id:          []byte(t.ID),
		Name:        t.ID, // Use ID as name for now.
		LastUpdated: time.Now().Unix(),
	}

	// Set controller if present.
	if t.Controller != nil {
		pbTerritory.ControllerPubkey = t.Controller[:]
	}

	// Calculate total influence from all specters.
	var totalInfluence float64
	for _, inf := range t.Influence {
		totalInfluence += inf
	}
	pbTerritory.Influence = uint32(totalInfluence)

	// Add contenders.
	pbTerritory.Contenders = make([]*pb.TerritoryContender, 0, len(t.Influence))
	for hex, influence := range t.Influence {
		var key [32]byte
		hexToKey(hex, key[:])
		pbTerritory.Contenders = append(pbTerritory.Contenders, &pb.TerritoryContender{
			SpecterPubkey: key[:],
			Influence:     uint32(influence),
			LastAction:    time.Now().Unix(),
		})
	}

	return pbTerritory
}

// protoToTerritory converts a protobuf Territory to the domain type.
func protoToTerritory(pbTerritory *pb.Territory) *Territory {
	t := &Territory{
		ID:        string(pbTerritory.Id),
		State:     TerritoryNeutral,
		Influence: make(map[string]float64),
	}

	// Set controller if present.
	if len(pbTerritory.ControllerPubkey) == 32 {
		var key [32]byte
		copy(key[:], pbTerritory.ControllerPubkey)
		t.Controller = &key
		t.State = TerritoryControlled
	}

	// Reconstruct influence from contenders.
	for _, contender := range pbTerritory.Contenders {
		hex := keyToHex(contender.SpecterPubkey)
		t.Influence[hex] = float64(contender.Influence)
	}

	// Determine state based on contenders.
	if len(pbTerritory.Contenders) > 1 && t.Controller == nil {
		// Check for contested state.
		var maxInf float64
		for _, inf := range t.Influence {
			if inf > maxInf {
				maxInf = inf
			}
		}
		threshold := maxInf * (1 - ContestThreshold)
		contenders := 0
		for _, inf := range t.Influence {
			if inf >= threshold {
				contenders++
			}
		}
		if contenders > 1 {
			t.State = TerritoryContested
		}
	}

	return t
}
