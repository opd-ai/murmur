// Package threads provides reply chain indexing and conversation reconstruction.
// Per DESIGN_DOCUMENT.md, replies reference parent Waves by hash.
package threads

import (
	"errors"
	"sync"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// MaxThreadDepth is the maximum depth of reply chains.
const MaxThreadDepth = 100

// Errors for thread operations.
var (
	ErrNotFound     = errors.New("thread not found")
	ErrNilStore     = errors.New("store is nil")
	ErrInvalidWave  = errors.New("invalid wave")
	ErrNoParent     = errors.New("wave has no parent")
	ErrCyclicThread = errors.New("cyclic thread detected")
	ErrMaxDepth     = errors.New("maximum thread depth exceeded")
)

// Index maintains reply chain relationships for Wave threading.
type Index struct {
	mu      sync.RWMutex
	db      *store.DB
	replies map[string][]string // parent_id -> child_ids
	parents map[string]string   // child_id -> parent_id
}

// NewIndex creates a new thread index with the given database.
func NewIndex(db *store.DB) (*Index, error) {
	if db == nil {
		return nil, ErrNilStore
	}

	return &Index{
		db:      db,
		replies: make(map[string][]string),
		parents: make(map[string]string),
	}, nil
}

// Add indexes a Wave's reply relationship.
func (idx *Index) Add(wave *pb.Wave) error {
	if wave == nil || len(wave.WaveId) == 0 {
		return ErrInvalidWave
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	waveID := string(wave.WaveId)

	// If this is a reply, record the parent relationship.
	if len(wave.ParentHash) > 0 {
		parentID := string(wave.ParentHash)

		// Check for cycles.
		if parentID == waveID {
			return ErrCyclicThread
		}

		// Record parent -> child relationship.
		idx.replies[parentID] = append(idx.replies[parentID], waveID)
		idx.parents[waveID] = parentID

		// Persist to database.
		if err := idx.persistReply(wave.ParentHash, wave.WaveId); err != nil {
			return err
		}
	}

	return nil
}

// persistReply saves a reply relationship to the database.
func (idx *Index) persistReply(parentID, childID []byte) error {
	// Store as key: "reply:<parent_id>:<child_id>" -> empty value
	key := append([]byte("reply:"), parentID...)
	key = append(key, ':')
	key = append(key, childID...)

	return idx.db.Put(store.BucketThreads, key, []byte{1})
}

// GetReplies returns all direct replies to a Wave.
func (idx *Index) GetReplies(waveID []byte) [][]byte {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	replies := idx.replies[string(waveID)]
	result := make([][]byte, len(replies))
	for i, r := range replies {
		result[i] = []byte(r)
	}
	return result
}

// GetParent returns the parent Wave ID for a reply.
func (idx *Index) GetParent(waveID []byte) ([]byte, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	parent, ok := idx.parents[string(waveID)]
	if !ok {
		return nil, ErrNoParent
	}
	return []byte(parent), nil
}

// GetThread reconstructs the full reply chain for a Wave.
// Returns the chain from root to the given Wave.
func (idx *Index) GetThread(waveID []byte) ([][]byte, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	chain := [][]byte{waveID}
	current := string(waveID)
	seen := make(map[string]bool)
	seen[current] = true

	for depth := 0; depth < MaxThreadDepth; depth++ {
		parent, ok := idx.parents[current]
		if !ok {
			break
		}

		if seen[parent] {
			return nil, ErrCyclicThread
		}
		seen[parent] = true

		chain = append([][]byte{[]byte(parent)}, chain...)
		current = parent
	}

	if len(chain) >= MaxThreadDepth {
		return nil, ErrMaxDepth
	}

	return chain, nil
}

// GetDepth returns the depth of a Wave in its thread (0 = root).
func (idx *Index) GetDepth(waveID []byte) (int, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	depth := 0
	current := string(waveID)
	seen := make(map[string]bool)
	seen[current] = true

	for depth < MaxThreadDepth {
		parent, ok := idx.parents[current]
		if !ok {
			break
		}

		if seen[parent] {
			return 0, ErrCyclicThread
		}
		seen[parent] = true

		depth++
		current = parent
	}

	return depth, nil
}

// GetRoot returns the root Wave ID of a thread.
func (idx *Index) GetRoot(waveID []byte) ([]byte, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	current := string(waveID)
	seen := make(map[string]bool)
	seen[current] = true

	for depth := 0; depth < MaxThreadDepth; depth++ {
		parent, ok := idx.parents[current]
		if !ok {
			return []byte(current), nil
		}

		if seen[parent] {
			return nil, ErrCyclicThread
		}
		seen[parent] = true

		current = parent
	}

	return nil, ErrMaxDepth
}

// Remove deletes a Wave from the thread index.
func (idx *Index) Remove(waveID []byte) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	id := string(waveID)

	// Remove from parent's replies list.
	if parent, ok := idx.parents[id]; ok {
		replies := idx.replies[parent]
		for i, r := range replies {
			if r == id {
				idx.replies[parent] = append(replies[:i], replies[i+1:]...)
				break
			}
		}
		delete(idx.parents, id)
	}

	// Orphan any replies (they keep their parent_id but we lose the index).
	delete(idx.replies, id)

	return nil
}

// ReplyCount returns the number of direct replies to a Wave.
func (idx *Index) ReplyCount(waveID []byte) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.replies[string(waveID)])
}

