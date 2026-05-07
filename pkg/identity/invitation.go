// Package identity provides invitation generation and acceptance for MURMUR.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, invitations enable warm-start onboarding
// with direct bootstrap through the inviter's node.
package identity

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/skip2/go-qrcode"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// InviteURIScheme is the URI scheme for MURMUR invitations.
const InviteURIScheme = "murmur://invite/"

// InviteURISchemeV2 is the URI scheme for signed out-of-band invitation codes.
// Per PLAN.md §7.3, this format supports text, QR, and paper transfer while
// carrying direct bootstrap addresses for disconnected environments.
const InviteURISchemeV2 = "murmur://invite2/"

const (
	inviteV2Version   = 2
	maxBootstrapAddrs = 8
)

// MaxWelcomeMessageLength is the maximum length of the welcome message.
const MaxWelcomeMessageLength = 128

// Invitation represents a MURMUR invitation.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, invitations contain:
// - Inviter's Peer ID (for bootstrap bypass)
// - Inviter's public key (for verified connection)
// - Optional welcome message (max 128 characters)
type Invitation struct {
	PeerID         peer.ID
	PublicKey      ed25519.PublicKey
	WelcomeMessage string
	BootstrapAddrs []string
	ExpiresUnix    int64
	Signature      []byte
}

// SignedInvitationOptions configures signed out-of-band invitation generation.
type SignedInvitationOptions struct {
	BootstrapAddrs []string
	WelcomeMessage string
	TTL            time.Duration
}

// GenerateInvitation creates a new invitation from the inviter's identity.
// The invitation encodes all necessary data for bootstrap bypass and
// direct connection establishment. Per VIRAL_GROWTH_AND_ONBOARDING.md,
// invitation generation is frictionless (two-tap).
func GenerateInvitation(peerID peer.ID, publicKey ed25519.PublicKey, welcomeMsg string) (*Invitation, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}

	// Truncate welcome message if too long.
	if len(welcomeMsg) > MaxWelcomeMessageLength {
		welcomeMsg = welcomeMsg[:MaxWelcomeMessageLength]
	}

	return &Invitation{
		PeerID:         peerID,
		PublicKey:      publicKey,
		WelcomeMessage: welcomeMsg,
	}, nil
}

// Encode serializes the invitation to a URL-safe Base64 string.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, the encoded invitation is
// approximately 100-150 characters, suitable for text messages,
// tweets, and forum posts.
func (inv *Invitation) Encode() (string, error) {
	// Create protobuf message.
	pbInv := &pb.Invitation{
		PeerId:         []byte(inv.PeerID),
		PublicKey:      inv.PublicKey,
		WelcomeMessage: inv.WelcomeMessage,
	}

	// Serialize to protobuf.
	data, err := proto.Marshal(pbInv)
	if err != nil {
		return "", fmt.Errorf("marshaling invitation: %w", err)
	}

	// Encode to URL-safe Base64.
	encoded := base64.URLEncoding.EncodeToString(data)

	return encoded, nil
}

// EncodeURI returns the invitation as a murmur:// URI.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, the murmur:// URI scheme
// enables deep linking on supported platforms.
func (inv *Invitation) EncodeURI() (string, error) {
	if len(inv.Signature) > 0 {
		encoded, err := inv.EncodeSigned()
		if err != nil {
			return "", err
		}
		return InviteURISchemeV2 + encoded, nil
	}

	encoded, err := inv.Encode()
	if err != nil {
		return "", err
	}

	return InviteURIScheme + encoded, nil
}

// DecodeInvitation deserializes an invitation from a Base64-encoded string.
// Accepts both raw Base64 and murmur:// URIs.
func DecodeInvitation(encoded string) (*Invitation, error) {
	if strings.HasPrefix(encoded, InviteURISchemeV2) {
		return DecodeSignedInvitation(strings.TrimPrefix(encoded, InviteURISchemeV2))
	}

	// Strip murmur:// prefix if present.
	encoded = strings.TrimPrefix(encoded, InviteURIScheme)

	// Decode from Base64.
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	// Unmarshal protobuf.
	pbInv := &pb.Invitation{}
	if err := proto.Unmarshal(data, pbInv); err != nil {
		return nil, fmt.Errorf("unmarshaling invitation: %w", err)
	}

	// Parse peer ID.
	peerID, err := peer.IDFromBytes(pbInv.PeerId)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	// Validate public key.
	if len(pbInv.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}

	return &Invitation{
		PeerID:         peerID,
		PublicKey:      pbInv.PublicKey,
		WelcomeMessage: pbInv.WelcomeMessage,
	}, nil
}

