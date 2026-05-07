//go:build !js

// Package store provides Bbolt-based persistent storage for MURMUR.
// This file contains protobuf marshal/unmarshal helpers integrated with the store.
package store

import (
	"fmt"

	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// MarshalPut marshals a protobuf message and stores it in the specified bucket.
// This is a generic helper that accepts any proto.Message implementation.
func (db *DB) MarshalPut(bucket, key []byte, msg proto.Message) error {
	if msg == nil {
		return fmt.Errorf("nil message")
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling protobuf: %w", err)
	}
	return db.Put(bucket, key, data)
}

// UnmarshalGet retrieves and unmarshals a protobuf message from the specified bucket.
// The msg parameter must be a pointer to the target protobuf message type.
// Returns nil error and leaves msg unchanged if key doesn't exist.
func (db *DB) UnmarshalGet(bucket, key []byte, msg proto.Message) error {
	data, err := db.Get(bucket, key)
	if err != nil {
		return err
	}
	if data == nil {
		return nil // Key not found - not an error
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("unmarshaling protobuf: %w", err)
	}
	return nil
}

// MarshalPutBatch marshals and stores multiple key-message pairs in a single transaction.
// This is more efficient than calling MarshalPut repeatedly for bulk inserts.
func (db *DB) MarshalPutBatch(bucket []byte, items map[string]proto.Message) error {
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		for key, msg := range items {
			if msg == nil {
				return fmt.Errorf("nil message for key %s", key)
			}
			data, err := proto.Marshal(msg)
			if err != nil {
				return fmt.Errorf("marshaling protobuf for key %s: %w", key, err)
			}
			if err := b.Put([]byte(key), data); err != nil {
				return fmt.Errorf("storing key %s: %w", key, err)
			}
		}
		return nil
	})
}

// Clone performs a deep copy of a protobuf message by marshaling and unmarshaling.
// This is useful when you need to modify a message without affecting the original.
func Clone[T proto.Message](msg T) (T, error) {
	var zero T
	if any(msg) == nil {
		return zero, fmt.Errorf("nil message")
	}
	return proto.Clone(msg).(T), nil
}

// Size returns the serialized size of a protobuf message in bytes.
func Size(msg proto.Message) int {
	if msg == nil {
		return 0
	}
	return proto.Size(msg)
}

// Equal compares two protobuf messages for equality.
func Equal(a, b proto.Message) bool {
	return proto.Equal(a, b)
}