// TotalReplies returns the total number of replies in a thread tree.
func (idx *Index) TotalReplies(waveID []byte) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.countRepliesRecursive(string(waveID), make(map[string]bool))
}

// countRepliesRecursive counts all replies in a subtree.
func (idx *Index) countRepliesRecursive(waveID string, seen map[string]bool) int {
	if seen[waveID] {
		return 0
	}
	seen[waveID] = true

	count := 0
	for _, reply := range idx.replies[waveID] {
		count++
		count += idx.countRepliesRecursive(reply, seen)
	}
	return count
}

// Thread represents a complete thread structure for serialization.
type Thread struct {
	Root    *pb.Wave
	Replies []*ThreadNode
}

// ThreadNode is a node in a thread tree.
type ThreadNode struct {
	Wave    *pb.Wave
	Replies []*ThreadNode
}

// LoadThread loads a complete thread tree from storage.
// Requires a loader function to fetch Waves by ID.
func (idx *Index) LoadThread(
	rootID []byte,
	loader func([]byte) (*pb.Wave, error),
) (*Thread, error) {
	root, err := loader(rootID)
	if err != nil {
		return nil, err
	}

	thread := &Thread{
		Root: root,
	}

	// Load replies recursively.
	thread.Replies, err = idx.loadReplies(rootID, loader, make(map[string]bool), 0)
	if err != nil {
		return nil, err
	}

	return thread, nil
}

// loadReplies recursively loads reply nodes.
func (idx *Index) loadReplies(
	parentID []byte,
	loader func([]byte) (*pb.Wave, error),
	seen map[string]bool,
	depth int,
) ([]*ThreadNode, error) {
	if depth >= MaxThreadDepth {
		return nil, ErrMaxDepth
	}

	idx.mu.RLock()
	replyIDs := idx.replies[string(parentID)]
	idx.mu.RUnlock()

	if len(replyIDs) == 0 {
		return nil, nil
	}

	nodes := make([]*ThreadNode, 0, len(replyIDs))
	for _, replyID := range replyIDs {
		if seen[replyID] {
			continue
		}
		seen[replyID] = true

		wave, err := loader([]byte(replyID))
		if err != nil {
			continue
		}

		node := &ThreadNode{Wave: wave}
		node.Replies, _ = idx.loadReplies([]byte(replyID), loader, seen, depth+1)
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// Serialize serializes a Thread to protobuf bytes.
func (t *Thread) Serialize() ([]byte, error) {
	// Create a simple serialization format.
	// For now, just serialize the root Wave.
	// Full thread serialization would require a Thread protobuf message.
	return proto.Marshal(t.Root)
}
