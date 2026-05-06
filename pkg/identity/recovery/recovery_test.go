package recovery

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/hashicorp/vault/shamir"
	pb "github.com/opd-ai/murmur/proto"
	"golang.org/x/crypto/curve25519"
	"google.golang.org/protobuf/proto"
)

func TestShamirSplitCombine(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	t.Run("3-of-5 reconstruction with any 3 shares", func(t *testing.T) {
		shares, err := shamir.Split(secret, 5, 3)
		if err != nil {
			t.Fatalf("shamir.Split failed: %v", err)
		}
		if len(shares) != 5 {
			t.Fatalf("expected 5 shares, got %d", len(shares))
		}

		for i := 0; i < 3; i++ {
			subset := [][]byte{shares[i], shares[(i+1)%5], shares[(i+2)%5]}
			reconstructed, err := shamir.Combine(subset)
			if err != nil {
				t.Fatalf("shamir.Combine failed: %v", err)
			}
			if string(reconstructed) != string(secret) {
				t.Errorf("reconstructed secret mismatch (subset %d)", i)
			}
		}
	})

	t.Run("M-1 shares cannot reconstruct (returns garbage)", func(t *testing.T) {
		shares, err := shamir.Split(secret, 5, 3)
		if err != nil {
			t.Fatal(err)
		}

		reconstructed, err := shamir.Combine(shares[:2])
		if err != nil {
			t.Skip("Shamir.Combine with M-1 shares may error or return garbage (library-dependent)")
		}
		if string(reconstructed) == string(secret) {
			t.Error("M-1 shares should not reconstruct the correct secret")
		}
	})

	t.Run("2-of-3 reconstruction", func(t *testing.T) {
		shares, err := shamir.Split(secret, 3, 2)
		if err != nil {
			t.Fatal(err)
		}

		reconstructed, err := shamir.Combine(shares[:2])
		if err != nil {
			t.Fatalf("shamir.Combine failed: %v", err)
		}
		if string(reconstructed) != string(secret) {
			t.Error("reconstructed secret mismatch")
		}
	})
}

