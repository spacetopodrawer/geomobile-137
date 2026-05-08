package neogeo

import (
	"encoding/binary"
	"fmt"
)

// NeoRageX5Protocol handles NEO-GEO network protocol encoding/decoding
type NeoRageX5Protocol struct {
	version uint16
}

// NewNeoRageX5Protocol creates a new protocol handler
func NewNeoRageX5Protocol() *NeoRageX5Protocol {
	return &NeoRageX5Protocol{
		version: 0x0500, // Version 5.0
	}
}

// EncodeFrame encodes a game frame to protocol format
func (p *NeoRageX5Protocol) EncodeFrame(frameID uint64, playerX, playerY float32, score uint32) []byte {
	buf := make([]byte, 32)

	// Frame header
	binary.BigEndian.PutUint16(buf[0:], 0xF00D) // Magic
	binary.BigEndian.PutUint16(buf[2:], p.version)

	// Frame data
	binary.BigEndian.PutUint64(buf[4:], frameID)
	binary.BigEndian.PutUint32(buf[12:], uint32(playerX))
	binary.BigEndian.PutUint32(buf[16:], uint32(playerY))
	binary.BigEndian.PutUint32(buf[20:], score)

	// Checksum
	checksum := uint16(0)
	for i := 0; i < 30; i += 2 {
		checksum ^= binary.BigEndian.Uint16(buf[i : i+2])
	}
	binary.BigEndian.PutUint16(buf[30:], checksum)

	return buf
}

// DecodeFrame decodes a protocol frame
func (p *NeoRageX5Protocol) DecodeFrame(data []byte) error {
	if len(data) < 32 {
		return fmt.Errorf("frame too short: %d bytes", len(data))
	}

	magic := binary.BigEndian.Uint16(data[0:])
	if magic != 0xF00D {
		return fmt.Errorf("invalid magic: 0x%04X", magic)
	}

	version := binary.BigEndian.Uint16(data[2:])
	if version != p.version {
		return fmt.Errorf("version mismatch: got 0x%04X, expected 0x%04X", version, p.version)
	}

	// Verify checksum
	checksum := uint16(0)
	for i := 0; i < 30; i += 2 {
		checksum ^= binary.BigEndian.Uint16(data[i : i+2])
	}

	storedChecksum := binary.BigEndian.Uint16(data[30:])
	if checksum != storedChecksum {
		return fmt.Errorf("checksum mismatch: got 0x%04X, expected 0x%04X", checksum, storedChecksum)
	}

	return nil
}

// EncodeInput encodes controller input to protocol format
func (p *NeoRageX5Protocol) EncodeInput(direction uint8, buttons uint8) []byte {
	return []byte{direction, buttons, 0, 0}
}

// DecodeInput decodes protocol input
func (p *NeoRageX5Protocol) DecodeInput(data []byte) (direction uint8, buttons uint8, err error) {
	if len(data) < 2 {
		return 0, 0, fmt.Errorf("input too short: %d bytes", len(data))
	}

	return data[0], data[1], nil
}