// GenerateSignedInvitation creates a signed out-of-band invitation code that can
// bootstrap via embedded addresses when default bootstraps are unavailable.
func GenerateSignedInvitation(
	peerID peer.ID,
	publicKey ed25519.PublicKey,
	privateKey ed25519.PrivateKey,
	opts SignedInvitationOptions,
) (*Invitation, error) {
	inv, err := GenerateInvitation(peerID, publicKey, opts.WelcomeMessage)
	if err != nil {
		return nil, err
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	if len(opts.BootstrapAddrs) == 0 {
		return nil, errors.New("at least one bootstrap address is required")
	}
	if len(opts.BootstrapAddrs) > maxBootstrapAddrs {
		return nil, fmt.Errorf("too many bootstrap addresses: got %d, max %d", len(opts.BootstrapAddrs), maxBootstrapAddrs)
	}

	ttl := opts.TTL
	if ttl < 0 {
		return nil, errors.New("invitation TTL cannot be negative")
	}
	if ttl == 0 {
		ttl = 30 * 24 * time.Hour
	}

	inv.BootstrapAddrs = append([]string(nil), opts.BootstrapAddrs...)
	inv.ExpiresUnix = time.Now().Add(ttl).Unix()

	payload, err := inv.marshalV2Payload()
	if err != nil {
		return nil, err
	}
	inv.Signature = ed25519.Sign(privateKey, payload)

	return inv, nil
}

// EncodeSigned serializes a signed invitation payload to URL-safe base64.
func (inv *Invitation) EncodeSigned() (string, error) {
	if len(inv.Signature) != ed25519.SignatureSize {
		return "", errors.New("missing or invalid signature")
	}
	payload, err := inv.marshalV2Payload()
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if _, err := out.Write(payload); err != nil {
		return "", fmt.Errorf("writing payload: %w", err)
	}
	if err := binary.Write(&out, binary.BigEndian, uint16(len(inv.Signature))); err != nil {
		return "", fmt.Errorf("writing signature length: %w", err)
	}
	if _, err := out.Write(inv.Signature); err != nil {
		return "", fmt.Errorf("writing signature: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(out.Bytes()), nil
}

// DecodeSignedInvitation decodes and verifies a signed out-of-band invitation code.
func DecodeSignedInvitation(encoded string) (*Invitation, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding signed invitation: %w", err)
	}

	inv, payload, sig, err := unmarshalV2(raw)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(inv.PublicKey, payload, sig) {
		return nil, errors.New("invalid invitation signature")
	}
	inv.Signature = append([]byte(nil), sig...)

	if inv.ExpiresUnix <= 0 {
		return nil, errors.New("missing invitation expiry")
	}
	if time.Now().Unix() > inv.ExpiresUnix {
		return nil, errors.New("invitation expired")
	}

	return inv, nil
}

func (inv *Invitation) marshalV2Payload() ([]byte, error) {
	if len(inv.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}
	if len(inv.BootstrapAddrs) == 0 {
		return nil, errors.New("at least one bootstrap address is required")
	}
	if len(inv.BootstrapAddrs) > maxBootstrapAddrs {
		return nil, fmt.Errorf("too many bootstrap addresses: got %d, max %d", len(inv.BootstrapAddrs), maxBootstrapAddrs)
	}

	var out bytes.Buffer
	out.WriteByte(inviteV2Version)
	if err := writeBytesWithU16(&out, []byte(inv.PeerID)); err != nil {
		return nil, err
	}
	if err := writeBytesWithU16(&out, inv.PublicKey); err != nil {
		return nil, err
	}
	if err := writeStringWithU16(&out, inv.WelcomeMessage); err != nil {
		return nil, err
	}
	if err := binary.Write(&out, binary.BigEndian, inv.ExpiresUnix); err != nil {
		return nil, fmt.Errorf("writing expiration: %w", err)
	}

	if len(inv.BootstrapAddrs) > 255 {
		return nil, errors.New("too many bootstrap addresses")
	}
	out.WriteByte(byte(len(inv.BootstrapAddrs)))
	for _, addr := range inv.BootstrapAddrs {
		if err := writeStringWithU16(&out, addr); err != nil {
			return nil, err
		}
	}

	return out.Bytes(), nil
}

func unmarshalV2(raw []byte) (*Invitation, []byte, []byte, error) {
	if len(raw) < 3 {
		return nil, nil, nil, errors.New("signed invitation payload too short")
	}

	r := bytes.NewReader(raw)
	version, err := r.ReadByte()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading version: %w", err)
	}
	if version != inviteV2Version {
		return nil, nil, nil, fmt.Errorf("unsupported invitation version: %d", version)
	}

	peerBytes, err := readBytesWithU16(r)
	if err != nil {
		return nil, nil, nil, err
	}
	peerID, err := peer.IDFromBytes(peerBytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	pubKey, err := readBytesWithU16(r)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, nil, nil, errors.New("invalid public key size")
	}

	welcome, err := readStringWithU16(r)
	if err != nil {
		return nil, nil, nil, err
	}

	var expiresUnix int64
	if err := binary.Read(r, binary.BigEndian, &expiresUnix); err != nil {
		return nil, nil, nil, fmt.Errorf("reading expiration: %w", err)
	}

	addrCount, err := r.ReadByte()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading bootstrap address count: %w", err)
	}
	if addrCount == 0 {
		return nil, nil, nil, errors.New("signed invitation has no bootstrap addresses")
	}

	addrs := make([]string, 0, addrCount)
	for i := byte(0); i < addrCount; i++ {
		addr, err := readStringWithU16(r)
		if err != nil {
			return nil, nil, nil, err
		}
		if addr == "" {
			return nil, nil, nil, errors.New("empty bootstrap address")
		}
		addrs = append(addrs, addr)
	}

	payloadLen := len(raw) - r.Len()
	if r.Len() < 2 {
		return nil, nil, nil, errors.New("missing signature length")
	}

	var sigLen uint16
	if err := binary.Read(r, binary.BigEndian, &sigLen); err != nil {
		return nil, nil, nil, fmt.Errorf("reading signature length: %w", err)
	}
	if sigLen != ed25519.SignatureSize {
		return nil, nil, nil, errors.New("invalid signature size")
	}
	if r.Len() != int(sigLen) {
		return nil, nil, nil, errors.New("invalid trailing signature data")
	}

	sig := make([]byte, sigLen)
	if _, err := r.Read(sig); err != nil {
		return nil, nil, nil, fmt.Errorf("reading signature: %w", err)
	}

	payload := raw[:payloadLen]
	inv := &Invitation{
		PeerID:         peerID,
		PublicKey:      append([]byte(nil), pubKey...),
		WelcomeMessage: welcome,
		BootstrapAddrs: addrs,
		ExpiresUnix:    expiresUnix,
	}

	return inv, payload, sig, nil
}

