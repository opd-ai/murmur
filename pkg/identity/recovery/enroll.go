package recovery

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/hashicorp/vault/shamir"
	"github.com/opd-ai/murmur/proto"
)

// EnrollRecoveryContacts splits the Master Key into N Shamir shares and distributes
// encrypted shares to contacts. Returns enrollment results for each contact.
func EnrollRecoveryContacts(
	masterPrivateKey ed25519.PrivateKey,
	masterPublicKey ed25519.PublicKey,
	x25519PrivateKey []byte,
	contacts []Contact,
	threshold, totalShares uint32,
	recoveryLabel string,
) ([]EnrollmentResult, error) {
	if err := validateEnrollmentParams(threshold, totalShares, len(contacts)); err != nil {
		return nil, err
	}

	shares, err := splitMasterKey(masterPrivateKey, int(totalShares), int(threshold))
	if err != nil {
		return nil, err
	}

	return distributeShares(shares, contacts, masterPrivateKey, masterPublicKey, x25519PrivateKey, threshold, totalShares, recoveryLabel), nil
}

// validateEnrollmentParams validates threshold and share parameters.
func validateEnrollmentParams(threshold, totalShares uint32, contactCount int) error {
	if threshold < MinThreshold || threshold > totalShares {
		return ErrInvalidThreshold
	}
	if totalShares > MaxTotalShares {
		return ErrInvalidShareCount
	}
	if int(totalShares) != contactCount {
		return fmt.Errorf("contacts count (%d) must match totalShares (%d)", contactCount, totalShares)
	}
	return nil
}

// splitMasterKey performs Shamir secret sharing on the master key.
func splitMasterKey(masterPrivateKey ed25519.PrivateKey, totalShares, threshold int) ([][]byte, error) {
	masterKeySeed := masterPrivateKey.Seed()
	shares, err := shamir.Split(masterKeySeed, totalShares, threshold)
	if err != nil {
		return nil, fmt.Errorf("Shamir split failed: %w", err)
	}
	return shares, nil
}

// distributeShares encrypts and signs shares for each contact.
func distributeShares(
	shares [][]byte,
	contacts []Contact,
	masterPrivateKey ed25519.PrivateKey,
	masterPublicKey ed25519.PublicKey,
	x25519PrivateKey []byte,
	threshold, totalShares uint32,
	recoveryLabel string,
) []EnrollmentResult {
	results := make([]EnrollmentResult, len(contacts))
	timestamp := time.Now().Unix()

	for i, contact := range contacts {
		results[i] = createEnrollmentForContact(
			contact, shares[i], uint32(i+1),
			masterPrivateKey, masterPublicKey, x25519PrivateKey,
			threshold, totalShares, timestamp, recoveryLabel,
		)
	}

	return results
}

// createEnrollmentForContact creates an enrollment result for a single contact.
func createEnrollmentForContact(
	contact Contact,
	share []byte,
	shareIndex uint32,
	masterPrivateKey ed25519.PrivateKey,
	masterPublicKey ed25519.PublicKey,
	x25519PrivateKey []byte,
	threshold, totalShares uint32,
	timestamp int64,
	recoveryLabel string,
) EnrollmentResult {
	result := EnrollmentResult{Contact: contact, ShareIndex: shareIndex}

	encryptedShare, nonce, err := encryptShareForContact(share, x25519PrivateKey, contact.X25519Key)
	if err != nil {
		result.Error = err
		return result
	}

	enrollment := buildEnrollmentProto(
		masterPublicKey, contact.PublicKey, encryptedShare, nonce,
		shareIndex, threshold, totalShares, timestamp, recoveryLabel,
	)

	signature, err := signEnrollment(enrollment, masterPrivateKey)
	if err != nil {
		result.Error = fmt.Errorf("signing failed: %w", err)
		return result
	}
	enrollment.EnrollmentSignature = signature

	result.Success = true
	result.Enrollment = enrollment
	return result
}

// encryptShareForContact encrypts a share using ECDH shared secret.
func encryptShareForContact(share, x25519PrivateKey, contactX25519Key []byte) ([]byte, []byte, error) {
	sharedSecret, err := deriveSharedSecret(x25519PrivateKey, contactX25519Key)
	if err != nil {
		return nil, nil, fmt.Errorf("ECDH failed: %w", err)
	}

	encryptedShare, nonce, err := encryptShare(share, sharedSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("encryption failed: %w", err)
	}

	return encryptedShare, nonce, nil
}

// buildEnrollmentProto constructs the protobuf enrollment message.
func buildEnrollmentProto(
	masterPublicKey, recipientPublicKey, encryptedShare, nonce []byte,
	shareIndex, threshold, totalShares uint32,
	timestamp int64,
	recoveryLabel string,
) *proto.RecoveryShareEnrollment {
	return &proto.RecoveryShareEnrollment{
		MasterPublicKey:    masterPublicKey,
		RecipientPublicKey: recipientPublicKey,
		EncryptedShare:     encryptedShare,
		Nonce:              nonce,
		ShareIndex:         shareIndex,
		Threshold:          threshold,
		TotalShares:        totalShares,
		TimestampUnix:      timestamp,
		RecoveryLabel:      recoveryLabel,
	}
}

// ValidateEnrollment validates an incoming enrollment message.
func ValidateEnrollment(enrollment *proto.RecoveryShareEnrollment) error {
	if err := validateTimestamp(enrollment.TimestampUnix); err != nil {
		return err
	}

	if enrollment.Threshold < MinThreshold {
		return ErrInvalidThreshold
	}
	if enrollment.Threshold > enrollment.TotalShares {
		return ErrInvalidThreshold
	}
	if enrollment.TotalShares > MaxTotalShares {
		return ErrInvalidShareCount
	}

	if err := verifyEnrollmentSignature(enrollment); err != nil {
		return err
	}

	return nil
}

// DecryptEnrollmentShare decrypts the share from an enrollment message.
func DecryptEnrollmentShare(
	enrollment *proto.RecoveryShareEnrollment,
	recipientX25519PrivateKey []byte,
	senderX25519PublicKey []byte,
) ([]byte, error) {
	sharedSecret, err := deriveSharedSecret(recipientX25519PrivateKey, senderX25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	share, err := decryptShare(enrollment.EncryptedShare, enrollment.Nonce, sharedSecret)
	if err != nil {
		return nil, err
	}

	return share, nil
}
