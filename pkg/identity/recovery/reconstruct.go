package recovery

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/hashicorp/vault/shamir"
	"github.com/opd-ai/murmur/proto"
)

// ReconstructMasterKey reconstructs the Master Private Key from M shares.
// Verifies the reconstructed key matches the expected public key.
func ReconstructMasterKey(
	responses []*proto.RecoveryResponse,
	expectedPublicKey ed25519.PublicKey,
	requesterX25519PrivateKey []byte,
	contactX25519PublicKeys map[uint32][]byte,
) (*RecoveryResult, error) {
	if len(responses) == 0 {
		return &RecoveryResult{
			Success: false,
			Error:   ErrNotEnoughShares,
		}, nil
	}

	shares := make([][]byte, len(responses))
	sharesUsed := make([]uint32, len(responses))

	for i, resp := range responses {
		contactX25519Key, ok := contactX25519PublicKeys[resp.ShareIndex]
		if !ok {
			return &RecoveryResult{
				Success: false,
				Error:   fmt.Errorf("missing X25519 key for share index %d", resp.ShareIndex),
			}, nil
		}

		sharedSecret, err := deriveSharedSecret(requesterX25519PrivateKey, contactX25519Key)
		if err != nil {
			return &RecoveryResult{
				Success: false,
				Error:   fmt.Errorf("ECDH failed for share %d: %w", resp.ShareIndex, err),
			}, nil
		}

		share, err := decryptShare(resp.EncryptedShare, resp.Nonce, sharedSecret)
		if err != nil {
			return &RecoveryResult{
				Success: false,
				Error:   fmt.Errorf("decryption failed for share %d: %w", resp.ShareIndex, err),
			}, nil
		}

		shares[i] = share
		sharesUsed[i] = resp.ShareIndex
	}

	masterKeySeed, err := shamir.Combine(shares)
	if err != nil {
		return &RecoveryResult{
			Success: false,
			Error:   ErrReconstructionFailed,
		}, nil
	}

	reconstructedKey := ed25519.NewKeyFromSeed(masterKeySeed)
	reconstructedPublicKey := reconstructedKey.Public().(ed25519.PublicKey)

	if !reconstructedPublicKey.Equal(expectedPublicKey) {
		return &RecoveryResult{
			Success: false,
			Error:   ErrInvalidMasterKey,
		}, nil
	}

	return &RecoveryResult{
		MasterKey:  reconstructedKey,
		SharesUsed: sharesUsed,
		Success:    true,
	}, nil
}

// CreateRecoveryRequest generates a recovery request message.
func CreateRecoveryRequest(
	masterPublicKey ed25519.PublicKey,
	requesterPrivateKey ed25519.PrivateKey,
	recoveryLabel string,
) (*proto.RecoveryRequest, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	req := &proto.RecoveryRequest{
		MasterPublicKey:    masterPublicKey,
		RequesterPublicKey: requesterPrivateKey.Public().(ed25519.PublicKey),
		ChallengeNonce:     nonce,
		TimestampUnix:      time.Now().Unix(),
		RecoveryLabel:      recoveryLabel,
	}

	signature := signRecoveryRequest(req, requesterPrivateKey)
	req.RequestSignature = signature

	return req, nil
}

// signRecoveryRequest signs a recovery request.
func signRecoveryRequest(req *proto.RecoveryRequest, privateKey ed25519.PrivateKey) []byte {
	data := make([]byte, 0, len(req.MasterPublicKey)+len(req.RequesterPublicKey)+len(req.ChallengeNonce)+8)
	data = append(data, req.MasterPublicKey...)
	data = append(data, req.RequesterPublicKey...)
	data = append(data, req.ChallengeNonce...)

	tsBuf := make([]byte, 8)
	ts := uint64(req.TimestampUnix)
	for i := 0; i < 8; i++ {
		tsBuf[7-i] = byte(ts >> (i * 8))
	}
	data = append(data, tsBuf...)

	return ed25519.Sign(privateKey, data)
}

// ValidateRecoveryRequest validates an incoming recovery request.
func ValidateRecoveryRequest(req *proto.RecoveryRequest) error {
	if err := validateTimestamp(req.TimestampUnix); err != nil {
		return err
	}

	data := make([]byte, 0, len(req.MasterPublicKey)+len(req.RequesterPublicKey)+len(req.ChallengeNonce)+8)
	data = append(data, req.MasterPublicKey...)
	data = append(data, req.RequesterPublicKey...)
	data = append(data, req.ChallengeNonce...)

	tsBuf := make([]byte, 8)
	ts := uint64(req.TimestampUnix)
	for i := 0; i < 8; i++ {
		tsBuf[7-i] = byte(ts >> (i * 8))
	}
	data = append(data, tsBuf...)

	if !ed25519.Verify(req.RequesterPublicKey, data, req.RequestSignature) {
		return ErrInvalidSignature
	}

	return nil
}

// CreateRecoveryResponse creates a response to a recovery request.
func CreateRecoveryResponse(
	masterPublicKey ed25519.PublicKey,
	share []byte,
	shareIndex uint32,
	requesterX25519PublicKey []byte,
	contactX25519PrivateKey []byte,
	contactEd25519PrivateKey ed25519.PrivateKey,
) (*proto.RecoveryResponse, error) {
	sharedSecret, err := deriveSharedSecret(contactX25519PrivateKey, requesterX25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	encryptedShare, nonce, err := encryptShare(share, sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	resp := &proto.RecoveryResponse{
		MasterPublicKey: masterPublicKey,
		EncryptedShare:  encryptedShare,
		Nonce:           nonce,
		ShareIndex:      shareIndex,
		TimestampUnix:   time.Now().Unix(),
	}

	signature := signRecoveryResponse(resp, contactEd25519PrivateKey)
	resp.ContactSignature = signature

	return resp, nil
}

// signRecoveryResponse signs a recovery response.
func signRecoveryResponse(resp *proto.RecoveryResponse, privateKey ed25519.PrivateKey) []byte {
	data := make([]byte, 0, len(resp.MasterPublicKey)+len(resp.EncryptedShare)+len(resp.Nonce)+12)
	data = append(data, resp.MasterPublicKey...)
	data = append(data, resp.EncryptedShare...)
	data = append(data, resp.Nonce...)

	buf := make([]byte, 4)
	buf[0] = byte(resp.ShareIndex >> 24)
	buf[1] = byte(resp.ShareIndex >> 16)
	buf[2] = byte(resp.ShareIndex >> 8)
	buf[3] = byte(resp.ShareIndex)
	data = append(data, buf...)

	tsBuf := make([]byte, 8)
	ts := uint64(resp.TimestampUnix)
	for i := 0; i < 8; i++ {
		tsBuf[7-i] = byte(ts >> (i * 8))
	}
	data = append(data, tsBuf...)

	return ed25519.Sign(privateKey, data)
}
