package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

// MsgType is an enumeration of the different message types and protocol errors
type MsgType int

const (
	MSG_SUCCESS      MsgType = 0x00 // 0x00, Success message
	MSG_BET          MsgType = 0x01 // 0x01, Bet message
	MSG_ECHO         MsgType = 0x02 // 0x02, Echo message
	MSG_ERROR        MsgType = 0x03 // 0x03, Error message
	REJECT_MALFORMED MsgType = 0x04 // 0x04, Malformed message rejection
	REJECT_INVALID   MsgType = 0x05 // 0x05, Invalid message rejection
)

func (m MsgType) String() string {
	switch m {
	case REJECT_MALFORMED:
		return "REJECT_MALFORMED"
	case REJECT_INVALID:
		return "REJECT_INVALID"
	case MSG_SUCCESS:
		return "MSG_SUCCESS"
	case MSG_BET:
		return "MSG_BET"
	case MSG_ECHO:
		return "MSG_ECHO"
	case MSG_ERROR:
		return "MSG_ERROR"
	default:
		return "UNKNOWN"
	}
}

// Protocol defines the behavior of our protocol
type Protocol struct {
	conn net.Conn
}

// NewProtocol creates a new instance of the protocol
func NewProtocol(conn net.Conn) *Protocol {
	return &Protocol{conn: conn}
}

// SerializeBet serializes a Bet object into binary format according to the protocol.
func (p *Protocol) SerializeBet(bet Bet) ([]byte, error) {
	var buffer bytes.Buffer

	// Serialize Agency (4 bytes, uint32)
	if err := binary.Write(&buffer, binary.BigEndian, uint32(bet.Agency)); err != nil {
		return nil, err
	}

	// Serialize FirstName (4 bytes length prefix + string)
	if err := p.serializeString(&buffer, bet.FirstName); err != nil {
		return nil, err
	}

	// Serialize LastName (4 bytes length prefix + string)
	if err := p.serializeString(&buffer, bet.LastName); err != nil {
		return nil, err
	}

	// Serialize Document (4 bytes length prefix + string)
	if err := p.serializeString(&buffer, bet.Document); err != nil {
		return nil, err
	}

	// Serialize BirthDate (10 bytes, string "YYYY-MM-DD")
	birthDateStr := bet.BirthDate.Format("2006-01-02")
	if len(birthDateStr) != 10 {
		return nil, fmt.Errorf("invalid birth date format")
	}
	if _, err := buffer.WriteString(birthDateStr); err != nil {
		return nil, err
	}

	// Serialize Number (4 bytes, uint32)
	if err := binary.Write(&buffer, binary.BigEndian, uint32(bet.Number)); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// serializeString serializes a string with a 4-byte length prefix.
func (p *Protocol) serializeString(buffer *bytes.Buffer, str string) error {
	strLength := uint32(len(str))
	if err := binary.Write(buffer, binary.BigEndian, strLength); err != nil {
		return err
	}
	if _, err := buffer.Write([]byte(str)); err != nil {
		return err
	}
	return nil
}

// SendAll ensures that all data is sent over the socket
func (p *Protocol) SendAll(data []byte) error {
	totalSent := 0
	for totalSent < len(data) {
		sent, err := p.conn.Write(data[totalSent:])
		if err != nil {
			return err
		}
		if sent == 0 {
			return fmt.Errorf("socket connection broken")
		}
		totalSent += sent
	}
	return nil
}

// ReadExactly ensures that exactly n bytes are read from the socket
func (p *Protocol) ReadExactly(n int) ([]byte, error) {
	data := make([]byte, n)
	totalRead := 0
	for totalRead < n {
		read, err := p.conn.Read(data[totalRead:])
		if err != nil {
			return nil, err
		}
		if read == 0 {
			return nil, fmt.Errorf("socket connection broken")
		}
		totalRead += read
	}
	return data, nil
}

// SendBet sends a serialized Bet object to the server.
func (p *Protocol) SendBet(bet Bet) error {
	betData, err := p.SerializeBet(bet)
	if err != nil {
		log.Errorf("Failed to serialize bet: %v", err)
		return err
	}

	messageLength := uint32(len(betData)) + 1 // +1 for the message type
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, messageLength)

	// Send the message length, type, and serialized data
	if err := p.SendAll(header); err != nil {
		log.Errorf("Failed to send message length: %v", err)
		return err
	}

	messageType := byte(MSG_BET) // Use constant for MSG_BET
	if err := p.SendAll([]byte{messageType}); err != nil {
		log.Errorf("Failed to send message type: %v", err)
		return err
	}

	if err := p.SendAll(betData); err != nil {
		log.Errorf("Failed to send bet data: %v", err)
		return err
	}

	return nil
}

// ReceiveResponse receives and parses the server's response
func (p *Protocol) ReceiveResponse() (int, string, error) {
	header, err := p.ReadExactly(4)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read response header: %v", err)
	}

	messageLength := binary.BigEndian.Uint32(header)
	statusCode, err := p.ReadExactly(1)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read status code: %v", err)
	}

	messageBody, err := p.ReadExactly(int(messageLength) - 5)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read message body: %v", err)
	}

	return int(statusCode[0]), string(messageBody), nil
}
