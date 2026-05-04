// Package ignition implements Proximity Ignition — in-person connection mechanics.
// This file provides NFC-specific compact encoding for tap exchange.
//
// Per RESONANCE_SYSTEM.md §Proximity Ignition:
// "When two MURMUR users are physically co-located, they can establish a connection
// by exchanging connection data over a local channel (QR code scan, NFC tap, or
// mDNS local discovery with mutual confirmation)."
//
// NFC has limited payload capacity:
//   - NTAG213: 137 bytes usable
//   - NTAG215: 504 bytes usable
//   - NTAG216: 872 bytes usable
//
// This file provides a compact binary format optimized for NFC tap exchange,
// targeting <137 bytes to support the widest range of NFC tags.
package ignition

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

// NFC format constants.
const (
	// NFCVersion is the NFC compact format version.
	NFCVersion uint8 = 1

	// NFCMinPayload is the minimum NFC payload size (version + pubkey + token + timestamp + sig).
	NFCMinPayload = 1 + 32 + TokenSize + 4 + ed25519.SignatureSize // 117 bytes

	// NFCMaxPayload is the maximum NFC payload for NTAG213 compatibility.
	NFCMaxPayload = 137

	// NFCMimeType is the NDEF MIME type for MURMUR ignition data.
	NFCMimeType = "application/vnd.murmur.ignition"
)

// Errors for NFC operations.
var (
	ErrNFCPayloadTooLarge = errors.New("NFC payload exceeds maximum size")
	ErrNFCInvalidPayload  = errors.New("invalid NFC payload format")
	ErrNFCNoAddresses     = errors.New("at least one address is required")
)

// NFCIgnitionData is a compact representation of ignition data for NFC exchange.
// Unlike the full IgnitionData, this uses a binary format optimized for size.
//
// Binary format (117-137 bytes):
//
//	[0]: version (1 byte)
//	[1-32]: public key (32 bytes)
//	[33-48]: token (16 bytes)
//	[49-52]: timestamp as seconds since epoch truncated to uint32 (4 bytes)
//	[53]: address type + count (1 byte: high nibble = type, low nibble = count)
//	[54-N]: address data (variable, compact encoding)
//	[N-end]: signature (64 bytes)
//
// Address types:
//
//	0x1n: IPv4 addresses (6 bytes each: 4 IP + 2 port)
//	0x2n: IPv6 addresses (18 bytes each: 16 IP + 2 port)
//	0x3n: Peer ID hash only (8 bytes, for relay-only connections)
type NFCIgnitionData struct {
	Version   uint8
	PublicKey ed25519.PublicKey
	Token     [TokenSize]byte
	Timestamp uint32 // Unix timestamp truncated to uint32 (valid until 2106)
	Addresses []NFCAddress
	Signature []byte
}

// NFCAddressType indicates the type of address encoding.
type NFCAddressType uint8

const (
	// NFCAddressTypeIPv4 is an IPv4 address with port (6 bytes).
	NFCAddressTypeIPv4 NFCAddressType = 0x10

	// NFCAddressTypeIPv6 is an IPv6 address with port (18 bytes).
	NFCAddressTypeIPv6 NFCAddressType = 0x20

	// NFCAddressTypePeerID is a truncated peer ID hash (8 bytes).
	NFCAddressTypePeerID NFCAddressType = 0x30
)

// NFCAddress represents a compact network address for NFC exchange.
type NFCAddress struct {
	Type   NFCAddressType
	IPv4   [4]byte  // Only for Type=IPv4
	IPv6   [16]byte // Only for Type=IPv6
	Port   uint16   // For IPv4 and IPv6
	PeerID [8]byte  // Only for Type=PeerID (truncated hash)
}

// GenerateNFCIgnitionData creates compact ignition data for NFC exchange.
// The addresses parameter should be simple IP:port strings or peer IDs.
// Only the first address is included to minimize payload size.
func GenerateNFCIgnitionData(privateKey ed25519.PrivateKey, addrs []string, token [TokenSize]byte) (*NFCIgnitionData, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPublicKey
	}
	if len(addrs) == 0 {
		return nil, ErrNFCNoAddresses
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Parse addresses into compact format (limit to first 3).
	nfcAddrs := make([]NFCAddress, 0, 3)
	for i, addr := range addrs {
		if i >= 3 {
			break // Max 3 addresses for NFC
		}
		nfcAddr, err := parseAddressCompact(addr)
		if err != nil {
			continue // Skip invalid addresses
		}
		nfcAddrs = append(nfcAddrs, nfcAddr)
	}

	if len(nfcAddrs) == 0 {
		return nil, ErrNFCNoAddresses
	}

	data := &NFCIgnitionData{
		Version:   NFCVersion,
		PublicKey: publicKey,
		Token:     token,
		Timestamp: uint32(currentTimestamp()),
		Addresses: nfcAddrs,
	}

	// Sign the data.
	sigInput := data.signatureInput()
	data.Signature = ed25519.Sign(privateKey, sigInput)

	return data, nil
}

