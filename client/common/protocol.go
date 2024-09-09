package common

import (
	"encoding/binary"
	"fmt"
	"net"
)

const HEADER_LENGTH = 4

// Protocol defines the behavior of our protocol
type Protocol struct {
	conn net.Conn
}

// NewProtocol creates a new instance of the protocol
func NewProtocol(conn net.Conn) *Protocol {
	return &Protocol{conn: conn}
}

// SendMessage sends a serialized message to the server.
func (p *Protocol) SendMessage(msg Message) error {
	// Serialize the message
	body, err := msg.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %v", err)
	}

	// Calculate the total length of the message (header + type + data)
	header := make([]byte, HEADER_LENGTH)
	bodyLength := 1 + uint32(len(body)) // 4 bytes header + 1 byte type + data length
	binary.BigEndian.PutUint32(header, bodyLength)

	// Send the header, message type, and the serialized data
	if err := p.SendAll(header); err != nil {
		return fmt.Errorf("failed to send message length: %v", err)
	}

	messageType := byte(msg.MessageType())
	if err := p.SendAll([]byte{messageType}); err != nil {
		return fmt.Errorf("failed to send message type: %v", err)
	}

	if err := p.SendAll(body); err != nil {
		return fmt.Errorf("failed to send message data: %v", err)
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

type Response struct {
	Type    MsgType
	Message string
	Body    interface{}
}

// ReceiveResponse reads and processes the server's response
func (p *Protocol) ReceiveResponse() ([]Response, error) {
	// Read all available data from the socket
	buffer := make([]byte, 4096)
	n, err := p.conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var responses []Response
	offset := 0

	for offset < n {
		// Read the body length to determine the size of the incoming message
		bodyLength := binary.BigEndian.Uint32(buffer[offset : offset+HEADER_LENGTH])
		offset += HEADER_LENGTH

		// Extract the message type
		msgType := MsgType(buffer[offset])
		offset += 1 // Move the pointer after the message type

		// Process the message based on its type
		switch msgType {
		case MSG_SUCCESS:
			response := p.handleSuccessMessage(buffer, offset)
			if response != nil {
				responses = append(responses, *response)
			}
			offset += int(bodyLength) - 1
		case MSG_ERROR:
			response := p.handleErrorMessage(buffer, offset)
			responses = append(responses, *response)
			offset += int(bodyLength) - 1
		case MSG_WINNERS_LIST:
			response := p.handleWinnersList(buffer, offset, int(bodyLength))
			responses = append(responses, *response)
			offset += int(bodyLength) - 1
		default:
			response := Response{
				Type:    msgType,
				Message: "UNKNOWN_MESSAGE_TYPE",
				Body:    nil,
			}
			offset += int(bodyLength) - 1
			responses = append(responses, response)
		}
	}

	return responses, nil
}

// Function to handle winners list messages (MSG_WINNERS_LIST)
func (p *Protocol) handleWinnersList(buffer []byte, offset, bodyLength int) *Response {
	winnerCount := binary.BigEndian.Uint32(buffer[offset : offset+4])
	offset += 4

	var winners []uint32
	for i := 0; i < int(winnerCount); i++ {
		if offset+4 > bodyLength {
			break
		}
		winner := binary.BigEndian.Uint32(buffer[offset : offset+4])
		winners = append(winners, winner)
		offset += 4
	}

	return &Response{
		Type:    MSG_WINNERS_LIST,
		Message: "Winners list received",
		Body:    winners,
	}
}

// Function to handle success messages (MSG_SUCCESS)
func (p *Protocol) handleSuccessMessage(buffer []byte, offset int) *Response {
	code := buffer[offset]
	if code == 0 {
		return nil
	}
	return &Response{
		Type:    MSG_SUCCESS,
		Message: SuccessCode(code).String(),
		Body:    nil,
	}
}

// Function to handle error messages (MSG_ERROR)
func (p *Protocol) handleErrorMessage(buffer []byte, offset int) *Response {
	code := buffer[offset]
	return &Response{
		Type:    MSG_ERROR,
		Message: ErrorCode(code).String(),
		Body:    nil,
	}
}
