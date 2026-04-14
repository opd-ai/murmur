// Package specters provides Anonymous Layer identity management.
// This file tests Specter Connection functionality.
package specters

import (
	"testing"

	"github.com/opd-ai/murmur/proto"
)

func TestNewSpecterConnectionRequest(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()

	// Mark both as announced.
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, err := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
	if err != nil {
		t.Fatalf("NewSpecterConnectionRequest failed: %v", err)
	}

	// Should be pending.
	if !conn.IsPending() {
		t.Error("new connection request should be pending")
	}
	if conn.IsComplete() {
		t.Error("new connection request should not be complete")
	}

	// Initiator should match.
	if conn.InitiatorPublicKey != s1.PublicKey {
		t.Error("initiator public key mismatch")
	}

	// Responder should match.
	if conn.ResponderPublicKey != s2.PublicKey {
		t.Error("responder public key mismatch")
	}

	// Should have shared secret hash.
	if len(conn.SharedSecretHash) == 0 {
		t.Error("should have shared secret hash")
	}
}

func TestSpecterConnectionRequestNotAnnounced(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()

	// s1 is not announced.
	_, err := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
	if err != ErrSpecterNotAnnounced {
		t.Errorf("expected ErrSpecterNotAnnounced, got %v", err)
	}
}

func TestSpecterConnectionSelfConnect(t *testing.T) {
	s1, _ := NewSpecter()
	s1.MarkAnnounced()

	_, err := NewSpecterConnectionRequest(s1, s1.PublicKey, ConnectionPeer)
	if err != ErrSpecterSelfConnection {
		t.Errorf("expected ErrSpecterSelfConnection, got %v", err)
	}
}

func TestSpecterConnectionAccept(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	// Accept as responder.
	if err := conn.Accept(s2); err != nil {
		t.Fatalf("Accept failed: %v", err)
	}

	// Should be complete.
	if conn.IsPending() {
		t.Error("accepted connection should not be pending")
	}
	if !conn.IsComplete() {
		t.Error("accepted connection should be complete")
	}
}

func TestSpecterConnectionAcceptNotAnnounced(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	// s2 is not announced.

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	err := conn.Accept(s2)
	if err != ErrSpecterNotAnnounced {
		t.Errorf("expected ErrSpecterNotAnnounced, got %v", err)
	}
}

func TestSpecterConnectionAcceptWrongKey(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s3, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()
	s3.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	// Try to accept with wrong Specter.
	err := conn.Accept(s3)
	if err != ErrSpecterNotAuthorized {
		t.Errorf("expected ErrSpecterNotAuthorized, got %v", err)
	}
}

func TestSpecterConnectionVerify(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
	conn.Accept(s2)

	// Should verify successfully.
	if err := conn.Verify(); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestSpecterConnectionVerifyInitiator(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	// Should verify initiator before accept.
	if err := conn.VerifyInitiator(); err != nil {
		t.Errorf("VerifyInitiator failed: %v", err)
	}
}

func TestSpecterConnectionMarshalUnmarshal(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionConfidant)
	conn.Accept(s2)

	// Marshal.
	data, err := conn.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal.
	conn2, err := UnmarshalSpecterConnection(data)
	if err != nil {
		t.Fatalf("UnmarshalSpecterConnection failed: %v", err)
	}

	// Compare fields.
	if conn2.InitiatorPublicKey != conn.InitiatorPublicKey {
		t.Error("initiator public key mismatch")
	}
	if conn2.ResponderPublicKey != conn.ResponderPublicKey {
		t.Error("responder public key mismatch")
	}
	if conn2.ConnectionType != conn.ConnectionType {
		t.Error("connection type mismatch")
	}
	if conn2.CreatedAt != conn.CreatedAt {
		t.Error("created_at mismatch")
	}
}

func TestSpecterConnectionInvolvesKey(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s3, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	if !conn.InvolvesKey(s1.PublicKey) {
		t.Error("should involve initiator key")
	}
	if !conn.InvolvesKey(s2.PublicKey) {
		t.Error("should involve responder key")
	}
	if conn.InvolvesKey(s3.PublicKey) {
		t.Error("should not involve unrelated key")
	}
}

func TestSpecterConnectionOtherParty(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	other, ok := conn.OtherParty(s1.PublicKey)
	if !ok || other != s2.PublicKey {
		t.Error("OtherParty for initiator should return responder")
	}

	other, ok = conn.OtherParty(s2.PublicKey)
	if !ok || other != s1.PublicKey {
		t.Error("OtherParty for responder should return initiator")
	}

	s3, _ := NewSpecter()
	_, ok = conn.OtherParty(s3.PublicKey)
	if ok {
		t.Error("OtherParty for unrelated key should return false")
	}
}

