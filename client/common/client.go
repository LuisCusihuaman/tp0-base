package common

import (
	"errors"
	"github.com/op/go-logging"
	"net"
	"strings"
	"sync"
	"time"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	MaxBatchSize  int
	NumSenders    int
	ZipPath       string
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

// Close closes the connection
func (c *Client) Close() {
	// Wait for a specified time before closing the connection
	time.Sleep(500 * time.Millisecond)
	err := c.conn.Close()
	if err != nil {
		log.Fatalf("Failed to close connection: %v", err)
	}
	log.Infof("Connection closed successfully for client_id: %v", c.config.ID)
}

// StartClientLoop starts the process of reading bets, batching them, sending the batches,
// notifying the server, and querying for winners.
func (c *Client) StartClientLoop() {
	var wg sync.WaitGroup
	batchProcessor := NewBatchProcessor(c.config.MaxBatchSize)
	betChan := make(chan Bet, c.config.MaxBatchSize)

	// Create a new CSVReader to read bets
	csvReader := NewCSVReader(c.config.ZipPath, c.config.ID)

	// Start reading bets in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.readBets(csvReader, betChan)
	}()

	// Start batching in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		batchProcessor.StartBatching(betChan)
	}()

	// Start sender goroutines
	for i := 0; i < c.config.NumSenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.startSender(batchProcessor)
		}()
	}

	wg.Wait()
}

// readBets handles the reading of bets from a CSV file.
func (c *Client) readBets(csvReader *CSVReader, betChan chan Bet) {
	err := csvReader.ReadBets(betChan)
	if err != nil {
		log.Errorf("Failed to read bets: %v", err)
		close(betChan) // Ensure channel closure on error
		return
	}
	close(betChan) // Close the channel after reading all bets
}

// startSender handles the socket creation, sending batches, notifying the server, and querying winners.
func (c *Client) startSender(batchProcessor *BatchProcessor) {
	// Create a connection and protocol instance
	if err := c.createClientSocket(); err != nil {
		log.Criticalf("Failed to create client socket: %v", err)
		return
	}
	defer c.Close()

	protocol := NewProtocol(c.conn)

	// Handle server responses in parallel
	go c.readServerResponses(protocol)

	// Send batches and notify when each is done
	if err := batchProcessor.SendBatches(protocol); err != nil {
		log.Errorf("Failed to send batch: %v", err)
		return
	}
}

// readServerResponses continuously listens to server responses and logs them.
func (c *Client) readServerResponses(protocol *Protocol) {
	for {
		responses, err := protocol.ReceiveResponse()
		if err != nil {
			// Check if the error is related to a closed connection
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				log.Infof("Connection closed: no more responses expected.")
				return
			}
			log.Errorf("Failed to receive server response: %v", err)
			return
		}
		for _, response := range responses {
			log.Infof("Received response: Type=%s | Message=%s", response.Type, response.Message)
		}
	}
}
