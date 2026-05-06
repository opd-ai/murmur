// Package store provides Bbolt-based persistent storage for MURMUR.
// This file implements Masked Event persistence per ANONYMOUS_GAME_MECHANICS.md.
package store

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

// Masked Event store errors.
var (
	ErrMaskedEventNotFound = errors.New("masked event not found")
	ErrMaskedEventExists   = errors.New("masked event already exists")
)

// StoredMaskedEvent represents a persisted Masked Event.
type StoredMaskedEvent struct {
	// ID is the unique event identifier.
	ID [32]byte

	// Topic describes the event theme.
	Topic string

	// CreatorSpecterKey is the creator's Specter public key.
	CreatorSpecterKey [32]byte

	// StartTime is when the event begins.
	StartTime time.Time

	// EndTime is when the event ends.
	EndTime time.Time

	// Duration is the event length.
	Duration time.Duration

	// MaxParticipants is the cap (0 = unlimited).
	MaxParticipants int

	// State is the lifecycle state (0=pending, 1=active, 2=ended).
	State int

	// CreatedAt is when the event was created.
	CreatedAt time.Time

	// ParticipantCount is the number of joined participants.
	ParticipantCount int

	// TotalWaves is the number of Masked Waves published.
	TotalWaves int

	// TotalAmplifications is the engagement count.
	TotalAmplifications int
}

// StoredMaskedParticipant represents a persisted event participant.
type StoredMaskedParticipant struct {
	// EventID links to the parent event.
	EventID [32]byte

	// MaskedPublicKey is the single-use public key.
	MaskedPublicKey [32]byte

	// Pseudonym is the event-themed identifier.
	Pseudonym string

	// JoinedAt is when the participant joined.
	JoinedAt time.Time

	// WaveCount is Masked Waves published.
	WaveCount int

	// AmplificationsReceived is engagement received.
	AmplificationsReceived int
}

// MaskedEventStore provides CRUD operations for Masked Events.
type MaskedEventStore struct {
	db *DB
}

// NewMaskedEventStore creates a new Masked Event store.
func NewMaskedEventStore(db *DB) *MaskedEventStore {
	return &MaskedEventStore{db: db}
}

// eventKey generates the key for an event.
func (s *MaskedEventStore) eventKey(id [32]byte) []byte {
	return append([]byte("event:"), id[:]...)
}

// participantKey generates the key for a participant.
func (s *MaskedEventStore) participantKey(eventID, maskedKey [32]byte) []byte {
	key := make([]byte, 0, 8+64)
	key = append(key, []byte("part:")...)
	key = append(key, eventID[:]...)
	key = append(key, maskedKey[:]...)
	return key
}

// eventIndexKey generates the key for time-based index.
func (s *MaskedEventStore) eventIndexKey(startTime time.Time, id [32]byte) []byte {
	key := make([]byte, 0, 48)
	key = append(key, []byte("idx:")...)
	// Prefix with Unix timestamp for chronological ordering.
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(startTime.Unix()))
	key = append(key, ts...)
	key = append(key, id[:]...)
	return key
}

// CreateEvent stores a new Masked Event.
func (s *MaskedEventStore) CreateEvent(event *StoredMaskedEvent) error {
	return s.db.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		key := s.eventKey(event.ID)

		// Check if already exists.
		if bucket.Get(key) != nil {
			return ErrMaskedEventExists
		}

		// Serialize event.
		data := s.serializeEvent(event)

		// Store event.
		if err := bucket.Put(key, data); err != nil {
			return err
		}

		// Create time index entry.
		indexKey := s.eventIndexKey(event.StartTime, event.ID)
		if err := bucket.Put(indexKey, event.ID[:]); err != nil {
			return err
		}

		return nil
	})
}

// GetEvent retrieves a Masked Event by ID.
func (s *MaskedEventStore) GetEvent(id [32]byte) (*StoredMaskedEvent, error) {
	var event *StoredMaskedEvent

	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		data := bucket.Get(s.eventKey(id))
		if data == nil {
			return ErrMaskedEventNotFound
		}

		event = s.deserializeEvent(data)
		return nil
	})

	return event, err
}