func writeStringWithU16(buf *bytes.Buffer, value string) error {
	return writeBytesWithU16(buf, []byte(value))
}

func writeBytesWithU16(buf *bytes.Buffer, data []byte) error {
	if len(data) > 65535 {
		return errors.New("field too large for invitation encoding")
	}
	if err := binary.Write(buf, binary.BigEndian, uint16(len(data))); err != nil {
		return fmt.Errorf("writing field length: %w", err)
	}
	if _, err := buf.Write(data); err != nil {
		return fmt.Errorf("writing field data: %w", err)
	}
	return nil
}

func readStringWithU16(r *bytes.Reader) (string, error) {
	b, err := readBytesWithU16(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readBytesWithU16(r *bytes.Reader) ([]byte, error) {
	var n uint16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, fmt.Errorf("reading field length: %w", err)
	}
	if r.Len() < int(n) {
		return nil, errors.New("field length exceeds remaining payload")
	}
	out := make([]byte, n)
	if _, err := r.Read(out); err != nil {
		return nil, fmt.Errorf("reading field data: %w", err)
	}
	return out, nil
}

// Validate checks if the invitation is valid.
func (inv *Invitation) Validate() error {
	if len(inv.PeerID) == 0 {
		return errors.New("missing peer ID")
	}
	if len(inv.PublicKey) != ed25519.PublicKeySize {
		return errors.New("invalid public key size")
	}
	if len(inv.WelcomeMessage) > MaxWelcomeMessageLength {
		return errors.New("welcome message too long")
	}

	return inv.validateSignedFields()
}

func (inv *Invitation) validateSignedFields() error {
	if len(inv.Signature) == 0 {
		return nil
	}
	if len(inv.BootstrapAddrs) == 0 {
		return errors.New("signed invitation must include bootstrap addresses")
	}
	if inv.ExpiresUnix <= 0 {
		return errors.New("signed invitation missing expiry")
	}
	if len(inv.Signature) != ed25519.SignatureSize {
		return errors.New("invalid signature size")
	}
	return nil
}

// String returns a human-readable representation of the invitation.
func (inv *Invitation) String() string {
	if inv.WelcomeMessage != "" {
		return fmt.Sprintf("Invitation from %s: %s", inv.PeerID.ShortString(), inv.WelcomeMessage)
	}
	return fmt.Sprintf("Invitation from %s", inv.PeerID.ShortString())
}

// GenerateQRCode creates a QR code image for the invitation.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, QR codes enable rapid in-person
// invitation at conferences, meetups, and social gatherings.
// The size parameter specifies the pixel size of the QR code (e.g., 256, 512).
func (inv *Invitation) GenerateQRCode(size int) (image.Image, error) {
	// Encode invitation as URI.
	uri, err := inv.EncodeURI()
	if err != nil {
		return nil, fmt.Errorf("encoding URI: %w", err)
	}

	// Generate QR code.
	qr, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("creating QR code: %w", err)
	}

	// Set size and disable border.
	qr.DisableBorder = true

	// Return as image.
	return qr.Image(size), nil
}

// GenerateQRCodePNG creates a QR code PNG for the invitation.
// Returns PNG-encoded bytes suitable for display or file storage.
func (inv *Invitation) GenerateQRCodePNG(size int) ([]byte, error) {
	// Encode invitation as URI.
	uri, err := inv.EncodeURI()
	if err != nil {
		return nil, fmt.Errorf("encoding URI: %w", err)
	}

	// Generate QR code PNG with Medium error correction.
	return qrcode.Encode(uri, qrcode.Medium, size)
}
