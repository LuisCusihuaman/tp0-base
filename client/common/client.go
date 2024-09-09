package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type Bet struct {
	Agency    int
	FirstName string
	LastName  string
	Document  string
	BirthDate time.Time
	Number    int
}

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.conn = conn
	return nil
}

// SerializeBet Serializes a Bet object into binary format according to the protocol.
func SerializeBet(bet Bet) ([]byte, error) {
	var buffer bytes.Buffer

	// Serialize Agency (4 bytes, uint32)
	if err := binary.Write(&buffer, binary.BigEndian, uint32(bet.Agency)); err != nil {
		return nil, err
	}

	// Serialize FirstName (4 bytes length prefix + string)
	if err := serializeString(&buffer, bet.FirstName); err != nil {
		return nil, err
	}

	// Serialize LastName (4 bytes length prefix + string)
	if err := serializeString(&buffer, bet.LastName); err != nil {
		return nil, err
	}

	// Serialize Document (4 bytes length prefix + string)
	if err := serializeString(&buffer, bet.Document); err != nil {
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
	fmt.Printf("Serialized Bet data: %x\n", buffer.Bytes())
	return buffer.Bytes(), nil
}

// serializeString Serializes a string with a 4-byte length prefix.
func serializeString(buffer *bytes.Buffer, str string) error {
	// Write the length of the string as a 4-byte uint32
	strLength := uint32(len(str))
	if err := binary.Write(buffer, binary.BigEndian, strLength); err != nil {
		return err
	}

	// Write the string itself
	if _, err := buffer.Write([]byte(str)); err != nil { // Ensure correct conversion to bytes
		return err
	}
	return nil
}

// SendBet Sends a serialized Bet object to the server.
func (c *Client) SendBet(bet Bet) error {
	// Serialize the Bet object
	betData, err := SerializeBet(bet)
	if err != nil {
		log.Errorf("Failed to serialize bet: %v", err)
		return err
	}

	// Calculate the length of the message
	// Length includes the size of the message type (1 byte) and the serialized bet data
	betLength := uint32(len(betData))
	messageLength := betLength + 1 // +1 for the message type, total length of the message to be sent

	// Send the length of the message (4 bytes)
	if err := binary.Write(c.conn, binary.BigEndian, messageLength); err != nil {
		log.Errorf("Failed to send message length: %v", err)
		return err
	}

	// Send the message type (1 byte)
	messageType := byte(MSG_BET) // Use the MSG_BET constant
	if _, err := c.conn.Write([]byte{messageType}); err != nil {
		log.Errorf("Failed to send message type: %v", err)
		return err
	}

	// Send the serialized Bet data
	if _, err := c.conn.Write(betData); err != nil {
		log.Errorf("Failed to send bet data: %v", err)
		return err
	}

	return nil
}

// ReceiveResponse Receives and parses the server's response
func (c *Client) ReceiveResponse() (int, string, error) {
	// Read the header (4 bytes)
	header := make([]byte, 4)
	_, err := c.conn.Read(header)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read response header: %v", err)
	}

	// Parse the message length from the header
	messageLength := binary.BigEndian.Uint32(header)

	// Read the status code (1 byte)
	statusCode := make([]byte, 1)
	_, err = c.conn.Read(statusCode)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read status code: %v", err)
	}

	// Read the message body (remaining bytes)
	messageBody := make([]byte, messageLength-5)
	_, err = c.conn.Read(messageBody)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read message body: %v", err)
	}

	return int(statusCode[0]), string(messageBody), nil
}

// StartClientLoop Send bet messages to the server until some time threshold is met
func (c *Client) StartClientLoop(bet Bet) {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration. Send an
		err := c.createClientSocket()
		if err != nil {
			return
		}

		// Send the bet to the server
		protocol := NewProtocol(c.conn)
		err = protocol.SendBet(bet)
		if err != nil {
			log.Errorf("Failed to send bet: %v", err)
			return
		}

		// Receive the response from the server
		statusCode, response, err := protocol.ReceiveResponse()
		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		// Determine the result of the operation
		if statusCode == int(MSG_SUCCESS) {
			log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %d", bet.Document, bet.Number)
		} else {
			log.Infof("action: apuesta_enviada | result: fail | dni: %s | numero: %d | response: %s", bet.Document, bet.Number, response)
		}

		// Close the connection
		c.conn.Close()

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) StopClientLoop() {
	log.Infof("action: exit | result: success | message: SIGINT received")
	_ = c.conn.Close()
}
