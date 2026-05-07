package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	crypto2 "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/opd-ai/murmur/pkg/networking/discovery"
)

func TestValidateConfig_DoesNotRequirePeerFile(t *testing.T) {
	t.Parallel()

	cfg := appConfig{
		stateDir:       t.TempDir(),
		p2pListenAddrs: defaultP2PListenAddrs,
		peerLimit:      defaultPeerLimit,
	}

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("validateConfig returned error: %v", err)
	}
}

func TestPeerTrackerSnapshotIncludesSelfAndPrunesStale(t *testing.T) {
	t.Parallel()

	tracker := newPeerTracker(time.Hour, 4)
	self := mustAddrInfo(t, 4001)
	recent := mustAddrInfo(t, 4002)
	stale := mustAddrInfo(t, 4003)

	tracker.Record(recent)
	tracker.mu.Lock()
	tracker.peers[stale.ID] = trackedPeer{
		addrs: map[string]multiaddr.Multiaddr{stale.Addrs[0].String(): stale.Addrs[0]},
		seen:  time.Now().Add(-2 * time.Hour),
	}
	tracker.mu.Unlock()

	snapshot := tracker.Snapshot(self)
	if len(snapshot) != 2 {
		t.Fatalf("expected self and recent peer, got %d entries", len(snapshot))
	}
	if snapshot[0].ID != self.ID {
		t.Fatalf("expected self to be first entry")
	}
	if snapshot[1].ID != recent.ID {
		t.Fatalf("expected recent peer to remain in snapshot")
	}
}

func TestBuildSignedPeerList_SignsDynamicPeers(t *testing.T) {
	t.Parallel()

	_, signKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	self := mustAddrInfo(t, 4001)
	other := mustAddrInfo(t, 4002)
	signedList, err := buildSignedPeerList(signKey, time.Hour, []peer.AddrInfo{self, other})
	if err != nil {
		t.Fatalf("buildSignedPeerList returned error: %v", err)
	}
	if len(signedList.Peers) != 2 {
		t.Fatalf("expected 2 peers in signed list, got %d", len(signedList.Peers))
	}
	if err := signedList.Verify(signKey.Public().(ed25519.PublicKey)); err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
}

func TestNewHandlerServesDynamicPeerList(t *testing.T) {
	t.Parallel()

	_, signKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	self := mustAddrInfo(t, 4001)
	signedList, err := buildSignedPeerList(signKey, time.Hour, []peer.AddrInfo{self})
	if err != nil {
		t.Fatalf("buildSignedPeerList returned error: %v", err)
	}

	handler := newHandler(fakeSource{
		signedList: signedList,
		health:     healthResponse{Status: "ok", PeerID: self.ID.String(), KnownPeers: 1},
	}, "*")

	request := httptest.NewRequest(http.MethodGet, "/peers.json", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var decoded discovery.SignedPeerList
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal peers.json: %v", err)
	}
	if len(decoded.Peers) != 1 {
		t.Fatalf("expected 1 peer in response, got %d", len(decoded.Peers))
	}
}

type fakeSource struct {
	signedList *discovery.SignedPeerList
	health     healthResponse
}

func (f fakeSource) SignedPeerList() (*discovery.SignedPeerList, error) {
	return f.signedList, nil
}

func (f fakeSource) HealthInfo() healthResponse {
	return f.health
}

func mustAddrInfo(t *testing.T, port int) peer.AddrInfo {
	t.Helper()
	priv, _, err := crypto2.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateEd25519Key: %v", err)
	}
	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		t.Fatalf("IDFromPrivateKey: %v", err)
	}
	addr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" + strconv.Itoa(port))
	if err != nil {
		t.Fatalf("NewMultiaddr: %v", err)
	}
	return peer.AddrInfo{ID: peerID, Addrs: []multiaddr.Multiaddr{addr}}
}