// signatureInput returns the data to be signed (excludes signature itself).
func (d *NFCIgnitionData) signatureInput() []byte {
	var buf bytes.Buffer

	buf.WriteByte(d.Version)
	buf.Write(d.PublicKey)
	buf.Write(d.Token[:])
	_ = binary.Write(&buf, binary.BigEndian, d.Timestamp)

	// Address type+count byte.
	if len(d.Addresses) > 0 {
		addrByte := byte(d.Addresses[0].Type) | byte(len(d.Addresses))
		buf.WriteByte(addrByte)

		// Address data.
		for _, addr := range d.Addresses {
			writeAddressCompact(&buf, addr)
		}
	} else {
		buf.WriteByte(0)
	}

	return buf.Bytes()
}

// Encode serializes NFCIgnitionData to compact binary format.
func (d *NFCIgnitionData) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Write signature input data.
	buf.Write(d.signatureInput())

	// Append signature.
	buf.Write(d.Signature)

	if buf.Len() > NFCMaxPayload {
		return nil, ErrNFCPayloadTooLarge
	}

	return buf.Bytes(), nil
}

// DecodeNFCIgnitionData parses compact binary NFC data.
func DecodeNFCIgnitionData(data []byte) (*NFCIgnitionData, error) {
	if len(data) < NFCMinPayload {
		return nil, ErrNFCInvalidPayload
	}

	version, idx, err := parseVersionField(data, 0)
	if err != nil {
		return nil, err
	}

	publicKey, idx, err := parsePublicKeyField(data, idx)
	if err != nil {
		return nil, err
	}

	token, idx, err := parseTokenField(data, idx)
	if err != nil {
		return nil, err
	}

	timestamp, idx, err := parseTimestampField(data, idx)
	if err != nil {
		return nil, err
	}

	addresses, idx, err := parseAddressesField(data, idx)
	if err != nil {
		return nil, err
	}

	signature, err := parseSignatureField(data, idx)
	if err != nil {
		return nil, err
	}

	result := &NFCIgnitionData{
		Version:   version,
		PublicKey: publicKey,
		Token:     token,
		Timestamp: timestamp,
		Addresses: addresses,
		Signature: signature,
	}

	if err := verifySignature(result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseVersionField extracts and validates the version byte.
func parseVersionField(data []byte, idx int) (byte, int, error) {
	version := data[idx]
	idx++
	if version != NFCVersion {
		return 0, idx, ErrVersionMismatch
	}
	return version, idx, nil
}

// parsePublicKeyField extracts the Ed25519 public key.
func parsePublicKeyField(data []byte, idx int) (ed25519.PublicKey, int, error) {
	publicKey := make(ed25519.PublicKey, 32)
	copy(publicKey, data[idx:idx+32])
	idx += 32
	return publicKey, idx, nil
}

// parseTokenField extracts the token.
func parseTokenField(data []byte, idx int) ([TokenSize]byte, int, error) {
	var token [TokenSize]byte
	copy(token[:], data[idx:idx+TokenSize])
	idx += TokenSize
	return token, idx, nil
}

// parseTimestampField extracts the timestamp.
func parseTimestampField(data []byte, idx int) (uint32, int, error) {
	timestamp := binary.BigEndian.Uint32(data[idx : idx+4])
	idx += 4
	return timestamp, idx, nil
}

// parseAddressesField extracts the addresses.
func parseAddressesField(data []byte, idx int) ([]NFCAddress, int, error) {
	if idx >= len(data) {
		return nil, idx, ErrNFCInvalidPayload
	}
	addrByte := data[idx]
	idx++
	addrType := NFCAddressType(addrByte & 0xF0)
	addrCount := int(addrByte & 0x0F)

	addresses := make([]NFCAddress, 0, addrCount)
	for i := 0; i < addrCount; i++ {
		addr, n, err := readAddressCompact(data[idx:], addrType)
		if err != nil {
			return nil, idx, err
		}
		addresses = append(addresses, addr)
		idx += n
	}
	return addresses, idx, nil
}

// parseSignatureField extracts the signature.
func parseSignatureField(data []byte, idx int) ([]byte, error) {
	if idx+ed25519.SignatureSize > len(data) {
		return nil, ErrNFCInvalidPayload
	}
	signature := make([]byte, ed25519.SignatureSize)
	copy(signature, data[idx:idx+ed25519.SignatureSize])
	return signature, nil
}

// verifySignature verifies the Ed25519 signature over the ignition data.
func verifySignature(result *NFCIgnitionData) error {
	sigInput := result.signatureInput()
	if !ed25519.Verify(result.PublicKey, sigInput, result.Signature) {
		return ErrInvalidSignature
	}
	return nil
}

// Verify checks that the NFC ignition data has a valid signature.
func (d *NFCIgnitionData) Verify() bool {
	if len(d.PublicKey) != ed25519.PublicKeySize {
		return false
	}
	sigInput := d.signatureInput()
	return ed25519.Verify(d.PublicKey, sigInput, d.Signature)
}

// PayloadSize returns the encoded payload size in bytes.
func (d *NFCIgnitionData) PayloadSize() int {
	size := NFCMinPayload - ed25519.SignatureSize + 1 // Base + addr type byte

	for _, addr := range d.Addresses {
		switch addr.Type {
		case NFCAddressTypeIPv4:
			size += 6
		case NFCAddressTypeIPv6:
			size += 18
		case NFCAddressTypePeerID:
			size += 8
		}
	}

	size += ed25519.SignatureSize
	return size
}

// ToIgnitionData converts NFCIgnitionData to full IgnitionData.
func (d *NFCIgnitionData) ToIgnitionData() *IgnitionData {
	addresses := make([]string, len(d.Addresses))
	for i, addr := range d.Addresses {
		addresses[i] = addr.String()
	}

	return &IgnitionData{
		Version:   d.Version,
		PublicKey: d.PublicKey,
		Token:     d.Token,
		Timestamp: int64(d.Timestamp),
		Addresses: addresses,
		Signature: d.Signature,
	}
}

// String returns the address as a multiaddr-like string.
func (a NFCAddress) String() string {
	switch a.Type {
	case NFCAddressTypeIPv4:
		ip := net.IP(a.IPv4[:])
		return fmt.Sprintf("/ip4/%s/tcp/%d", ip.String(), a.Port)
	case NFCAddressTypeIPv6:
		ip := net.IP(a.IPv6[:])
		return fmt.Sprintf("/ip6/%s/tcp/%d", ip.String(), a.Port)
	case NFCAddressTypePeerID:
		return fmt.Sprintf("/p2p/%x", a.PeerID)
	default:
		return ""
	}
}

// parseAddressCompact parses a string address into compact format.
func parseAddressCompact(addr string) (NFCAddress, error) {
	var result NFCAddress

	// Try to parse as IP:port.
	host, port, err := parseHostPort(addr)
	if err == nil {
		ip := net.ParseIP(host)
		if ip == nil {
			return result, ErrInvalidAddress
		}

		if ip4 := ip.To4(); ip4 != nil {
			result.Type = NFCAddressTypeIPv4
			copy(result.IPv4[:], ip4)
			result.Port = port
			return result, nil
		}

		if ip6 := ip.To16(); ip6 != nil {
			result.Type = NFCAddressTypeIPv6
			copy(result.IPv6[:], ip6)
			result.Port = port
			return result, nil
		}
	}

	// Try to parse as multiaddr-style.
	return parseMultiaddrCompact(addr)
}

// parseHostPort extracts host and port from "host:port" or multiaddr.
func parseHostPort(addr string) (string, uint16, error) {
	// Handle multiaddr format: /ip4/x.x.x.x/tcp/port
	if len(addr) > 0 && addr[0] == '/' {
		return parseMultiaddrHostPort(addr)
	}

	// Handle standard host:port.
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}

	var port uint16
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return "", 0, err
	}

	return host, port, nil
}

