package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// Bet represents a bet entity
type Bet struct {
	Agency    int
	FirstName string
	LastName  string
	Document  uint32
	BirthDate time.Time
	Number    int
}

// Serialize serializes a Bet object into binary format according to the protocol.
func (b *Bet) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	// Serialize Agency (uint32)
	if err := binary.Write(&buffer, binary.BigEndian, uint32(b.Agency)); err != nil {
		return nil, err
	}

	// Serialize FirstName (variable length string)
	if err := serializeString(&buffer, b.FirstName); err != nil {
		return nil, err
	}

	// Serialize LastName (variable length string)
	if err := serializeString(&buffer, b.LastName); err != nil {
		return nil, err
	}

	// Serialize Document (uint32)
	if err := binary.Write(&buffer, binary.BigEndian, b.Document); err != nil {
		return nil, err
	}

	// Serialize BirthDate (fixed 10-byte string)
	birthDateStr := b.BirthDate.Format("2006-01-02")
	if len(birthDateStr) != 10 {
		return nil, fmt.Errorf("invalid birth date format")
	}
	if _, err := buffer.WriteString(birthDateStr); err != nil {
		return nil, err
	}

	// Serialize Number (uint32)
	if err := binary.Write(&buffer, binary.BigEndian, uint32(b.Number)); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// MessageType returns the message type for a Bet entity
func (b *Bet) MessageType() MsgType {
	return MSG_BET
}

// serializeString is a helper function to serialize a string with a 4-byte length prefix.
func serializeString(buffer *bytes.Buffer, str string) error {
	strLength := uint32(len(str))
	if err := binary.Write(buffer, binary.BigEndian, strLength); err != nil {
		return err
	}
	if _, err := buffer.Write([]byte(str)); err != nil {
		return err
	}
	return nil
}
