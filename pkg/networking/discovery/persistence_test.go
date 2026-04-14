package discovery

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestNewPeerTable(t *testing.T) {
	h, db, cleanup := setupPeerTableTest(t)
	defer cleanup()

	pt, err := NewPeerTable(h, db)
	require.NoError(t, err)
	require.NotNil(t, pt)
}

func TestPeerTable_SaveAndLoad(t *testing.T) {
	h, db, cleanup := setupPeerTableTest(t)
	defer cleanup()

	pt, err := NewPeerTable(h, db)
	require.NoError(t, err)

	ctx := context.Background()

	// Initially no peers.
	peers, err := pt.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, peers, 0)

	// Add some peers to peerstore.
	for i := 0; i < 3; i++ {
		peerID := generatePersistencePeerID(t)
		addr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
		h.Peerstore().AddAddrs(peerID, []multiaddr.Multiaddr{addr}, time.Hour)
	}

	// Save.
	err = pt.Save(ctx)
	require.NoError(t, err)

	// Check count.
	count, err := pt.Count()
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Load should restore the saved peers (same host, same db).
	peers, err = pt.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, peers, 3)
}

func TestPeerTable_Clear(t *testing.T) {
	h, db, cleanup := setupPeerTableTest(t)
	defer cleanup()

	pt, err := NewPeerTable(h, db)
	require.NoError(t, err)

	ctx := context.Background()

	// Add some peers.
	peerID := generatePersistencePeerID(t)
	addr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	h.Peerstore().AddAddrs(peerID, []multiaddr.Multiaddr{addr}, time.Hour)

	err = pt.Save(ctx)
	require.NoError(t, err)

	count, err := pt.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Clear.
	err = pt.Clear()
	require.NoError(t, err)

	count, err = pt.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPeerTable_PruneStale(t *testing.T) {
	h, db, cleanup := setupPeerTableTest(t)
	defer cleanup()

	pt, err := NewPeerTable(h, db)
	require.NoError(t, err)

	ctx := context.Background()

	// Add peers.
	peerID := generatePersistencePeerID(t)
	addr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	h.Peerstore().AddAddrs(peerID, []multiaddr.Multiaddr{addr}, time.Hour)

	err = pt.Save(ctx)
	require.NoError(t, err)

	// Prune with very short max age (everything should be pruned).
	err = pt.PruneStale(time.Nanosecond)
	require.NoError(t, err)

	count, err := pt.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPeerTable_ExcludesSelf(t *testing.T) {
	h, db, cleanup := setupPeerTableTest(t)
	defer cleanup()

	pt, err := NewPeerTable(h, db)
	require.NoError(t, err)

	ctx := context.Background()

	// Add self to peerstore.
	addr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001")
	h.Peerstore().AddAddrs(h.ID(), []multiaddr.Multiaddr{addr}, time.Hour)

	// Add one other peer.
	peerID := generatePersistencePeerID(t)
	h.Peerstore().AddAddrs(peerID, []multiaddr.Multiaddr{addr}, time.Hour)

	err = pt.Save(ctx)
	require.NoError(t, err)

	// Should only have one peer (not self).
	count, err := pt.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPeerRecord_JSON(t *testing.T) {
	record := PeerRecord{
		ID:        "12D3KooWTestPeer",
		Addrs:     []string{"/ip4/192.168.1.1/tcp/4001"},
		LastSeen:  time.Now(),
		Connected: true,
	}

	assert.NotEmpty(t, record.ID)
	assert.NotEmpty(t, record.Addrs)
}

// Helper functions.

func setupPeerTableTest(t *testing.T) (host.Host, *bbolt.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "peertable-test-*")
	require.NoError(t, err)

	return setupPeerTableTestWithDB(t, tmpDir)
}

func setupPeerTableTestWithDB(t *testing.T, dir string) (host.Host, *bbolt.DB, func()) {
	t.Helper()

	dbPath := filepath.Join(dir, "test.db")
	db, err := bbolt.Open(dbPath, 0o600, nil)
	require.NoError(t, err)

	h := createPersistenceTestHost(t)

	cleanup := func() {
		h.Close()
		db.Close()
		os.RemoveAll(dir)
	}

	return h, db, cleanup
}

func createPersistenceTestHost(t *testing.T) host.Host {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(t, err)

	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	return h
}

func generatePersistencePeerID(t *testing.T) peer.ID {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(t, err)

	peerID, err := peer.IDFromPrivateKey(privKey)
	require.NoError(t, err)

	return peerID
}
