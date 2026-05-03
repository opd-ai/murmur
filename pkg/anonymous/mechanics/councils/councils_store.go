// Package mechanics implements anonymous layer game mechanics for MURMUR.
// This file contains CouncilStore for managing Phantom Council instances.
package councils

import (
	"encoding/hex"
	"sort"
	"sync"
)

// CouncilStore manages Phantom Council instances.
type CouncilStore struct {
	mu       sync.RWMutex
	councils map[string]*PhantomCouncil
}

// NewCouncilStore creates a new CouncilStore.
func NewCouncilStore() *CouncilStore {
	return &CouncilStore{
		councils: make(map[string]*PhantomCouncil),
	}
}

// AddCouncil adds a council to the store.
func (s *CouncilStore) AddCouncil(council *PhantomCouncil) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.councils[hex.EncodeToString(council.ID[:])] = council
}

// GetCouncil retrieves a council by ID.
func (s *CouncilStore) GetCouncil(id [32]byte) *PhantomCouncil {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.councils[hex.EncodeToString(id[:])]
}

// RemoveCouncil removes a council from the store.
func (s *CouncilStore) RemoveCouncil(id [32]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.councils, hex.EncodeToString(id[:]))
}

// GetActiveCouncils returns all active councils.
func (s *CouncilStore) GetActiveCouncils() []*PhantomCouncil {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*PhantomCouncil
	for _, c := range s.councils {
		if c.IsActive() {
			active = append(active, c)
		}
	}

	// Sort by creation time.
	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt.Before(active[j].CreatedAt)
	})

	return active
}

// GetCouncilsForMember returns councils where Specter is a member.
func (s *CouncilStore) GetCouncilsForMember(specter [32]byte) []*PhantomCouncil {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var memberCouncils []*PhantomCouncil
	for _, c := range s.councils {
		if c.IsMember(specter) {
			memberCouncils = append(memberCouncils, c)
		}
	}
	return memberCouncils
}

// Count returns the number of councils in the store.
func (s *CouncilStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.councils)
}

// PruneDisbanded removes disbanded councils.
func (s *CouncilStore) PruneDisbanded() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	pruned := 0
	for id, c := range s.councils {
		if c.State == CouncilDisbanded {
			delete(s.councils, id)
			pruned++
		}
	}
	return pruned
}
