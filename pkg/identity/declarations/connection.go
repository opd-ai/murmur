// Package declarations provides identity declaration creation and parsing.
// This file implements Connection Declarations and Revocations.
// Per DESIGN_DOCUMENT.md §2.2, connections are bilateral signed relationships.
package declarations

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/proto"
	pb "google.golang.org/protobuf/proto"
)

// Connection type constants.
const (
	ConnectionFriend  = proto.ConnectionType_CONNECTION_TYPE_FRIEND
	ConnectionFollow  = proto.ConnectionType_CONNECTION_TYPE_FOLLOW
	ConnectionBlock   = proto.ConnectionType_CONNECTION_TYPE_BLOCK
	ConnectionTrusted = proto.ConnectionType_CONNECTION_TYPE_TRUSTED
)

// Revocation reason constants.
const (
	RevocationUserRequest = proto.RevocationReason_REVOCATION_REASON_USER_REQUEST
	RevocationInactivity  = proto.RevocationReason_REVOCATION_REASON_INACTIVITY
	RevocationPolicy      = proto.RevocationReason_REVOCATION_REASON_POLICY
	RevocationKeyRotation = proto.RevocationReason_REVOCATION_REASON_KEY_ROTATION
)

// Connection errors.
var (
	ErrMissingInitiator    = errors.New("missing initiator public key")
	ErrMissingResponder    = errors.New("missing responder public key")
	ErrSelfConnection      = errors.New("cannot connect to self")
	ErrMissingSignature    = errors.New("missing required signature")
	ErrInvalidInitiatorSig = errors.New("invalid initiator signature")
	ErrInvalidResponderSig = errors.New("invalid responder signature")
	ErrConnectionNotFound  = errors.New("connection not found")
	ErrAlreadyConnected    = errors.New("already connected")
	ErrNotAuthorized       = errors.New("not authorized to perform this action")
)

// ConnectionDeclaration represents a bilateral relationship between two identities.
// Per DESIGN_DOCUMENT.md §2.2, connections require signatures from both parties.
type ConnectionDeclaration struct {
	// InitiatorPublicKey is the Ed25519 public key of the initiator.
	InitiatorPublicKey []byte

	// ResponderPublicKey is the Ed25519 public key of the responder.
	ResponderPublicKey []byte

	// InitiatorSignature is the initiator's signature.
	InitiatorSignature []byte

	// ResponderSignature is the responder's signature.
	ResponderSignature []byte

	// CreatedAt is the Unix timestamp when connection was established.
	CreatedAt int64

	// ConnectionType defines the relationship type.
	ConnectionType proto.ConnectionType

	// MutualName is an optional name visible to the connected party.
	MutualName string
}

// NewConnectionRequest creates a new connection request from initiator to responder.
// The initiator signs the request; the responder must call AcceptConnection() to complete.
func NewConnectionRequest(initiator *keys.KeyPair, responderPubKey []byte, connType proto.ConnectionType) (*ConnectionDeclaration, error) {
	if initiator == nil {
		return nil, ErrNilKeyPair
	}
	if len(responderPubKey) != ed25519.PublicKeySize {
		return nil, ErrMissingResponder
	}
	if bytesEqual(initiator.PublicKey, responderPubKey) {
		return nil, ErrSelfConnection
	}

	conn := &ConnectionDeclaration{
		InitiatorPublicKey: initiator.PublicKey,
		ResponderPublicKey: responderPubKey,
		CreatedAt:          time.Now().Unix(),
		ConnectionType:     connType,
	}

	// Sign as initiator.
	payload := conn.signingPayload()
	conn.InitiatorSignature = initiator.Sign(payload)

	return conn, nil
}

// AcceptConnection signs the connection request as responder, completing the bilateral agreement.
func (c *ConnectionDeclaration) AcceptConnection(responder *keys.KeyPair) error {
	if responder == nil {
		return ErrNilKeyPair
	}
	if !bytesEqual(c.ResponderPublicKey, responder.PublicKey) {
		return ErrNotAuthorized
	}
	if len(c.InitiatorSignature) == 0 {
		return ErrMissingSignature
	}

	// Sign as responder.
	payload := c.signingPayload()
	c.ResponderSignature = responder.Sign(payload)

	return nil
}

// Verify verifies both initiator and responder signatures.
func (c *ConnectionDeclaration) Verify() error {
	if err := c.validateConnectionFields(); err != nil {
		return err
	}

	payload := c.signingPayload()
	return c.verifySignatures(payload)
}

