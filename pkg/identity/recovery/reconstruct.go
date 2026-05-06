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
		return newRecoveryResult(false, ErrNotEnoughShares, nil, nil), nil
	}

	shares, sharesUsed, err := decryptReceivedShares(responses, requesterX25519PrivateKey, contactX25519PublicKeys)
	if err != nil {
		return newRecoveryResult(false, err, nil, nil), nil
	}

	masterKey, err := combineSharesToMasterKey(shares, expectedPublicKey)
	if err != nil {
		return newRecoveryResult(false, err, nil, nil), nil
	}

	return newRecoveryResult(true, nil, masterKey, sharesUsed), nil
}

// newRecoveryResult constructs a RecoveryResult with given parameters.
func newRecoveryResult(success bool, err error, masterKey ed25519.PrivateKey, sharesUsed []uint32) *RecoveryResult {
	return &RecoveryResult{
		Success:    success,
		Error:      err,
		MasterKey:  masterKey,
		SharesUsed: sharesUsed,
	}
}

// decryptReceivedShares decrypts all received shares using ECDH.
func decryptReceivedShares(
	responses []*proto.RecoveryResponse,
	requesterX25519PrivateKey []byte,
	contactX25519PublicKeys map[uint32][]byte,
) ([][]byte, []uint32, error) {
	shares := make([][]byte, len(responses))
	sharesUsed := make([]uint32, len(responses))

	for i, resp := range responses {
		share, err := decryptSingleShare(resp, requesterX25519PrivateKey, contactX25519PublicKeys)
		if err != nil {
			return nil, nil, err
		}
		shares[i] = share
		sharesUsed[i] = resp.ShareIndex
	}

	return shares, sharesUsed, nil
}

// decryptSingleShare decrypts a single recovery response share.
func decryptSingleShare(
	resp *proto.RecoveryResponse,
	requesterX25519PrivateKey []byte,
	contactX25519PublicKeys map[uint32][]byte,
) ([]byte, error) {
	contactX25519Key, ok := contactX25519PublicKeys[resp.ShareIndex]
	if !ok {
		return nil, fmt.Errorf("missing X25519 key for share index %d", resp.ShareIndex)
	}

	sharedSecret, err := deriveSharedSecret(requesterX25519PrivateKey, contactX25519Key)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed for share %d: %w", resp.ShareIndex, err)
	}

	share, err := decryptShare(resp.EncryptedShare, resp.Nonce, sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("decryption failed for share %d: %w", resp.ShareIndex, err)
	}

	return share, nil
}

// combineSharesToMasterKey combines Shamir shares and verifies the result.
func combineSharesToMasterKey(shares [][]byte, expectedPublicKey ed25519.PublicKey) (ed25519.PrivateKey, error) {
	masterKeySeed, err := shamir.Combine(shares)
	if err != nil {
		return nil, ErrReconstructionFailed
	}

	reconstructedKey := ed25519.NewKeyFromSeed(masterKeySeed)
	reconstructedPublicKey := reconstructedKey.Public().(ed25519.PublicKey)

	if !reconstructedPublicKey.Equal(expectedPublicKey) {
		return nil, ErrInvalidMasterKey
	}

	return reconstructedKey, nil
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

// buildRecoveryRequestData serializes recovery request fields for signing/verification.
func buildRecoveryRequestData(req *proto.RecoveryRequest) []byte {
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

	return data
}

// signRecoveryRequest signs a recovery request.
func signRecoveryRequest(req *proto.RecoveryRequest, privateKey ed25519.PrivateKey) []byte {
	data := buildRecoveryRequestData(req)
	return ed25519.Sign(privateKey, data)
}

// ValidateRecoveryRequest validates an incoming recovery request.
func ValidateRecoveryRequest(req *proto.RecoveryRequest) error {
	if err := validateTimestamp(req.TimestampUnix); err != nil {
		return err
	}

	data := buildRecoveryRequestData(req)

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