func TestEnrollRecoveryContacts(t *testing.T) {
	masterPub, masterPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	x25519Priv := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(x25519Priv); err != nil {
		t.Fatal(err)
	}

	contacts := make([]Contact, 5)
	for i := range contacts {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		x25519Pub := make([]byte, curve25519.PointSize)
		if _, err := rand.Read(x25519Pub); err != nil {
			t.Fatal(err)
		}
		contacts[i] = Contact{
			PublicKey: pub,
			X25519Key: x25519Pub,
			Label:     "Contact",
		}
	}

	results, err := EnrollRecoveryContacts(
		masterPriv,
		masterPub,
		x25519Priv,
		contacts,
		3,
		5,
		"Test Recovery",
	)
	if err != nil {
		t.Fatalf("EnrollRecoveryContacts failed: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	for i, res := range results {
		if !res.Success {
			t.Errorf("enrollment %d failed: %v", i, res.Error)
		}
		if res.Enrollment == nil {
			t.Errorf("enrollment %d has nil Enrollment", i)
			continue
		}

		if err := ValidateEnrollment(res.Enrollment); err != nil {
			t.Errorf("enrollment %d validation failed: %v", i, err)
		}

		if res.Enrollment.Threshold != 3 {
			t.Errorf("enrollment %d: expected threshold 3, got %d", i, res.Enrollment.Threshold)
		}
		if res.Enrollment.TotalShares != 5 {
			t.Errorf("enrollment %d: expected total_shares 5, got %d", i, res.Enrollment.TotalShares)
		}
		if res.Enrollment.ShareIndex != uint32(i+1) {
			t.Errorf("enrollment %d: expected share_index %d, got %d", i, i+1, res.Enrollment.ShareIndex)
		}
	}
}

func TestValidateEnrollment(t *testing.T) {
	t.Run("valid enrollment passes", func(t *testing.T) {
		masterPub, masterPriv, _ := ed25519.GenerateKey(rand.Reader)
		x25519Priv := make([]byte, curve25519.ScalarSize)
		rand.Read(x25519Priv)

		pub, _, _ := ed25519.GenerateKey(rand.Reader)
		x25519Pub := make([]byte, curve25519.PointSize)
		rand.Read(x25519Pub)

		pub2, _, _ := ed25519.GenerateKey(rand.Reader)
		x25519Pub2 := make([]byte, curve25519.PointSize)
		rand.Read(x25519Pub2)

		contacts := []Contact{
			{PublicKey: pub, X25519Key: x25519Pub},
			{PublicKey: pub2, X25519Key: x25519Pub2},
		}

		results, err := EnrollRecoveryContacts(masterPriv, masterPub, x25519Priv, contacts, 2, 2, "Test")
		if err != nil {
			t.Fatalf("EnrollRecoveryContacts error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for i, res := range results {
			if !res.Success {
				t.Fatalf("enrollment %d failed: %v", i, res.Error)
			}
		}

		err = ValidateEnrollment(results[0].Enrollment)
		if err != nil {
			t.Errorf("expected valid enrollment, got error: %v", err)
		}
	})

	t.Run("invalid threshold fails", func(t *testing.T) {
		enrollment := &pb.RecoveryShareEnrollment{
			Threshold:     1,
			TotalShares:   2,
			TimestampUnix: time.Now().Unix(),
		}
		err := ValidateEnrollment(enrollment)
		if err != ErrInvalidThreshold {
			t.Errorf("expected ErrInvalidThreshold, got %v", err)
		}
	})

	t.Run("threshold > total_shares fails", func(t *testing.T) {
		enrollment := &pb.RecoveryShareEnrollment{
			Threshold:     5,
			TotalShares:   3,
			TimestampUnix: time.Now().Unix(),
		}
		err := ValidateEnrollment(enrollment)
		if err != ErrInvalidThreshold {
			t.Errorf("expected ErrInvalidThreshold, got %v", err)
		}
	})
}

func TestReconstructMasterKey(t *testing.T) {
	masterPub, masterPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	userX25519Priv := make([]byte, curve25519.ScalarSize)
	rand.Read(userX25519Priv)
	userX25519Pub, _ := curve25519.X25519(userX25519Priv, curve25519.Basepoint)

	contacts := make([]Contact, 5)
	contactX25519Privs := make([][]byte, 5)
	contactEd25519Privs := make([]ed25519.PrivateKey, 5)

	for i := range contacts {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		x25519Priv := make([]byte, curve25519.ScalarSize)
		rand.Read(x25519Priv)
		x25519Pub, _ := curve25519.X25519(x25519Priv, curve25519.Basepoint)

		contacts[i] = Contact{PublicKey: pub, X25519Key: x25519Pub}
		contactX25519Privs[i] = x25519Priv
		contactEd25519Privs[i] = priv
	}

	results, err := EnrollRecoveryContacts(masterPriv, masterPub, userX25519Priv, contacts, 3, 5, "Test")
	if err != nil {
		t.Fatal(err)
	}

	shares := make([][]byte, 5)
	for i, res := range results {
		share, err := DecryptEnrollmentShare(res.Enrollment, contactX25519Privs[i], userX25519Pub)
		if err != nil {
			t.Fatalf("share %d decryption failed: %v", i, err)
		}
		shares[i] = share
	}

	t.Run("reconstruct with 3 shares", func(t *testing.T) {
		responses := make([]*pb.RecoveryResponse, 3)
		contactX25519Keys := make(map[uint32][]byte)

		for i := 0; i < 3; i++ {
			resp, err := CreateRecoveryResponse(
				masterPub,
				shares[i],
				uint32(i+1),
				userX25519Pub,
				contactX25519Privs[i],
				contactEd25519Privs[i],
			)
			if err != nil {
				t.Fatal(err)
			}
			responses[i] = resp
			contactX25519Keys[uint32(i+1)] = contacts[i].X25519Key
		}

		result, err := ReconstructMasterKey(responses, masterPub, userX25519Priv, contactX25519Keys)
		if err != nil {
			t.Fatalf("ReconstructMasterKey failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("reconstruction failed: %v", result.Error)
		}

		reconstructedPriv := ed25519.PrivateKey(result.MasterKey)
		reconstructedPub := reconstructedPriv.Public().(ed25519.PublicKey)
		if !reconstructedPub.Equal(masterPub) {
			t.Error("reconstructed public key does not match")
		}
	})

	t.Run("reconstruction with 2 shares fails", func(t *testing.T) {
		responses := make([]*pb.RecoveryResponse, 2)
		contactX25519Keys := make(map[uint32][]byte)

		for i := 0; i < 2; i++ {
			resp, _ := CreateRecoveryResponse(
				masterPub,
				shares[i],
				uint32(i+1),
				userX25519Pub,
				contactX25519Privs[i],
				contactEd25519Privs[i],
			)
			responses[i] = resp
			contactX25519Keys[uint32(i+1)] = contacts[i].X25519Key
		}

		result, err := ReconstructMasterKey(responses, masterPub, userX25519Priv, contactX25519Keys)
		if err != nil {
			t.Fatal(err)
		}
		if result.Success {
			t.Error("expected reconstruction to fail with M-1 shares")
		}
	})
}

func TestRecoveryRequestValidation(t *testing.T) {
	masterPub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, requesterPriv, _ := ed25519.GenerateKey(rand.Reader)

	req, err := CreateRecoveryRequest(masterPub, requesterPriv, "Test")
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateRecoveryRequest(req); err != nil {
		t.Errorf("valid request failed validation: %v", err)
	}

	tamperedReq := proto.Clone(req).(*pb.RecoveryRequest)
	tamperedReq.ChallengeNonce = make([]byte, 32)
	rand.Read(tamperedReq.ChallengeNonce)

	if err := ValidateRecoveryRequest(tamperedReq); err == nil {
		t.Error("tampered request passed validation")
	}
}
