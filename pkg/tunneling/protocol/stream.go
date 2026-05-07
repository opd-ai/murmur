package protocol

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

const (
	// StreamProtocolID is the tunnel stream protocol ID used for tunnel cells.
	StreamProtocolID = "/murmur/tunnel/1"
	// FrameMagic is the first byte of every framed tunnel protocol message.
	FrameMagic = byte(0x7f)

	headerSize      = 6
	maxPayloadBytes = 1 << 20
)

// FrameType identifies a framed tunnel cell message.
type FrameType byte

const (
	FrameTypeRegister FrameType = 1
	FrameTypeData     FrameType = 2
	FrameTypeTeardown FrameType = 3
)

// WriteFrame serializes and writes a framed payload.
func WriteFrame(w io.Writer, frameType FrameType, payload []byte) error {
	if len(payload) > maxPayloadBytes {
		return fmt.Errorf("payload too large: %d", len(payload))
	}

	header := make([]byte, headerSize)
	header[0] = FrameMagic
	header[1] = byte(frameType)
	binary.BigEndian.PutUint32(header[2:], uint32(len(payload)))

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}
	return nil
}

// ReadFrame reads and validates one framed payload from reader.
func ReadFrame(r io.Reader) (FrameType, []byte, error) {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}
	if header[0] != FrameMagic {
		return 0, nil, fmt.Errorf("invalid frame magic: 0x%x", header[0])
	}

	return readFramePayload(r, header)
}

// ReadFrameWithFirstByte reads a frame where the caller already consumed the first header byte.
func ReadFrameWithFirstByte(r io.Reader, firstByte byte) (FrameType, []byte, error) {
	if firstByte != FrameMagic {
		return 0, nil, fmt.Errorf("invalid frame magic: 0x%x", firstByte)
	}

	header := make([]byte, headerSize)
	header[0] = firstByte
	if _, err := io.ReadFull(r, header[1:]); err != nil {
		return 0, nil, err
	}

	return readFramePayload(r, header)
}

func readFramePayload(r io.Reader, header []byte) (FrameType, []byte, error) {
	payloadLen := binary.BigEndian.Uint32(header[2:])
	if payloadLen > maxPayloadBytes {
		return 0, nil, fmt.Errorf("frame payload too large: %d", payloadLen)
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
	}

	return FrameType(header[1]), payload, nil
}

// EncodeRegisterCell marshals a TunnelRegisterCell with signed metadata.
func EncodeRegisterCell(id tunneling.TunnelID, pub ed25519.PublicKey, priv ed25519.PrivateKey, bandwidthLimit uint64) ([]byte, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size")
	}
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}

	timestamp := time.Now().Unix()
	toSign := registerSignBytes([]byte(id), timestamp)
	sig := ed25519.Sign(priv, toSign)

	cell := &pb.TunnelRegisterCell{
		TunnelId:       []byte(id),
		OperatorPubkey: pub,
		Signature:      sig,
		TimestampUnix:  timestamp,
		BandwidthLimit: bandwidthLimit,
	}
	b, err := proto.Marshal(cell)
	if err != nil {
		return nil, fmt.Errorf("marshal register cell: %w", err)
	}
	return b, nil
}

// DecodeAndVerifyRegisterCell unmarshals and verifies a register cell signature and timestamp.
func DecodeAndVerifyRegisterCell(payload []byte, now time.Time) (*pb.TunnelRegisterCell, error) {
	cell := &pb.TunnelRegisterCell{}
	if err := proto.Unmarshal(payload, cell); err != nil {
		return nil, fmt.Errorf("unmarshal register cell: %w", err)
	}
	if len(cell.OperatorPubkey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid operator public key size")
	}
	if len(cell.Signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("invalid register signature size")
	}

	if !withinSkewWindow(cell.TimestampUnix, now.Unix(), 300) {
		return nil, fmt.Errorf("register timestamp outside allowable skew window")
	}

	if !ed25519.Verify(ed25519.PublicKey(cell.OperatorPubkey), registerSignBytes(cell.TunnelId, cell.TimestampUnix), cell.Signature) {
		return nil, fmt.Errorf("invalid register signature")
	}

	return cell, nil
}

func registerSignBytes(id []byte, ts int64) []byte {
	buf := make([]byte, len(id)+8)
	copy(buf, id)
	binary.BigEndian.PutUint64(buf[len(id):], uint64(ts))
	return buf
}

func withinSkewWindow(ts, now, maxSkewSeconds int64) bool {
	delta := now - ts
	if delta < 0 {
		delta = -delta
	}
	return delta <= maxSkewSeconds
}
