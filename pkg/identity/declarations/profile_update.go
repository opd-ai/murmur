// Package declarations provides identity declaration creation and parsing.
// This file implements Profile Update declarations for changing identity metadata.
// Per DESIGN_DOCUMENT.md, profile updates require new signatures.
package declarations

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/proto"
	pb "google.golang.org/protobuf/proto"
)

// ProfileUpdate errors.
var (
	ErrNoChanges         = errors.New("no changes in profile update")
	ErrVersionNotHigher  = errors.New("version must be higher than previous")
	ErrProfileNotSigned  = errors.New("profile update not signed")
	ErrInvalidProfileSig = errors.New("invalid profile update signature")
)

// ProfileUpdate represents an update to identity metadata.
// Per DESIGN_DOCUMENT.md, profile updates require new signatures and
// incremented version numbers.
type ProfileUpdate struct {
	// PublicKey is the Ed25519 public key of the identity.
	PublicKey []byte

	// DisplayName is the new display name (empty means unchanged).
	DisplayName string

	// Bio is the new bio (empty means unchanged).
	Bio string

	// SigilPNG is the new sigil PNG (nil means unchanged).
	SigilPNG []byte

	// PrivacyMode is the new privacy mode.
	PrivacyMode modes.Mode

	// UpdatedAt is the Unix timestamp of the update.
	UpdatedAt int64

	// Version must be higher than the previous declaration version.
	Version uint32

	// Signature is the Ed25519 signature over update fields.
	Signature []byte

	// hasDisplayName indicates if DisplayName was explicitly set.
	hasDisplayName bool

	// hasBio indicates if Bio was explicitly set.
	hasBio bool

	// hasSigil indicates if SigilPNG was explicitly set.
	hasSigil bool

	// hasPrivacyMode indicates if PrivacyMode was explicitly set.
	hasPrivacyMode bool
}

// NewProfileUpdate creates a new profile update for the given keypair.
// Use SetXxx methods to specify which fields to update.
func NewProfileUpdate(kp *keys.KeyPair, previousVersion uint32) (*ProfileUpdate, error) {
	if kp == nil {
		return nil, ErrNilKeyPair
	}

	return &ProfileUpdate{
		PublicKey: kp.PublicKey,
		UpdatedAt: time.Now().Unix(),
		Version:   previousVersion + 1,
	}, nil
}

// SetDisplayName sets the new display name.
func (p *ProfileUpdate) SetDisplayName(name string) error {
	if len(name) > MaxDisplayNameLen {
		return ErrDisplayNameTooLon
	}
	p.DisplayName = name
	p.hasDisplayName = true
	return nil
}

// SetBio sets the new bio.
func (p *ProfileUpdate) SetBio(bio string) error {
	if len(bio) > MaxBioLen {
		return ErrBioTooLong
	}
	p.Bio = bio
	p.hasBio = true
	return nil
}

// SetSigil sets the new sigil PNG.
func (p *ProfileUpdate) SetSigil(png []byte) {
	p.SigilPNG = png
	p.hasSigil = true
}

// SetPrivacyMode sets the new privacy mode.
func (p *ProfileUpdate) SetPrivacyMode(mode modes.Mode) {
	p.PrivacyMode = mode
	p.hasPrivacyMode = true
}

// HasChanges returns true if at least one field was set for update.
func (p *ProfileUpdate) HasChanges() bool {
	return p.hasDisplayName || p.hasBio || p.hasSigil || p.hasPrivacyMode
}

// Sign signs the profile update.
func (p *ProfileUpdate) Sign(kp *keys.KeyPair) error {
	if kp == nil {
		return ErrNilKeyPair
	}
	if !p.HasChanges() {
		return ErrNoChanges
	}

	payload := p.signingPayload()
	p.Signature = kp.Sign(payload)
	return nil
}

// Verify verifies the profile update signature.
func (p *ProfileUpdate) Verify() error {
	if len(p.Signature) == 0 {
		return ErrProfileNotSigned
	}
	if len(p.PublicKey) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}

	payload := p.signingPayload()
	if !keys.Verify(p.PublicKey, payload, p.Signature) {
		return ErrInvalidProfileSig
	}

	return nil
}

// ValidateAgainst validates this update against a previous declaration.
func (p *ProfileUpdate) ValidateAgainst(prev *Declaration) error {
	if prev == nil {
		return nil // No previous declaration to validate against.
	}

	if !bytesEqual(p.PublicKey, prev.PublicKey) {
		return ErrInvalidPublicKey
	}

	if p.Version <= prev.Version {
		return ErrVersionNotHigher
	}

	return nil
}

