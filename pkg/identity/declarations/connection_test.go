package declarations

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/proto"
)

func TestNewConnectionRequest(t *testing.T) {
	initiator, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	responder, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	conn, err := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)
	if err != nil {
		t.Fatalf("NewConnectionRequest() error: %v", err)
	}

	if conn.IsPending() != true {
		t.Error("New connection request should be pending")
	}
	if conn.IsComplete() {
		t.Error("New connection request should not be complete")
	}
	if err := conn.VerifyInitiator(); err != nil {
		t.Errorf("VerifyInitiator() error: %v", err)
	}
}

func TestNewConnectionRequestNilInitiator(t *testing.T) {
	responder, _ := keys.GenerateKeyPair()

	_, err := NewConnectionRequest(nil, responder.PublicKey, ConnectionFriend)
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

func TestNewConnectionRequestInvalidResponder(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()

	_, err := NewConnectionRequest(initiator, []byte("short"), ConnectionFriend)
	if err != ErrMissingResponder {
		t.Errorf("Expected ErrMissingResponder, got %v", err)
	}
}

func TestNewConnectionRequestSelfConnection(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	_, err := NewConnectionRequest(kp, kp.PublicKey, ConnectionFriend)
	if err != ErrSelfConnection {
		t.Errorf("Expected ErrSelfConnection, got %v", err)
	}
}

func TestAcceptConnection(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)

	err := conn.AcceptConnection(responder)
	if err != nil {
		t.Fatalf("AcceptConnection() error: %v", err)
	}

	if conn.IsPending() {
		t.Error("Accepted connection should not be pending")
	}
	if !conn.IsComplete() {
		t.Error("Accepted connection should be complete")
	}
}

func TestAcceptConnectionWrongResponder(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()
	wrongResponder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)

	err := conn.AcceptConnection(wrongResponder)
	if err != ErrNotAuthorized {
		t.Errorf("Expected ErrNotAuthorized, got %v", err)
	}
}

func TestConnectionVerify(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)
	_ = conn.AcceptConnection(responder)

	err := conn.Verify()
	if err != nil {
		t.Errorf("Verify() error: %v", err)
	}
}

func TestConnectionVerifyIncomplete(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)

	// Should fail - missing responder signature.
	err := conn.Verify()
	if err != ErrMissingSignature {
		t.Errorf("Expected ErrMissingSignature, got %v", err)
	}
}

func TestConnectionMarshalUnmarshal(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionTrusted)
	_ = conn.AcceptConnection(responder)

	data, err := conn.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	decoded, err := UnmarshalConnection(data)
	if err != nil {
		t.Fatalf("UnmarshalConnection() error: %v", err)
	}

	if !bytesEqual(decoded.InitiatorPublicKey, conn.InitiatorPublicKey) {
		t.Error("Initiator public key mismatch")
	}
	if !bytesEqual(decoded.ResponderPublicKey, conn.ResponderPublicKey) {
		t.Error("Responder public key mismatch")
	}
	if decoded.ConnectionType != ConnectionTrusted {
		t.Error("Connection type mismatch")
	}
	if err := decoded.Verify(); err != nil {
		t.Errorf("Decoded Verify() error: %v", err)
	}
}

func TestConnectionWithMutualName(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	// Create connection request with MutualName set before signing.
	conn := &ConnectionDeclaration{
		InitiatorPublicKey: initiator.PublicKey,
		ResponderPublicKey: responder.PublicKey,
		CreatedAt:          1234567890,
		ConnectionType:     ConnectionTrusted,
		MutualName:         "Test Connection",
	}

	// Sign as initiator.
	payload := conn.signingPayload()
	conn.InitiatorSignature = initiator.Sign(payload)

	// Sign as responder.
	conn.ResponderSignature = responder.Sign(payload)

	// Marshal and unmarshal.
	data, err := conn.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	decoded, err := UnmarshalConnection(data)
	if err != nil {
		t.Fatalf("UnmarshalConnection() error: %v", err)
	}

	if decoded.MutualName != "Test Connection" {
		t.Errorf("MutualName = %q, want %q", decoded.MutualName, "Test Connection")
	}
	if err := decoded.Verify(); err != nil {
		t.Errorf("Decoded Verify() error: %v", err)
	}
}

func TestConnectionInvolvesKey(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()
	stranger, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)

	if !conn.InvolvesKey(initiator.PublicKey) {
		t.Error("Connection should involve initiator")
	}
	if !conn.InvolvesKey(responder.PublicKey) {
		t.Error("Connection should involve responder")
	}
	if conn.InvolvesKey(stranger.PublicKey) {
		t.Error("Connection should not involve stranger")
	}
}

func TestConnectionOtherParty(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)

	other := conn.OtherParty(initiator.PublicKey)
	if !bytesEqual(other, responder.PublicKey) {
		t.Error("OtherParty(initiator) should return responder")
	}

	other = conn.OtherParty(responder.PublicKey)
	if !bytesEqual(other, initiator.PublicKey) {
		t.Error("OtherParty(responder) should return initiator")
	}

	stranger, _ := keys.GenerateKeyPair()
	if conn.OtherParty(stranger.PublicKey) != nil {
		t.Error("OtherParty(stranger) should return nil")
	}
}