// UpdateEvent updates an existing Masked Event.
func (s *MaskedEventStore) UpdateEvent(event *StoredMaskedEvent) error {
	return s.db.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		key := s.eventKey(event.ID)

		// Check if exists.
		if bucket.Get(key) == nil {
			return ErrMaskedEventNotFound
		}

		data := s.serializeEvent(event)
		return bucket.Put(key, data)
	})
}

// DeleteEvent removes a Masked Event and its participants.
func (s *MaskedEventStore) DeleteEvent(id [32]byte) error {
	return s.db.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		// Get event for index key.
		data := bucket.Get(s.eventKey(id))
		if data == nil {
			return ErrMaskedEventNotFound
		}
		event := s.deserializeEvent(data)

		// Delete event.
		if err := bucket.Delete(s.eventKey(id)); err != nil {
			return err
		}

		// Delete time index.
		indexKey := s.eventIndexKey(event.StartTime, id)
		if err := bucket.Delete(indexKey); err != nil {
			return err
		}

		// Delete all participants for this event.
		prefix := append([]byte("part:"), id[:]...)
		cursor := bucket.Cursor()
		for k, _ := cursor.Seek(prefix); k != nil && len(k) > len(prefix); k, _ = cursor.Next() {
			if !hasPrefix(k, prefix) {
				break
			}
			if err := bucket.Delete(k); err != nil {
				return err
			}
		}

		return nil
	})
}

// AddParticipant adds a participant to an event.
func (s *MaskedEventStore) AddParticipant(p *StoredMaskedParticipant) error {
	return s.db.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		// Verify event exists.
		if bucket.Get(s.eventKey(p.EventID)) == nil {
			return ErrMaskedEventNotFound
		}

		key := s.participantKey(p.EventID, p.MaskedPublicKey)
		data := s.serializeParticipant(p)
		return bucket.Put(key, data)
	})
}

// GetParticipant retrieves a participant.
func (s *MaskedEventStore) GetParticipant(eventID, maskedKey [32]byte) (*StoredMaskedParticipant, error) {
	var participant *StoredMaskedParticipant

	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		key := s.participantKey(eventID, maskedKey)
		data := bucket.Get(key)
		if data == nil {
			return ErrMaskedEventNotFound
		}

		participant = s.deserializeParticipant(data)
		return nil
	})

	return participant, err
}

// UpdateParticipant updates a participant record.
func (s *MaskedEventStore) UpdateParticipant(p *StoredMaskedParticipant) error {
	return s.db.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		key := s.participantKey(p.EventID, p.MaskedPublicKey)
		if bucket.Get(key) == nil {
			return ErrMaskedEventNotFound
		}

		data := s.serializeParticipant(p)
		return bucket.Put(key, data)
	})
}

// ListParticipants returns all participants for an event.
func (s *MaskedEventStore) ListParticipants(eventID [32]byte) ([]*StoredMaskedParticipant, error) {
	var participants []*StoredMaskedParticipant

	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		prefix := append([]byte("part:"), eventID[:]...)
		cursor := bucket.Cursor()

		for k, v := cursor.Seek(prefix); k != nil; k, v = cursor.Next() {
			if !hasPrefix(k, prefix) {
				break
			}
			participant := s.deserializeParticipant(v)
			participants = append(participants, participant)
		}

		return nil
	})

	return participants, err
}

// ListActiveEvents returns events that are currently active.
func (s *MaskedEventStore) ListActiveEvents() ([]*StoredMaskedEvent, error) {
	var events []*StoredMaskedEvent

	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		prefix := []byte("event:")
		cursor := bucket.Cursor()

		for k, v := cursor.Seek(prefix); k != nil; k, v = cursor.Next() {
			if !hasPrefix(k, prefix) {
				break
			}
			event := s.deserializeEvent(v)
			if event.State == 1 { // Active
				events = append(events, event)
			}
		}

		return nil
	})

	return events, err
}

