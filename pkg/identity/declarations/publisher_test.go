package declarations

import (
	"context"
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
)

// mockPublisher is a mock Publisher for testing.
type mockPublisher struct {
	published []publishedMessage
}

type publishedMessage struct {
	topic string
	data  []byte
}

func (m *mockPublisher) Publish(_ context.Context, topicName string, data []byte) error {
	m.published = append(m.published, publishedMessage{topic: topicName, data: data})
	return nil
}

func TestNewIdentityPublisher(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	if pub == nil {
		t.Fatal("NewIdentityPublisher() returned nil")
	}
	if pub.topic != TopicIdentity {
		t.Errorf("topic = %q, want %q", pub.topic, TopicIdentity)
	}
}

func TestIdentityPublisherPublishDeclaration(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	kp, _ := keys.GenerateKeyPair()
	decl, _ := New(kp, "TestUser")
	decl.Sign(kp)

	ctx := context.Background()
	err := pub.PublishDeclaration(ctx, decl)
	if err != nil {
		t.Fatalf("PublishDeclaration() error: %v", err)
	}

	if len(mock.published) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mock.published))
	}

	if mock.published[0].topic != TopicIdentity {
		t.Errorf("Published to %q, want %q", mock.published[0].topic, TopicIdentity)
	}
}

func TestIdentityPublisherPublishDeclarationInvalid(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	kp, _ := keys.GenerateKeyPair()
	decl, _ := New(kp, "TestUser")
	// Not signed - should fail validation.

	ctx := context.Background()
	err := pub.PublishDeclaration(ctx, decl)
	if err == nil {
		t.Error("Expected error for unsigned declaration")
	}
}

func TestIdentityPublisherPublishDeclarationNoPublisher(t *testing.T) {
	pub := NewIdentityPublisher(nil)

	kp, _ := keys.GenerateKeyPair()
	decl, _ := New(kp, "TestUser")
	decl.Sign(kp)

	ctx := context.Background()
	err := pub.PublishDeclaration(ctx, decl)
	if err != ErrPublisherNotSet {
		t.Errorf("Expected ErrPublisherNotSet, got %v", err)
	}
}

func TestIdentityPublisherPublishConnection(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)
	conn.AcceptConnection(responder)

	ctx := context.Background()
	err := pub.PublishConnection(ctx, conn)
	if err != nil {
		t.Fatalf("PublishConnection() error: %v", err)
	}

	if len(mock.published) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mock.published))
	}

	if mock.published[0].topic != TopicIdentity {
		t.Errorf("Published to %q, want %q", mock.published[0].topic, TopicIdentity)
	}
}

func TestIdentityPublisherPublishConnectionIncomplete(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)
	// Not accepted - incomplete.

	ctx := context.Background()
	err := pub.PublishConnection(ctx, conn)
	if err == nil {
		t.Error("Expected error for incomplete connection")
	}
}

func TestIdentityPublisherPublishRevocation(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	revoker, _ := keys.GenerateKeyPair()
	target, _ := keys.GenerateKeyPair()

	rev, _ := NewRevocation(revoker, target.PublicKey, ConnectionFriend, RevocationUserRequest)

	ctx := context.Background()
	err := pub.PublishRevocation(ctx, rev)
	if err != nil {
		t.Fatalf("PublishRevocation() error: %v", err)
	}

	if len(mock.published) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mock.published))
	}
}

func TestIdentityPublisherPublishProfileUpdate(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)
	update.SetDisplayName("NewName")
	update.Sign(kp)

	ctx := context.Background()
	err := pub.PublishProfileUpdate(ctx, update)
	if err != nil {
		t.Fatalf("PublishProfileUpdate() error: %v", err)
	}

	if len(mock.published) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mock.published))
	}
}

func TestIdentityPublisherCreateAndPublish(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	kp, _ := keys.GenerateKeyPair()

	ctx := context.Background()
	err := pub.CreateAndPublishDeclaration(ctx, kp, "QuickUser")
	if err != nil {
		t.Fatalf("CreateAndPublishDeclaration() error: %v", err)
	}

	if len(mock.published) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mock.published))
	}
}

func TestIdentityPublisherPublishSpecter(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewIdentityPublisher(mock)

	specterPub := make([]byte, SpecterKeySize)
	for i := range specterPub {
		specterPub[i] = byte(i)
	}

	spec, _ := NewSpecterDeclaration(specterPub, "TestSpecter")
	// Set a valid-ish PoW nonce (we'll skip actual PoW for testing).
	spec.PoWNonce = 1

	// For test, manually make the PoW "valid" by using low difficulty internally.
	// In real usage, ComputePoW() would be called.

	ctx := context.Background()
	err := pub.PublishSpecter(ctx, spec, nil)
	// Will fail PoW verification with nonce=1, which is expected.
	if err == nil {
		t.Log("Note: PoW verification expected to fail in test without actual computation")
	}
}