// ValidateTimestamp checks if the timestamp is within acceptable bounds.
func (p *ProfileUpdate) ValidateTimestamp() error {
	now := time.Now().Unix()
	maxSkew := int64(MaxTimestampSkew.Seconds())
	if p.UpdatedAt < now-maxSkew {
		return ErrTimestampTooOld
	}
	if p.UpdatedAt > now+maxSkew {
		return ErrTimestampTooNew
	}
	return nil
}

// Validate performs full validation of the profile update.
func (p *ProfileUpdate) Validate(prev *Declaration) error {
	if err := p.ValidateTimestamp(); err != nil {
		return err
	}
	if err := p.ValidateAgainst(prev); err != nil {
		return err
	}
	if err := p.Verify(); err != nil {
		return err
	}
	if len(p.DisplayName) > MaxDisplayNameLen {
		return ErrDisplayNameTooLon
	}
	if len(p.Bio) > MaxBioLen {
		return ErrBioTooLong
	}
	return nil
}

// signingPayload creates the byte sequence to be signed.
// Format: public_key || display_name_len || display_name || bio_len || bio ||
// sigil_len || sigil || privacy_mode || updated_at || version
func (p *ProfileUpdate) signingPayload() []byte {
	size := len(p.PublicKey) + 4 + len(p.DisplayName) + 4 + len(p.Bio) + 4 + len(p.SigilPNG) + 4 + 8 + 4
	buf := make([]byte, 0, size)

	buf = append(buf, p.PublicKey...)
	buf = appendStringWithLen(buf, p.DisplayName)
	buf = appendStringWithLen(buf, p.Bio)

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(p.SigilPNG)))
	buf = append(buf, lenBuf...)
	buf = append(buf, p.SigilPNG...)

	binary.BigEndian.PutUint32(lenBuf, uint32(p.PrivacyMode))
	buf = append(buf, lenBuf...)

	tsBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBuf, uint64(p.UpdatedAt))
	buf = append(buf, tsBuf...)

	binary.BigEndian.PutUint32(lenBuf, p.Version)
	buf = append(buf, lenBuf...)

	return buf
}

// ApplyTo applies this update to a declaration, returning the updated declaration.
func (p *ProfileUpdate) ApplyTo(decl *Declaration) *Declaration {
	result := &Declaration{
		PublicKey:   decl.PublicKey,
		DisplayName: decl.DisplayName,
		Bio:         decl.Bio,
		SigilPNG:    decl.SigilPNG,
		PrivacyMode: decl.PrivacyMode,
		Timestamp:   p.UpdatedAt,
		Version:     p.Version,
	}

	if p.hasDisplayName {
		result.DisplayName = p.DisplayName
	}
	if p.hasBio {
		result.Bio = p.Bio
	}
	if p.hasSigil {
		result.SigilPNG = p.SigilPNG
	}
	if p.hasPrivacyMode {
		result.PrivacyMode = p.PrivacyMode
	}

	return result
}

// Marshal serializes the profile update to protobuf wire format.
func (p *ProfileUpdate) Marshal() ([]byte, error) {
	pbUpdate := &proto.ProfileUpdate{
		PublicKey:   p.PublicKey,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		SigilPng:    p.SigilPNG,
		PrivacyMode: modeToProto(p.PrivacyMode),
		UpdatedAt:   p.UpdatedAt,
		Version:     p.Version,
		Signature:   p.Signature,
	}
	return pb.Marshal(pbUpdate)
}

// UnmarshalProfileUpdate deserializes a profile update from protobuf wire format.
func UnmarshalProfileUpdate(data []byte) (*ProfileUpdate, error) {
	pbUpdate := &proto.ProfileUpdate{}
	if err := pb.Unmarshal(data, pbUpdate); err != nil {
		return nil, fmt.Errorf("unmarshaling profile update: %w", err)
	}

	update := &ProfileUpdate{
		PublicKey:   pbUpdate.PublicKey,
		DisplayName: pbUpdate.DisplayName,
		Bio:         pbUpdate.Bio,
		SigilPNG:    pbUpdate.SigilPng,
		PrivacyMode: protoToMode(pbUpdate.PrivacyMode),
		UpdatedAt:   pbUpdate.UpdatedAt,
		Version:     pbUpdate.Version,
		Signature:   pbUpdate.Signature,
	}

	// Mark non-empty fields as having been set.
	if pbUpdate.DisplayName != "" {
		update.hasDisplayName = true
	}
	if pbUpdate.Bio != "" {
		update.hasBio = true
	}
	if len(pbUpdate.SigilPng) > 0 {
		update.hasSigil = true
	}
	if pbUpdate.PrivacyMode != proto.PrivacyMode_PRIVACY_MODE_UNSPECIFIED {
		update.hasPrivacyMode = true
	}

	return update, nil
}