// validateConnectionFields checks that all required connection fields are present.
func (c *ConnectionDeclaration) validateConnectionFields() error {
	if len(c.InitiatorPublicKey) != ed25519.PublicKeySize {
		return ErrMissingInitiator
	}
	if len(c.ResponderPublicKey) != ed25519.PublicKeySize {
		return ErrMissingResponder
	}
	if len(c.InitiatorSignature) == 0 || len(c.ResponderSignature) == 0 {
		return ErrMissingSignature
	}
	return nil
}

// verifySignatures verifies both initiator and responder signatures.
func (c *ConnectionDeclaration) verifySignatures(payload []byte) error {
	if !keys.Verify(c.InitiatorPublicKey, payload, c.InitiatorSignature) {
		return ErrInvalidInitiatorSig
	}
	if !keys.Verify(c.ResponderPublicKey, payload, c.ResponderSignature) {
		return ErrInvalidResponderSig
	}
	return nil
}

// VerifyInitiator verifies only the initiator's signature (for pending requests).
func (c *ConnectionDeclaration) VerifyInitiator() error {
	if len(c.InitiatorPublicKey) != ed25519.PublicKeySize {
		return ErrMissingInitiator
	}
	if len(c.InitiatorSignature) == 0 {
		return ErrMissingSignature
	}

	payload := c.signingPayload()
	if !keys.Verify(c.InitiatorPublicKey, payload, c.InitiatorSignature) {
		return ErrInvalidInitiatorSig
	}

	return nil
}

// IsPending returns true if the connection is awaiting responder signature.
func (c *ConnectionDeclaration) IsPending() bool {
	return len(c.InitiatorSignature) > 0 && len(c.ResponderSignature) == 0
}

// IsComplete returns true if both parties have signed.
func (c *ConnectionDeclaration) IsComplete() bool {
	return len(c.InitiatorSignature) > 0 && len(c.ResponderSignature) > 0
}

// signingPayload creates the byte sequence to be signed by both parties.
// Format: initiator_pubkey || responder_pubkey || timestamp || connection_type || mutual_name_len || mutual_name
func (c *ConnectionDeclaration) signingPayload() []byte {
	size := ed25519.PublicKeySize*2 + 8 + 4 + 4 + len(c.MutualName)
	buf := make([]byte, 0, size)

	buf = append(buf, c.InitiatorPublicKey...)
	buf = append(buf, c.ResponderPublicKey...)
	buf = appendU64BigEndian(buf, uint64(c.CreatedAt))
	buf = appendU32BigEndian(buf, uint32(c.ConnectionType))

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(c.MutualName)))
	buf = append(buf, lenBuf...)
	buf = append(buf, []byte(c.MutualName)...)

	return buf
}

// Marshal serializes the connection declaration to protobuf wire format.
func (c *ConnectionDeclaration) Marshal() ([]byte, error) {
	pbConn := &proto.ConnectionDeclaration{
		InitiatorPublicKey: c.InitiatorPublicKey,
		ResponderPublicKey: c.ResponderPublicKey,
		InitiatorSignature: c.InitiatorSignature,
		ResponderSignature: c.ResponderSignature,
		CreatedAt:          c.CreatedAt,
		ConnectionType:     c.ConnectionType,
		MutualName:         c.MutualName,
	}
	return pb.Marshal(pbConn)
}

// UnmarshalConnection deserializes a connection declaration from protobuf wire format.
func UnmarshalConnection(data []byte) (*ConnectionDeclaration, error) {
	pbConn := &proto.ConnectionDeclaration{}
	if err := pb.Unmarshal(data, pbConn); err != nil {
		return nil, fmt.Errorf("unmarshaling connection: %w", err)
	}
	return &ConnectionDeclaration{
		InitiatorPublicKey: pbConn.InitiatorPublicKey,
		ResponderPublicKey: pbConn.ResponderPublicKey,
		InitiatorSignature: pbConn.InitiatorSignature,
		ResponderSignature: pbConn.ResponderSignature,
		CreatedAt:          pbConn.CreatedAt,
		ConnectionType:     pbConn.ConnectionType,
		MutualName:         pbConn.MutualName,
	}, nil
}

// InvolvesKey returns true if the given public key is either initiator or responder.
func (c *ConnectionDeclaration) InvolvesKey(publicKey []byte) bool {
	return bytesEqual(c.InitiatorPublicKey, publicKey) || bytesEqual(c.ResponderPublicKey, publicKey)
}

// OtherParty returns the public key of the other party given one party's key.
func (c *ConnectionDeclaration) OtherParty(publicKey []byte) []byte {
	if bytesEqual(c.InitiatorPublicKey, publicKey) {
		return c.ResponderPublicKey
	}
	if bytesEqual(c.ResponderPublicKey, publicKey) {
		return c.InitiatorPublicKey
	}
	return nil
}