func TestSpecterConnectionRevocation(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	// Create revocation.
	rev, err := NewSpecterConnectionRevocation(s1, s2.PublicKey, ConnectionPeer)
	if err != nil {
		t.Fatalf("NewSpecterConnectionRevocation failed: %v", err)
	}

	// Verify.
	if err := rev.Verify(); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestSpecterConnectionRevocationNotAnnounced(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	// s1 is not announced.

	_, err := NewSpecterConnectionRevocation(s1, s2.PublicKey, ConnectionPeer)
	if err != ErrSpecterNotAnnounced {
		t.Errorf("expected ErrSpecterNotAnnounced, got %v", err)
	}
}

func TestSpecterConnectionRevocationMarshalUnmarshal(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()

	rev, _ := NewSpecterConnectionRevocation(s1, s2.PublicKey, ConnectionConfidant)

	// Marshal.
	data, err := rev.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal.
	rev2, err := UnmarshalSpecterConnectionRevocation(data)
	if err != nil {
		t.Fatalf("UnmarshalSpecterConnectionRevocation failed: %v", err)
	}

	// Compare fields.
	if rev2.RevokerPublicKey != rev.RevokerPublicKey {
		t.Error("revoker public key mismatch")
	}
	if rev2.TargetPublicKey != rev.TargetPublicKey {
		t.Error("target public key mismatch")
	}
	if rev2.ConnectionType != rev.ConnectionType {
		t.Error("connection type mismatch")
	}
}

func TestSpecterConnectionRevocationMatchesConnection(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
	conn.Accept(s2)

	// Matching revocation.
	rev, _ := NewSpecterConnectionRevocation(s1, s2.PublicKey, ConnectionPeer)
	if !rev.MatchesConnection(conn) {
		t.Error("revocation should match connection")
	}

	// Wrong type.
	rev2, _ := NewSpecterConnectionRevocation(s1, s2.PublicKey, ConnectionConfidant)
	if rev2.MatchesConnection(conn) {
		t.Error("revocation with wrong type should not match")
	}
}

func TestSpecterConnectionStore(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s3, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()
	s3.MarkAnnounced()

	store := NewSpecterConnectionStore()

	// Create and add connection.
	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
	conn.Accept(s2)
	if err := store.Add(conn); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Should be connected.
	if !store.IsConnected(s1.PublicKey, s2.PublicKey) {
		t.Error("s1 and s2 should be connected")
	}
	if !store.IsConnected(s2.PublicKey, s1.PublicKey) {
		t.Error("s2 and s1 should be connected (bidirectional)")
	}
	if store.IsConnected(s1.PublicKey, s3.PublicKey) {
		t.Error("s1 and s3 should not be connected")
	}

	// Get connection.
	retrieved, ok := store.Get(s1.PublicKey, s2.PublicKey)
	if !ok || retrieved != conn {
		t.Error("should retrieve stored connection")
	}

	// Count.
	if store.Count(s1.PublicKey) != 1 {
		t.Errorf("s1 should have 1 connection, got %d", store.Count(s1.PublicKey))
	}

	// Add another connection.
	conn2, _ := NewSpecterConnectionRequest(s1, s3.PublicKey, ConnectionConfidant)
	conn2.Accept(s3)
	store.Add(conn2)

	if store.Count(s1.PublicKey) != 2 {
		t.Errorf("s1 should have 2 connections, got %d", store.Count(s1.PublicKey))
	}

	// Get all connections.
	conns := store.GetConnectionsFor(s1.PublicKey)
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}

	// Remove connection.
	store.Remove(s1.PublicKey, s2.PublicKey)
	if store.IsConnected(s1.PublicKey, s2.PublicKey) {
		t.Error("s1 and s2 should no longer be connected")
	}
	if store.Count(s1.PublicKey) != 1 {
		t.Errorf("s1 should have 1 connection after removal, got %d", store.Count(s1.PublicKey))
	}
}

func TestSpecterConnectionStoreAddPending(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	store := NewSpecterConnectionStore()

	// Create pending connection (not accepted).
	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	// Should not be able to add pending connection.
	err := store.Add(conn)
	if err != ErrMissingSpecterSignature {
		t.Errorf("expected ErrMissingSpecterSignature, got %v", err)
	}
}

func TestSpecterConnectionVerifySharedSecret(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s3, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()
	s3.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)

	// Initiator should verify.
	if !conn.VerifySharedSecret(s1) {
		t.Error("initiator should verify shared secret")
	}

	// Responder should verify.
	if !conn.VerifySharedSecret(s2) {
		t.Error("responder should verify shared secret")
	}

	// Unrelated Specter should not verify.
	if conn.VerifySharedSecret(s3) {
		t.Error("unrelated Specter should not verify shared secret")
	}
}

func TestSpecterConnectionTypes(t *testing.T) {
	// Test all connection types.
	types := []proto.SpecterConnectionType{
		ConnectionPeer,
		ConnectionConfidant,
		ConnectionBlocked,
	}

	for _, connType := range types {
		s1, _ := NewSpecter()
		s2, _ := NewSpecter()
		s1.MarkAnnounced()
		s2.MarkAnnounced()

		conn, err := NewSpecterConnectionRequest(s1, s2.PublicKey, connType)
		if err != nil {
			t.Errorf("failed to create connection type %v: %v", connType, err)
			continue
		}

		if conn.ConnectionType != connType {
			t.Errorf("connection type mismatch: expected %v, got %v", connType, conn.ConnectionType)
		}
	}
}

func TestSpecterConnectionConcurrency(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s1.MarkAnnounced()
	s2.MarkAnnounced()

	conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
	conn.Accept(s2)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			conn.IsPending()
			conn.IsComplete()
			conn.InvolvesKey(s1.PublicKey)
			conn.OtherParty(s2.PublicKey)
			conn.Marshal()
			conn.Verify()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSpecterConnectionStoreConcurrency(t *testing.T) {
	store := NewSpecterConnectionStore()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			s1, _ := NewSpecter()
			s2, _ := NewSpecter()
			s1.MarkAnnounced()
			s2.MarkAnnounced()

			conn, _ := NewSpecterConnectionRequest(s1, s2.PublicKey, ConnectionPeer)
			conn.Accept(s2)
			store.Add(conn)
			store.IsConnected(s1.PublicKey, s2.PublicKey)
			store.Count(s1.PublicKey)
			store.GetConnectionsFor(s1.PublicKey)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
