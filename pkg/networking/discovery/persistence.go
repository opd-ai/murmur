// Package discovery provides peer routing table persistence.
// Per NETWORK_ARCHITECTURE.md, peer routing tables should persist across restarts
// for faster network rejoining.

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"go.etcd.io/bbolt"
)

// PeerTableBucket is the bucket name for storing peer routing table.
var PeerTableBucket = []byte("peers")

// PeerTableKey is the key for storing the peer list.
var PeerTableKey = []byte("routing_table")

// PeerRecord represents a persisted peer entry.
type PeerRecord struct {
	ID        string    `json:"id"`
	Addrs     []string  `json:"addrs"`
	LastSeen  time.Time `json:"last_seen"`
	Connected bool      `json:"connected"`
}

// PeerTable manages persistent peer routing table storage.
type PeerTable struct {
	h  host.Host
	db *bbolt.DB
	mu sync.RWMutex
}

// NewPeerTable creates a new peer table manager.
func NewPeerTable(h host.Host, db *bbolt.DB) (*PeerTable, error) {
	// Ensure bucket exists.
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(PeerTableBucket)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("creating peer bucket: %w", err)
	}

	return &PeerTable{
		h:  h,
		db: db,
	}, nil
}

// Save persists the current peer routing table to the database.
func (pt *PeerTable) Save(ctx context.Context) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	records := pt.collectPeerRecords()
	return pt.savePeerRecords(records)
}

// collectPeerRecords gathers peer information from the peerstore.
func (pt *PeerTable) collectPeerRecords() []PeerRecord {
	allPeers := pt.h.Peerstore().PeersWithAddrs()
	records := make([]PeerRecord, 0, len(allPeers))

	selfID := pt.h.ID()
	connectedPeers := make(map[peer.ID]bool)
	for _, p := range pt.h.Network().Peers() {
		connectedPeers[p] = true
	}

	for _, p := range allPeers {
		if p == selfID {
			continue
		}

		addrs := pt.h.Peerstore().Addrs(p)
		if len(addrs) == 0 {
			continue
		}

		addrStrs := make([]string, len(addrs))
		for i, addr := range addrs {
			addrStrs[i] = addr.String()
		}

		records = append(records, PeerRecord{
			ID:        p.String(),
			Addrs:     addrStrs,
			LastSeen:  time.Now(),
			Connected: connectedPeers[p],
		})
	}

	return records
}

// savePeerRecords writes peer records to the database.
func (pt *PeerTable) savePeerRecords(records []PeerRecord) error {
	data, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshaling peer records: %w", err)
	}

	return pt.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(PeerTableBucket)
		if b == nil {
			return fmt.Errorf("peer bucket not found")
		}
		return b.Put(PeerTableKey, data)
	})
}

// Load restores peers from the database into the peerstore.
func (pt *PeerTable) Load(ctx context.Context) ([]peer.AddrInfo, error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	records, err := pt.loadPeerRecords()
	if err != nil {
		return nil, err
	}

	return pt.restorePeers(records)
}

// loadPeerRecords reads peer records from the database.
func (pt *PeerTable) loadPeerRecords() ([]PeerRecord, error) {
	var records []PeerRecord

	err := pt.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(PeerTableBucket)
		if b == nil {
			return nil // No bucket means no peers saved yet
		}

		data := b.Get(PeerTableKey)
		if data == nil {
			return nil // No data saved yet
		}

		return json.Unmarshal(data, &records)
	})

	return records, err
}

// restorePeers adds loaded peers to the peerstore.
func (pt *PeerTable) restorePeers(records []PeerRecord) ([]peer.AddrInfo, error) {
	infos := make([]peer.AddrInfo, 0, len(records))

	for _, record := range records {
		info, err := pt.recordToAddrInfo(record)
		if err != nil {
			continue // Skip invalid records
		}

		// Add to peerstore with 24-hour TTL (will be refreshed on connection).
		pt.h.Peerstore().AddAddrs(info.ID, info.Addrs, 24*time.Hour)
		infos = append(infos, info)
	}

	return infos, nil
}

// recordToAddrInfo converts a PeerRecord to peer.AddrInfo.
func (pt *PeerTable) recordToAddrInfo(record PeerRecord) (peer.AddrInfo, error) {
	peerID, err := peer.Decode(record.ID)
	if err != nil {
		return peer.AddrInfo{}, fmt.Errorf("decoding peer ID: %w", err)
	}

	addrs := make([]multiaddr.Multiaddr, 0, len(record.Addrs))
	for _, addrStr := range record.Addrs {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			continue // Skip invalid addresses
		}
		addrs = append(addrs, addr)
	}

	if len(addrs) == 0 {
		return peer.AddrInfo{}, fmt.Errorf("no valid addresses")
	}

	return peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}, nil
}

// Clear removes all saved peers from the database.
func (pt *PeerTable) Clear() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	return pt.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(PeerTableBucket)
		if b == nil {
			return nil
		}
		return b.Delete(PeerTableKey)
	})
}

// Count returns the number of saved peers.
func (pt *PeerTable) Count() (int, error) {
	records, err := pt.loadPeerRecords()
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// PruneStale removes peers that haven't been seen in the given duration.
func (pt *PeerTable) PruneStale(maxAge time.Duration) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	records, err := pt.loadPeerRecords()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	fresh := make([]PeerRecord, 0, len(records))
	for _, record := range records {
		if record.LastSeen.After(cutoff) {
			fresh = append(fresh, record)
		}
	}

	return pt.savePeerRecords(fresh)
}

// PeriodicSave starts a goroutine that saves the peer table periodically.
// Returns a cancel function to stop the periodic saving.
func (pt *PeerTable) PeriodicSave(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Do one final save on shutdown.
				_ = pt.Save(context.Background())
				return
			case <-ticker.C:
				_ = pt.Save(ctx)
			}
		}
	}()

	return cancel
}
