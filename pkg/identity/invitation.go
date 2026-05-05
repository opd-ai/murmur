// Package identity provides invitation generation and acceptance for MURMUR.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, invitations enable warm-start onboarding
// with direct bootstrap through the inviter's node.
package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/skip2/go-qrcode"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// InviteURIScheme is the URI scheme for MURMUR invitations.
const InviteURIScheme = "murmur://invite/"

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
	encoded, err := inv.Encode()
	if err != nil {
		return "", err
	}

	return InviteURIScheme + encoded, nil
}

// DecodeInvitation deserializes an invitation from a Base64-encoded string.
// Accepts both raw Base64 and murmur:// URIs.
func DecodeInvitation(encoded string) (*Invitation, error) {
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