func TestNewRevocation(t *testing.T) {
	revoker, _ := keys.GenerateKeyPair()
	target, _ := keys.GenerateKeyPair()

	rev, err := NewRevocation(revoker, target.PublicKey, ConnectionFriend, RevocationUserRequest)
	if err != nil {
		t.Fatalf("NewRevocation() error: %v", err)
	}

	if err := rev.Verify(); err != nil {
		t.Errorf("Verify() error: %v", err)
	}
}

func TestNewRevocationNilRevoker(t *testing.T) {
	target, _ := keys.GenerateKeyPair()

	_, err := NewRevocation(nil, target.PublicKey, ConnectionFriend, RevocationUserRequest)
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

func TestNewRevocationInvalidTarget(t *testing.T) {
	revoker, _ := keys.GenerateKeyPair()

	_, err := NewRevocation(revoker, []byte("short"), ConnectionFriend, RevocationUserRequest)
	if err != ErrInvalidPublicKey {
		t.Errorf("Expected ErrInvalidPublicKey, got %v", err)
	}
}

func TestRevocationMarshalUnmarshal(t *testing.T) {
	revoker, _ := keys.GenerateKeyPair()
	target, _ := keys.GenerateKeyPair()

	rev, _ := NewRevocation(revoker, target.PublicKey, ConnectionTrusted, RevocationPolicy)

	data, err := rev.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	decoded, err := UnmarshalRevocation(data)
	if err != nil {
		t.Fatalf("UnmarshalRevocation() error: %v", err)
	}

	if !bytesEqual(decoded.RevokerPublicKey, revoker.PublicKey) {
		t.Error("Revoker public key mismatch")
	}
	if !bytesEqual(decoded.TargetPublicKey, target.PublicKey) {
		t.Error("Target public key mismatch")
	}
	if decoded.ConnectionType != ConnectionTrusted {
		t.Error("Connection type mismatch")
	}
	if decoded.Reason != RevocationPolicy {
		t.Error("Reason mismatch")
	}
	if err := decoded.Verify(); err != nil {
		t.Errorf("Decoded Verify() error: %v", err)
	}
}

func TestRevocationMatchesConnection(t *testing.T) {
	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()
	stranger, _ := keys.GenerateKeyPair()

	conn, _ := NewConnectionRequest(initiator, responder.PublicKey, ConnectionFriend)
	_ = conn.AcceptConnection(responder)

	// Revocation by initiator should match.
	rev1, _ := NewRevocation(initiator, responder.PublicKey, ConnectionFriend, RevocationUserRequest)
	if !rev1.MatchesConnection(conn) {
		t.Error("Initiator revocation should match connection")
	}

	// Revocation by responder should match.
	rev2, _ := NewRevocation(responder, initiator.PublicKey, ConnectionFriend, RevocationUserRequest)
	if !rev2.MatchesConnection(conn) {
		t.Error("Responder revocation should match connection")
	}

	// Revocation by stranger should not match.
	rev3, _ := NewRevocation(stranger, initiator.PublicKey, ConnectionFriend, RevocationUserRequest)
	if rev3.MatchesConnection(conn) {
		t.Error("Stranger revocation should not match connection")
	}

	// Wrong connection type should not match.
	rev4, _ := NewRevocation(initiator, responder.PublicKey, ConnectionTrusted, RevocationUserRequest)
	if rev4.MatchesConnection(conn) {
		t.Error("Wrong type revocation should not match connection")
	}
}

func TestRevocationMatchesConnectionNil(t *testing.T) {
	revoker, _ := keys.GenerateKeyPair()
	target, _ := keys.GenerateKeyPair()

	rev, _ := NewRevocation(revoker, target.PublicKey, ConnectionFriend, RevocationUserRequest)

	if rev.MatchesConnection(nil) {
		t.Error("Should not match nil connection")
	}
}

func TestConnectionTypes(t *testing.T) {
	// Test each connection type works correctly.
	types := []proto.ConnectionType{
		ConnectionFriend,
		ConnectionFollow,
		ConnectionBlock,
		ConnectionTrusted,
	}

	initiator, _ := keys.GenerateKeyPair()
	responder, _ := keys.GenerateKeyPair()

	for _, ct := range types {
		conn, err := NewConnectionRequest(initiator, responder.PublicKey, ct)
		if err != nil {
			t.Errorf("NewConnectionRequest(%v) error: %v", ct, err)
			continue
		}
		if conn.ConnectionType != ct {
			t.Errorf("ConnectionType = %v, want %v", conn.ConnectionType, ct)
		}
	}
}

func TestRevocationReasons(t *testing.T) {
	// Test each revocation reason works correctly.
	reasons := []proto.RevocationReason{
		RevocationUserRequest,
		RevocationInactivity,
		RevocationPolicy,
		RevocationKeyRotation,
	}

	revoker, _ := keys.GenerateKeyPair()
	target, _ := keys.GenerateKeyPair()

	for _, reason := range reasons {
		rev, err := NewRevocation(revoker, target.PublicKey, ConnectionFriend, reason)
		if err != nil {
			t.Errorf("NewRevocation(%v) error: %v", reason, err)
			continue
		}
		if rev.Reason != reason {
			t.Errorf("Reason = %v, want %v", rev.Reason, reason)
		}
	}
}