// parseMultiaddrHostPort extracts host and port from multiaddr.
func parseMultiaddrHostPort(addr string) (string, uint16, error) {
	// Simple multiaddr parser for /ip4/x.x.x.x/tcp/port or /ip6/...
	parts := splitMultiaddr(addr)

	var host string
	var port uint16

	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "ip4", "ip6":
			host = parts[i+1]
		case "tcp", "udp":
			_, err := fmt.Sscanf(parts[i+1], "%d", &port)
			if err != nil {
				return "", 0, err
			}
		}
	}

	if host == "" || port == 0 {
		return "", 0, ErrInvalidAddress
	}

	return host, port, nil
}

// splitMultiaddr splits a multiaddr string into components.
func splitMultiaddr(addr string) []string {
	parts := make([]string, 0)
	current := ""

	for _, c := range addr {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// parseMultiaddrCompact parses a full multiaddr into compact format.
func parseMultiaddrCompact(addr string) (NFCAddress, error) {
	var result NFCAddress
	parts := splitMultiaddr(addr)

	for i := 0; i < len(parts)-1; i++ {
		if err := parseMultiaddrPart(&result, parts[i], parts[i+1]); err != nil {
			return result, err
		}
	}

	if result.Type == 0 {
		return result, ErrInvalidAddress
	}

	return result, nil
}

// parseMultiaddrPart processes a single multiaddr component.
func parseMultiaddrPart(result *NFCAddress, protocol, value string) error {
	switch protocol {
	case "ip4":
		return parseIPv4(result, value)
	case "ip6":
		return parseIPv6(result, value)
	case "tcp", "udp":
		return parsePort(result, value)
	case "p2p":
		parsePeerID(result, value)
	}
	return nil
}

// parseIPv4 parses and stores an IPv4 address.
func parseIPv4(result *NFCAddress, value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return ErrInvalidAddress
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ErrInvalidAddress
	}
	result.Type = NFCAddressTypeIPv4
	copy(result.IPv4[:], ip4)
	return nil
}

// parseIPv6 parses and stores an IPv6 address.
func parseIPv6(result *NFCAddress, value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return ErrInvalidAddress
	}
	ip6 := ip.To16()
	if ip6 == nil {
		return ErrInvalidAddress
	}
	result.Type = NFCAddressTypeIPv6
	copy(result.IPv6[:], ip6)
	return nil
}

