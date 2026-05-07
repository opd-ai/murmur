package bootstrap

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/opd-ai/murmur/pkg/identity"
)

func TestAcceptInvitation(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := identity.GenerateInvitation(peerID, pub, "Welcome!")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	uri, err := inv.EncodeURI()
	if err != nil {
		t.Fatalf("encoding URI: %v", err)
	}

	// Accept the invitation.
	accepted, err := AcceptInvitation(uri)
	if err != nil {
		t.Fatalf("accepting invitation: %v", err)
	}

	// Verify fields match.
	if accepted.PeerID != inv.PeerID {
		t.Errorf("peer ID mismatch: got %v, want %v", accepted.PeerID, inv.PeerID)
	}
	if string(accepted.PublicKey) != string(inv.PublicKey) {
		t.Error("public key mismatch")
	}
	if accepted.WelcomeMessage != inv.WelcomeMessage {
		t.Errorf("welcome message mismatch: got %q, want %q", accepted.WelcomeMessage, inv.WelcomeMessage)
	}
}

func TestAcceptInvitation_EmptyURI(t *testing.T) {
	_, err := AcceptInvitation("")
	if err == nil {
		t.Error("expected error for empty URI, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' error, got %v", err)
	}
}

func TestAcceptInvitation_InvalidURI(t *testing.T) {
	_, err := AcceptInvitation("murmur://invite/invalid-base64")
	if err == nil {
		t.Error("expected error for invalid URI, got nil")
	}
}

func TestAcceptInvitation_InvalidInvitation(t *testing.T) {
	// Create an invitation with invalid public key size.
	uri := "murmur://invite/" + "AAAA" // Too short to be a valid invitation

	_, err := AcceptInvitation(uri)
	if err == nil {
		t.Error("expected error for invalid invitation, got nil")
	}
}

func TestBuildBootstrapAddrFromInvitation(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := identity.GenerateInvitation(peerID, pub, "")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Build bootstrap address.
	addr := BuildBootstrapAddrFromInvitation(inv)

	// Verify address format.
	if !strings.HasPrefix(addr, "/p2p/") {
		t.Errorf("expected /p2p/ prefix, got %q", addr)
	}

	// Verify peer ID is included.
	if !strings.Contains(addr, peerID.String()) {
		t.Errorf("address missing peer ID: %q", addr)
	}
}

func TestAcceptInvitation_RoundTrip(t *testing.T) {
	// Generate invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	original, err := identity.GenerateInvitation(peerID, pub, "Hello, MURMUR!")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Encode to URI.
	uri, err := original.EncodeURI()
	if err != nil {
		t.Fatalf("encoding URI: %v", err)
	}

	// Accept (decode) the invitation.
	accepted, err := AcceptInvitation(uri)
	if err != nil {
		t.Fatalf("accepting invitation: %v", err)
	}

	// Build bootstrap address.
	addr := BuildBootstrapAddrFromInvitation(accepted)

	// Verify the address can be used for bootstrap.
	if addr == "" {
		t.Error("bootstrap address is empty")
	}

	// Verify peer ID round-trips correctly.
	if accepted.PeerID != original.PeerID {
		t.Error("peer ID did not round-trip correctly")
	}
}

func TestBuildBootstrapAddrsFromInvitation(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv := &identity.Invitation{
		PeerID:         peerID,
		PublicKey:      pub,
		BootstrapAddrs: []string{"  /ip4/127.0.0.1/tcp/4100/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp  ", ""},
	}

	addrs := BuildBootstrapAddrsFromInvitation(inv)
	if len(addrs) != 1 {
		t.Fatalf("expected one cleaned bootstrap address, got %d", len(addrs))
	}
	if addrs[0] != "/ip4/127.0.0.1/tcp/4100/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp" {
		t.Fatalf("unexpected bootstrap address: %q", addrs[0])
	}
}

func TestAcceptInvitationSigned(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := identity.GenerateSignedInvitation(peerID, pub, priv, identity.SignedInvitationOptions{
		BootstrapAddrs: []string{"/ip4/127.0.0.1/tcp/4101/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp"},
		WelcomeMessage: "Warm start",
		TTL:            5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("generating signed invitation: %v", err)
	}

	uri, err := inv.EncodeURI()
	if err != nil {
		t.Fatalf("encoding signed URI: %v", err)
	}

	accepted, err := AcceptInvitation(uri)
	if err != nil {
		t.Fatalf("accepting signed invitation: %v", err)
	}

	addrs := BuildBootstrapAddrsFromInvitation(accepted)
	if len(addrs) != 1 || addrs[0] != inv.BootstrapAddrs[0] {
		t.Fatalf("signed invitation did not preserve bootstrap addresses")
	}
}
