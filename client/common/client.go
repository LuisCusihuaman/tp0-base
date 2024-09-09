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
	retryChan := make(chan WinnersResponse)

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
			c.startSender(batchProcessor, retryChan)
		}()
	}

	wg.Wait()
	close(retryChan)
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
func (c *Client) startSender(batchProcessor *BatchProcessor, retryChan chan WinnersResponse) {
	// Create a connection and protocol instance
	if err := c.createClientSocket(); err != nil {
		log.Criticalf("Failed to create client socket: %v", err)
		return
	}
	defer c.Close()

	protocol := NewProtocol(c.conn)

	// Handle server responses in parallel
	go c.readServerResponses(protocol, retryChan)

	// Send batches and notify when each is done
	if err := batchProcessor.SendBatches(protocol); err != nil {
		log.Errorf("Failed to send batch: %v", err)
		return
	}

	// Notify the server that all bets have been sent
	notifyMsg := NewNotifyMessage(c.config.ID)
	log.Infof("Sending MSG_NOTIFY for agency: %v", c.config.ID)
	if err := protocol.SendMessage(notifyMsg); err != nil {
		log.Errorf("Failed to send notification: %v", err)
		return
	}

	// Query the list of winners
	log.Infof("Sending initial MSG_WINNERS_QUERY for agency: %v", c.config.ID)
	queryMsg := NewWinnersQueryMessage(c.config.ID)
	if err := protocol.SendMessage(queryMsg); err != nil {
		log.Errorf("Failed to send initial query for winners: %v", err)
		return
	}

	// Start retry mechanism to query the list of winners based on signals from retryChan
	c.queryWinnersWithRetry(protocol, retryChan)
}

// readServerResponses continuously listens to server responses and sends a retry signal if needed.
func (c *Client) readServerResponses(protocol *Protocol, retryChan chan WinnersResponse) {
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
			if response.Type == MSG_WINNERS_LIST {
				winners := response.Body.([]uint32)
				retryChan <- WinnersResponse{Winners: winners, Err: nil}
				return
			} else if response.Message == "ERROR_LOTTERY_NOT_DONE" {
				retryChan <- WinnersResponse{Winners: nil, Err: errors.New("lottery not done")}
			} else {
				log.Infof("Received response: Type=%s | Message=%s", response.Type, response.Message)
			}
		}
	}
}

// queryWinnersWithRetry handles querying winners and retrying based on signals from retryChan.
func (c *Client) queryWinnersWithRetry(protocol *Protocol, retryChan chan WinnersResponse) {
	maxRetries := 5
	retryInterval := 40 * time.Second
	retries := 1 // Start from 1 because the first query has already been sent

	for retries <= maxRetries {
		select {
		case response := <-retryChan:
			if response.Err == nil {
				winnerCount := len(response.Winners)
				log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", winnerCount)
				return
			}
			log.Errorf("action: consulta_ganadores | result: fail | reason: %v", response.Err)
			retries++
			log.Infof("Retrying MSG_WINNERS_QUERY for agency: %v (attempt %d)", c.config.ID, retries)

		case <-time.After(retryInterval):
			log.Infof("No response received, retrying MSG_WINNERS_QUERY for agency: %v (attempt %d)", c.config.ID, retries)
			queryMsg := NewWinnersQueryMessage(c.config.ID)
			if err := protocol.SendMessage(queryMsg); err != nil {
				log.Errorf("Failed to send winners query: %v", err)
				return
			}
		}

		if retries >= maxRetries {
			log.Errorf("Max retries reached. Unable to get winners list.")
			break // Exit loop if max retries reached
		}

		time.Sleep(retryInterval)
	}
}
