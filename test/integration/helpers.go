//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/opd-ai/murmur/pkg/content/storage"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/networking/transport"
	"github.com/opd-ai/murmur/pkg/store"
	"github.com/stretchr/testify/require"
)

// TestNode represents a single MURMUR node in an integration test.
type TestNode struct {
	ID        int
	TempDir   string
	DB        *store.DB
	KeyPair   *keys.KeyPair
	Host      *transport.Host
	PubSub    *gossip.PubSub
	WaveCache *storage.Cache
	Context   context.Context
	Cancel    context.CancelFunc
}

// NewTestNode creates a new test node with all subsystems initialized.
func NewTestNode(t *testing.T, id int) *TestNode {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Open database
	dbPath := filepath.Join(tmpDir, fmt.Sprintf("node-%d.db", id))
	db, err := store.Open(dbPath)
	require.NoError(t, err, "opening database for node %d", id)

	// Generate keypair
	kp, err := keys.GenerateKeyPair()
	require.NoError(t, err, "generating keypair for node %d", id)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create libp2p host
	h, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: kp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0", // Bind to random port
		},
	})
	require.NoError(t, err, "creating host for node %d", id)

	// Create GossipSub
	ps, err := gossip.New(ctx, h)
	require.NoError(t, err, "creating pubsub for node %d", id)

	// Create Wave cache
	wc, err := storage.NewCache(db)
	require.NoError(t, err, "creating wave cache for node %d", id)

	node := &TestNode{
		ID:        id,
		TempDir:   tmpDir,
		DB:        db,
		KeyPair:   kp,
		Host:      h,
		PubSub:    ps,
		WaveCache: wc,
		Context:   ctx,
		Cancel:    cancel,
	}

	// Register cleanup
	t.Cleanup(func() {
		node.Close()
	})

	return node
}

// Close shuts down the test node and cleans up resources.
func (n *TestNode) Close() {
	if n.Cancel != nil {
		n.Cancel()
	}
	if n.Host != nil {
		n.Host.Close()
	}
	if n.DB != nil {
		n.DB.Close()
	}
}

// PeerID returns the libp2p peer ID for this node.
func (n *TestNode) PeerID() peer.ID {
	return n.Host.PeerID()
}

// Addrs returns the multiaddrs for this node.
func (n *TestNode) Addrs() []string {
	addrs := n.Host.Addrs()
	strs := make([]string, len(addrs))
	for i, addr := range addrs {
		strs[i] = addr.String()
	}
	return strs
}

// ConnectTo connects this node to another node.
func (n *TestNode) ConnectTo(t *testing.T, other *TestNode) {
	t.Helper()

	// Build peer AddrInfo
	addrInfo := peer.AddrInfo{
		ID:    other.PeerID(),
		Addrs: other.Host.Addrs(),
	}

	// Connect with timeout
	ctx, cancel := context.WithTimeout(n.Context, 5*time.Second)
	defer cancel()

	err := n.Host.Connect(ctx, addrInfo)
	require.NoError(t, err, "node %d connecting to node %d", n.ID, other.ID)
}

// WaitForPeers waits until this node is connected to the expected number of peers.
func (n *TestNode) WaitForPeers(t *testing.T, expected int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		peers := n.Host.Network().Peers()
		if len(peers) >= expected {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Timeout reached
	peers := n.Host.Network().Peers()
	require.FailNowf(t, "peer connection timeout",
		"node %d: expected %d peers, got %d after %v",
		n.ID, expected, len(peers), timeout)
}

// SubscribeWaves subscribes this node to the /murmur/waves/1 topic and returns a channel
// that receives incoming Wave envelopes.
func (n *TestNode) SubscribeWaves(t *testing.T) <-chan []byte {
	t.Helper()

	ch := make(chan []byte, 10)

	// Create handler that sends messages to channel
	handler := func(ctx context.Context, msg *pubsub.Message) {
		select {
		case ch <- msg.Data:
		case <-ctx.Done():
			return
		default:
			// Non-blocking send, drop message if channel is full
		}
	}

	err := n.PubSub.Subscribe(n.Context, "/murmur/waves/1", handler)
	require.NoError(t, err, "subscribing to /murmur/waves/1 for node %d", n.ID)

	return ch
}

// PublishWave publishes raw wave envelope bytes to the /murmur/waves/1 topic.
func (n *TestNode) PublishWave(t *testing.T, data []byte) {
	t.Helper()

	err := n.PubSub.Publish(n.Context, "/murmur/waves/1", data)
	require.NoError(t, err, "publishing wave from node %d", n.ID)
}

// WaitForMessage waits for a message to arrive on the channel within the timeout.
func WaitForMessage(t *testing.T, ch <-chan []byte, timeout time.Duration) []byte {
	t.Helper()

	select {
	case data := <-ch:
		return data
	case <-time.After(timeout):
		require.FailNow(t, "message receive timeout", "no message received within %v", timeout)
		return nil
	}
}

// ConnectMesh connects all nodes in a full mesh topology.
func ConnectMesh(t *testing.T, nodes []*TestNode) {
	t.Helper()

	for i, nodeA := range nodes {
		for j, nodeB := range nodes {
			if i < j {
				nodeA.ConnectTo(t, nodeB)
			}
		}
	}

	// Wait for all connections to stabilize
	for _, node := range nodes {
		node.WaitForPeers(t, len(nodes)-1, 10*time.Second)
	}
}

// WaitForGossipStability waits for GossipSub mesh to stabilize after connections.
// Per libp2p docs, mesh formation takes ~2 seconds after peer connections.
func WaitForGossipStability(t *testing.T) {
	t.Helper()
	time.Sleep(3 * time.Second)
}