// ListEventsByTimeRange returns events starting within a time range.
func (s *MaskedEventStore) ListEventsByTimeRange(start, end time.Time) ([]*StoredMaskedEvent, error) {
	var events []*StoredMaskedEvent

	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		// Use time index to iterate by start time.
		idxPrefix := []byte("idx:")

		cursor := bucket.Cursor()
		for k, v := cursor.Seek(idxPrefix); k != nil; k, v = cursor.Next() {
			if !hasPrefix(k, idxPrefix) {
				break
			}
			// Extract timestamp from key.
			if len(k) < 12 {
				continue
			}
			ts := binary.BigEndian.Uint64(k[4:12])
			eventTime := time.Unix(int64(ts), 0)

			if eventTime.Before(start) {
				continue
			}
			if eventTime.After(end) {
				break
			}

			// v contains the event ID.
			var eventID [32]byte
			copy(eventID[:], v)

			eventData := bucket.Get(s.eventKey(eventID))
			if eventData != nil {
				events = append(events, s.deserializeEvent(eventData))
			}
		}

		return nil
	})

	return events, err
}

// CountParticipants returns the participant count for an event.
func (s *MaskedEventStore) CountParticipants(eventID [32]byte) (int, error) {
	count := 0

	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		prefix := append([]byte("part:"), eventID[:]...)
		cursor := bucket.Cursor()

		for k, _ := cursor.Seek(prefix); k != nil; k, _ = cursor.Next() {
			if !hasPrefix(k, prefix) {
				break
			}
			count++
		}

		return nil
	})

	return count, err
}

// CleanupExpiredEvents deletes events that ended before the given time.
func (s *MaskedEventStore) CleanupExpiredEvents(before time.Time) (int, error) {
	var toDelete [][32]byte

	// First pass: collect events to delete.
	err := s.db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketMaskedEvents)

		prefix := []byte("event:")
		cursor := bucket.Cursor()

		for k, v := cursor.Seek(prefix); k != nil; k, v = cursor.Next() {
			if !hasPrefix(k, prefix) {
				break
			}
			event := s.deserializeEvent(v)
			if event.EndTime.Before(before) {
				toDelete = append(toDelete, event.ID)
			}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	// Second pass: delete.
	deleted := 0
	for _, id := range toDelete {
		if err := s.DeleteEvent(id); err == nil {
			deleted++
		}
	}

	return deleted, nil
}

// Serialization helpers.
// Simple binary format for storage efficiency.

// writeFieldsAndString writes a sequence of uint32 fields followed by a length-prefixed string.
func writeFieldsAndString(data []byte, offset int, fields []uint32, str string) int {
	for _, field := range fields {
		binary.BigEndian.PutUint32(data[offset:], field)
		offset += 4
	}
	strBytes := []byte(str)
	binary.BigEndian.PutUint16(data[offset:], uint16(len(strBytes)))
	offset += 2
	copy(data[offset:], strBytes)
	return offset + len(strBytes)
}

func (s *MaskedEventStore) serializeEvent(e *StoredMaskedEvent) []byte {
	// Format: ID(32) + CreatorKey(32) + StartTime(8) + EndTime(8) + Duration(8) +
	//         MaxParticipants(4) + State(4) + CreatedAt(8) + ParticipantCount(4) +
	//         TotalWaves(4) + TotalAmplifications(4) + TopicLen(2) + Topic
	size := 32 + 32 + 8 + 8 + 8 + 4 + 4 + 8 + 4 + 4 + 4 + 2 + len(e.Topic)
	data := make([]byte, size)

	offset := 0
	copy(data[offset:], e.ID[:])
	offset += 32
	copy(data[offset:], e.CreatorSpecterKey[:])
	offset += 32
	binary.BigEndian.PutUint64(data[offset:], uint64(e.StartTime.Unix()))
	offset += 8
	binary.BigEndian.PutUint64(data[offset:], uint64(e.EndTime.Unix()))
	offset += 8
	binary.BigEndian.PutUint64(data[offset:], uint64(e.Duration))
	offset += 8
	binary.BigEndian.PutUint32(data[offset:], uint32(e.MaxParticipants))
	offset += 4
	binary.BigEndian.PutUint32(data[offset:], uint32(e.State))
	offset += 4
	binary.BigEndian.PutUint64(data[offset:], uint64(e.CreatedAt.Unix()))
	offset += 8
	writeFieldsAndString(data, offset, []uint32{
		uint32(e.ParticipantCount),
		uint32(e.TotalWaves),
		uint32(e.TotalAmplifications),
	}, e.Topic)

	return data
}

