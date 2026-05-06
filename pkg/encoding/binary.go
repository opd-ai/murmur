// Package encoding provides binary encoding utilities for MURMUR.
package encoding

import "encoding/binary"

// AppendUint64BE appends a uint64 as big-endian bytes to a buffer.
// This consolidates the duplicate pattern:
//
// buf := make([]byte, 8)
// binary.BigEndian.PutUint64(buf, value)
// dest = append(dest, buf...)
func AppendUint64BE(dest []byte, value uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, value)
	return append(dest, buf...)
}

// AppendInt64BE appends an int64 as big-endian bytes to a buffer.
func AppendInt64BE(dest []byte, value int64) []byte {
	return AppendUint64BE(dest, uint64(value))
}

// AppendUint32BE appends a uint32 as big-endian bytes to a buffer.
func AppendUint32BE(dest []byte, value uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, value)
	return append(dest, buf...)
}

// AppendInt32BE appends an int32 as big-endian bytes to a buffer.
func AppendInt32BE(dest []byte, value int32) []byte {
	return AppendUint32BE(dest, uint32(value))
}

// Int64ToBytes converts an int64 to an 8-byte big-endian array.
// This consolidates duplicate implementations in waves/types.go and mechanics/common.go.
func Int64ToBytes(v int64) [8]byte {
	var b [8]byte
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}
