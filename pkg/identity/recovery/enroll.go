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
	if threshold < MinThreshold {
		return nil, ErrInvalidThreshold
	}
	if threshold > totalShares {
		return nil, ErrInvalidThreshold
	}
	if totalShares > MaxTotalShares {
		return nil, ErrInvalidShareCount
	}
	if int(totalShares) != len(contacts) {
		return nil, fmt.Errorf("contacts count (%d) must match totalShares (%d)", len(contacts), totalShares)
	}

	masterKeySeed := masterPrivateKey.Seed()
	shares, err := shamir.Split(masterKeySeed, int(totalShares), int(threshold))
	if err != nil {
		return nil, fmt.Errorf("Shamir split failed: %w", err)
	}

	results := make([]EnrollmentResult, len(contacts))
	timestamp := time.Now().Unix()

	for i, contact := range contacts {
		result := EnrollmentResult{
			Contact:    contact,
			ShareIndex: uint32(i + 1),
		}

		sharedSecret, err := deriveSharedSecret(x25519PrivateKey, contact.X25519Key)
		if err != nil {
			result.Error = fmt.Errorf("ECDH failed: %w", err)
			results[i] = result
			continue
		}

		encryptedShare, nonce, err := encryptShare(shares[i], sharedSecret)
		if err != nil {
			result.Error = fmt.Errorf("encryption failed: %w", err)
			results[i] = result
			continue
		}

		enrollment := &proto.RecoveryShareEnrollment{
			MasterPublicKey:    masterPublicKey,
			RecipientPublicKey: contact.PublicKey,
			EncryptedShare:     encryptedShare,
			Nonce:              nonce,
			ShareIndex:         uint32(i + 1),
			Threshold:          threshold,
			TotalShares:        totalShares,
			TimestampUnix:      timestamp,
			RecoveryLabel:      recoveryLabel,
		}

		signature, err := signEnrollment(enrollment, masterPrivateKey)
		if err != nil {
			result.Error = fmt.Errorf("signing failed: %w", err)
			results[i] = result
			continue
		}
		enrollment.EnrollmentSignature = signature

		result.Success = true
		result.Enrollment = enrollment
		results[i] = result
	}

	return results, nil
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
