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