// parsePort parses and stores a port number.
func parsePort(result *NFCAddress, value string) error {
	var port uint16
	_, err := fmt.Sscanf(value, "%d", &port)
	if err != nil {
		return ErrInvalidAddress
	}
	result.Port = port
	return nil
}

// parsePeerID parses and stores a truncated peer ID.
func parsePeerID(result *NFCAddress, value string) {
	result.Type = NFCAddressTypePeerID
	if len(value) >= 16 {
		copy(result.PeerID[:], value[:8])
	}
}

// writeAddressCompact writes a compact address to a buffer.
func writeAddressCompact(buf *bytes.Buffer, addr NFCAddress) {
	switch addr.Type {
	case NFCAddressTypeIPv4:
		buf.Write(addr.IPv4[:])
		_ = binary.Write(buf, binary.BigEndian, addr.Port)

	case NFCAddressTypeIPv6:
		buf.Write(addr.IPv6[:])
		_ = binary.Write(buf, binary.BigEndian, addr.Port)

	case NFCAddressTypePeerID:
		buf.Write(addr.PeerID[:])
	}
}

// readAddressCompact reads a compact address from a buffer.
func readAddressCompact(data []byte, addrType NFCAddressType) (NFCAddress, int, error) {
	var addr NFCAddress
	addr.Type = addrType

	switch addrType {
	case NFCAddressTypeIPv4:
		if len(data) < 6 {
			return addr, 0, ErrNFCInvalidPayload
		}
		copy(addr.IPv4[:], data[:4])
		addr.Port = binary.BigEndian.Uint16(data[4:6])
		return addr, 6, nil

	case NFCAddressTypeIPv6:
		if len(data) < 18 {
			return addr, 0, ErrNFCInvalidPayload
		}
		copy(addr.IPv6[:], data[:16])
		addr.Port = binary.BigEndian.Uint16(data[16:18])
		return addr, 18, nil

	case NFCAddressTypePeerID:
		if len(data) < 8 {
			return addr, 0, ErrNFCInvalidPayload
		}
		copy(addr.PeerID[:], data[:8])
		return addr, 8, nil

	default:
		return addr, 0, ErrNFCInvalidPayload
	}
}

// currentTimestamp returns the current Unix timestamp.
// Extracted for testing purposes.
var currentTimestamp = func() int64 {
	return timeNow().Unix()
}

// timeNow is the time function, replaceable for testing.
var timeNow = time.Now

// NFCNDEFRecord wraps NFC ignition data as an NDEF record.
// NDEF (NFC Data Exchange Format) is the standard format for NFC tag data.
type NFCNDEFRecord struct {
	// TypeNameFormat is the NDEF TNF value (0x02 for MIME media type).
	TypeNameFormat byte

	// Type is the MIME type string.
	Type string

	// Payload is the raw ignition data.
	Payload []byte
}

// ToNDEF wraps the ignition data in an NDEF record.
func (d *NFCIgnitionData) ToNDEF() (*NFCNDEFRecord, error) {
	payload, err := d.Encode()
	if err != nil {
		return nil, err
	}

	return &NFCNDEFRecord{
		TypeNameFormat: 0x02, // MIME media type
		Type:           NFCMimeType,
		Payload:        payload,
	}, nil
}

// DecodeNFCNDEF parses an NDEF record into NFCIgnitionData.
func DecodeNFCNDEF(record *NFCNDEFRecord) (*NFCIgnitionData, error) {
	if record.TypeNameFormat != 0x02 {
		return nil, errors.New("unsupported NDEF type name format")
	}
	if record.Type != NFCMimeType {
		return nil, errors.New("unexpected NDEF type")
	}

	return DecodeNFCIgnitionData(record.Payload)
}