// ConnectionRevocation represents a cancellation of an existing connection.
// Per DESIGN_DOCUMENT.md, revocation is unilateral - either party can revoke.
type ConnectionRevocation struct {
	// RevokerPublicKey is the Ed25519 public key of the revoking party.
	RevokerPublicKey []byte

	// TargetPublicKey is the Ed25519 public key of the other party.
	TargetPublicKey []byte

	// Signature is the revoker's signature.
	Signature []byte

	// RevokedAt is the Unix timestamp of revocation.
	RevokedAt int64

	// ConnectionType is the type of connection being revoked.
	ConnectionType proto.ConnectionType

	// Reason is the revocation reason.
	Reason proto.RevocationReason
}

// NewRevocation creates a new connection revocation.
func NewRevocation(revoker *keys.KeyPair, targetPubKey []byte, connType proto.ConnectionType, reason proto.RevocationReason) (*ConnectionRevocation, error) {
	if revoker == nil {
		return nil, ErrNilKeyPair
	}
	if len(targetPubKey) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}

	rev := &ConnectionRevocation{
		RevokerPublicKey: revoker.PublicKey,
		TargetPublicKey:  targetPubKey,
		RevokedAt:        time.Now().Unix(),
		ConnectionType:   connType,
		Reason:           reason,
	}

	// Sign the revocation.
	payload := rev.signingPayload()
	rev.Signature = revoker.Sign(payload)

	return rev, nil
}

// Verify verifies the revocation signature.
func (r *ConnectionRevocation) Verify() error {
	if len(r.RevokerPublicKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}
	if len(r.Signature) == 0 {
		return ErrMissingSignature
	}

	payload := r.signingPayload()
	if !keys.Verify(r.RevokerPublicKey, payload, r.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// signingPayload creates the byte sequence to be signed.
// Format: revoker_pubkey || target_pubkey || revoked_at || connection_type || reason
func (r *ConnectionRevocation) signingPayload() []byte {
	size := ed25519.PublicKeySize*2 + 8 + 4 + 4
	buf := make([]byte, 0, size)

	buf = append(buf, r.RevokerPublicKey...)
	buf = append(buf, r.TargetPublicKey...)
	buf = appendU64BigEndian(buf, uint64(r.RevokedAt))
	buf = appendU32BigEndian(buf, uint32(r.ConnectionType))
	buf = appendU32BigEndian(buf, uint32(r.Reason))

	return buf
}

// appendU64BigEndian appends a uint64 in big-endian format.
func appendU64BigEndian(buf []byte, val uint64) []byte {
	tmp := make([]byte, 8)
	binary.BigEndian.PutUint64(tmp, val)
	return append(buf, tmp...)
}

// appendU32BigEndian appends a uint32 in big-endian format.
func appendU32BigEndian(buf []byte, val uint32) []byte {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, val)
	return append(buf, tmp...)
}

// appendStringWithLen appends a string with 4-byte big-endian length prefix.
func appendStringWithLen(buf []byte, s string) []byte {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, uint32(len(s)))
	buf = append(buf, tmp...)
	return append(buf, []byte(s)...)
}

// Marshal serializes the revocation to protobuf wire format.
func (r *ConnectionRevocation) Marshal() ([]byte, error) {
	pbRev := &proto.ConnectionRevocation{
		RevokerPublicKey: r.RevokerPublicKey,
		TargetPublicKey:  r.TargetPublicKey,
		Signature:        r.Signature,
		RevokedAt:        r.RevokedAt,
		ConnectionType:   r.ConnectionType,
		Reason:           r.Reason,
	}
	return pb.Marshal(pbRev)
}

// UnmarshalRevocation deserializes a revocation from protobuf wire format.
func UnmarshalRevocation(data []byte) (*ConnectionRevocation, error) {
	pbRev := &proto.ConnectionRevocation{}
	if err := pb.Unmarshal(data, pbRev); err != nil {
		return nil, fmt.Errorf("unmarshaling revocation: %w", err)
	}
	return &ConnectionRevocation{
		RevokerPublicKey: pbRev.RevokerPublicKey,
		TargetPublicKey:  pbRev.TargetPublicKey,
		Signature:        pbRev.Signature,
		RevokedAt:        pbRev.RevokedAt,
		ConnectionType:   pbRev.ConnectionType,
		Reason:           pbRev.Reason,
	}, nil
}

// MatchesConnection returns true if this revocation matches the given connection.
func (r *ConnectionRevocation) MatchesConnection(conn *ConnectionDeclaration) bool {
	if conn == nil {
		return false
	}
	if r.ConnectionType != conn.ConnectionType {
		return false
	}
	// Revoker must be one of the parties in the connection.
	return conn.InvolvesKey(r.RevokerPublicKey) && conn.InvolvesKey(r.TargetPublicKey)
}

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