func (s *MaskedEventStore) deserializeEvent(data []byte) *StoredMaskedEvent {
	if len(data) < 118 {
		return nil
	}

	e := &StoredMaskedEvent{}
	offset := 0

	copy(e.ID[:], data[offset:offset+32])
	offset += 32
	copy(e.CreatorSpecterKey[:], data[offset:offset+32])
	offset += 32
	e.StartTime = time.Unix(int64(binary.BigEndian.Uint64(data[offset:])), 0)
	offset += 8
	e.EndTime = time.Unix(int64(binary.BigEndian.Uint64(data[offset:])), 0)
	offset += 8
	e.Duration = time.Duration(binary.BigEndian.Uint64(data[offset:]))
	offset += 8
	e.MaxParticipants = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	e.State = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	e.CreatedAt = time.Unix(int64(binary.BigEndian.Uint64(data[offset:])), 0)
	offset += 8
	e.ParticipantCount = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	e.TotalWaves = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	e.TotalAmplifications = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	topicLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2
	if offset+topicLen <= len(data) {
		e.Topic = string(data[offset : offset+topicLen])
	}

	return e
}

func (s *MaskedEventStore) serializeParticipant(p *StoredMaskedParticipant) []byte {
	// Format: EventID(32) + MaskedKey(32) + JoinedAt(8) + WaveCount(4) +
	//         AmplificationsReceived(4) + PseudonymLen(2) + Pseudonym
	size := 32 + 32 + 8 + 4 + 4 + 2 + len(p.Pseudonym)
	data := make([]byte, size)

	offset := 0
	copy(data[offset:], p.EventID[:])
	offset += 32
	copy(data[offset:], p.MaskedPublicKey[:])
	offset += 32
	binary.BigEndian.PutUint64(data[offset:], uint64(p.JoinedAt.Unix()))
	offset += 8
	writeFieldsAndString(data, offset, []uint32{
		uint32(p.WaveCount),
		uint32(p.AmplificationsReceived),
	}, p.Pseudonym)

	return data
}

func (s *MaskedEventStore) deserializeParticipant(data []byte) *StoredMaskedParticipant {
	if len(data) < 82 {
		return nil
	}

	p := &StoredMaskedParticipant{}
	offset := 0

	copy(p.EventID[:], data[offset:offset+32])
	offset += 32
	copy(p.MaskedPublicKey[:], data[offset:offset+32])
	offset += 32
	p.JoinedAt = time.Unix(int64(binary.BigEndian.Uint64(data[offset:])), 0)
	offset += 8
	p.WaveCount = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	p.AmplificationsReceived = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	pseudonymLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2
	if offset+pseudonymLen <= len(data) {
		p.Pseudonym = string(data[offset : offset+pseudonymLen])
	}

	return p
}

// EventIDString returns the hex-encoded event ID.
func EventIDString(id [32]byte) string {
	return hex.EncodeToString(id[:])
}

// ParseEventID parses a hex-encoded event ID.
func ParseEventID(s string) ([32]byte, error) {
	var id [32]byte
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return id, err
	}
	if len(bytes) != 32 {
		return id, errors.New("invalid event ID length")
	}
	copy(id[:], bytes)
	return id, nil
}
