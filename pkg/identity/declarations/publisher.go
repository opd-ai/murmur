// Package declarations provides identity declaration creation and parsing.
// This file implements GossipSub publication for identity declarations.
// Per DESIGN_DOCUMENT.md, identity declarations are published on /murmur/identity/1.
package declarations

import (
	"context"
	"errors"
	"fmt"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/proto"
	pb "google.golang.org/protobuf/proto"
)

// Publication errors.
var (
	ErrPublisherNotSet    = errors.New("publisher not set")
	ErrDeclarationInvalid = errors.New("declaration validation failed")
)

// Publisher provides an interface for publishing to GossipSub.
// This abstracts the networking layer from the declarations package.
type Publisher interface {
	// Publish sends a message to the specified topic.
	Publish(ctx context.Context, topicName string, data []byte) error
}

// IdentityPublisher handles publishing identity-related declarations.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, identity declarations go to /murmur/identity/1.
type IdentityPublisher struct {
	publisher Publisher
	topic     string
}

// TopicIdentity is the GossipSub topic for identity declarations.
const TopicIdentity = "/murmur/identity/1"

// NewIdentityPublisher creates a new identity publisher.
func NewIdentityPublisher(pub Publisher) *IdentityPublisher {
	return &IdentityPublisher{
		publisher: pub,
		topic:     TopicIdentity,
	}
}

// PublishDeclaration publishes an identity declaration to GossipSub.
// The declaration must be signed and validated before publishing.
func (p *IdentityPublisher) PublishDeclaration(ctx context.Context, decl *Declaration) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if err := decl.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationInvalid, err)
	}

	// Wrap in GossipMessage.
	gossipMsg := &proto.GossipMessage{
		Content: &proto.GossipMessage_IdentityDeclaration{
			IdentityDeclaration: &proto.IdentityDeclaration{
				PublicKey:   decl.PublicKey,
				DisplayName: decl.DisplayName,
				Bio:         decl.Bio,
				CreatedAt:   decl.Timestamp,
				Version:     decl.Version,
				Signature:   decl.Signature,
				SigilPng:    decl.SigilPNG,
				PrivacyMode: modeToProto(decl.PrivacyMode),
			},
		},
	}

	data, err := pb.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling declaration for publish: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// PublishConnection publishes a connection declaration to GossipSub.
// The connection must be complete (both signatures present) before publishing.
func (p *IdentityPublisher) PublishConnection(ctx context.Context, conn *ConnectionDeclaration) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if err := conn.Verify(); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationInvalid, err)
	}

	// Wrap in GossipMessage.
	gossipMsg := &proto.GossipMessage{
		Content: &proto.GossipMessage_ConnectionDeclaration{
			ConnectionDeclaration: &proto.ConnectionDeclaration{
				InitiatorPublicKey: conn.InitiatorPublicKey,
				ResponderPublicKey: conn.ResponderPublicKey,
				InitiatorSignature: conn.InitiatorSignature,
				ResponderSignature: conn.ResponderSignature,
				CreatedAt:          conn.CreatedAt,
				ConnectionType:     conn.ConnectionType,
				MutualName:         conn.MutualName,
			},
		},
	}

	data, err := pb.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling connection for publish: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// PublishRevocation publishes a connection revocation to GossipSub.
func (p *IdentityPublisher) PublishRevocation(ctx context.Context, rev *ConnectionRevocation) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if err := rev.Verify(); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationInvalid, err)
	}

	// Wrap in GossipMessage.
	gossipMsg := &proto.GossipMessage{
		Content: &proto.GossipMessage_ConnectionRevocation{
			ConnectionRevocation: &proto.ConnectionRevocation{
				RevokerPublicKey: rev.RevokerPublicKey,
				TargetPublicKey:  rev.TargetPublicKey,
				Signature:        rev.Signature,
				RevokedAt:        rev.RevokedAt,
				ConnectionType:   rev.ConnectionType,
				Reason:           rev.Reason,
			},
		},
	}

	data, err := pb.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling revocation for publish: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// PublishProfileUpdate publishes a profile update to GossipSub.
func (p *IdentityPublisher) PublishProfileUpdate(ctx context.Context, update *ProfileUpdate) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if err := update.Verify(); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationInvalid, err)
	}

	// Wrap in GossipMessage.
	gossipMsg := &proto.GossipMessage{
		Content: &proto.GossipMessage_ProfileUpdate{
			ProfileUpdate: &proto.ProfileUpdate{
				PublicKey:   update.PublicKey,
				DisplayName: update.DisplayName,
				Bio:         update.Bio,
				SigilPng:    update.SigilPNG,
				PrivacyMode: modeToProto(update.PrivacyMode),
				UpdatedAt:   update.UpdatedAt,
				Version:     update.Version,
				Signature:   update.Signature,
			},
		},
	}

	data, err := pb.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling profile update for publish: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// PublishSpecter publishes a Specter declaration to GossipSub.
// Note: Specter declarations may use a different topic in the future.
func (p *IdentityPublisher) PublishSpecter(ctx context.Context, spec *SpecterDeclaration, ed25519PubKey []byte) error {
	if p.publisher == nil {
		return ErrPublisherNotSet
	}
	if err := spec.VerifyPoW(); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationInvalid, err)
	}

	// Wrap in GossipMessage.
	gossipMsg := &proto.GossipMessage{
		Content: &proto.GossipMessage_SpecterDeclaration{
			SpecterDeclaration: &proto.SpecterDeclaration{
				PublicKey:        spec.PublicKey,
				Pseudonym:        spec.Pseudonym,
				SigilPng:         spec.SigilPNG,
				CreatedAt:        spec.CreatedAt,
				PowNonce:         spec.PoWNonce,
				Signature:        spec.Signature,
				InitialResonance: spec.InitialResonance,
			},
		},
	}

	data, err := pb.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("marshaling specter for publish: %w", err)
	}

	return p.publisher.Publish(ctx, p.topic, data)
}

// CreateAndPublishDeclaration is a convenience method to create, sign, and publish.
func (p *IdentityPublisher) CreateAndPublishDeclaration(ctx context.Context, kp *keys.KeyPair, displayName string) error {
	decl, err := New(kp, displayName)
	if err != nil {
		return fmt.Errorf("creating declaration: %w", err)
	}

	if err := decl.Sign(kp); err != nil {
		return fmt.Errorf("signing declaration: %w", err)
	}

	return p.PublishDeclaration(ctx, decl)
}
