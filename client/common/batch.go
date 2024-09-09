package common

import (
	"bytes"
	"encoding/binary"
)

// BatchMessage represents a batch of Bet messages
type BatchMessage struct {
	Bets []Bet
}

// Serialize serializes a BatchMessage object into binary format.
func (b *BatchMessage) Serialize() ([]byte, error) {
	var buffer bytes.Buffer

	// Serialize the count of bets in the batch
	if err := binary.Write(&buffer, binary.BigEndian, uint32(len(b.Bets))); err != nil {
		return nil, err
	}

	// Serialize each Bet in the batch
	for _, bet := range b.Bets {
		betData, err := bet.Serialize()
		if err != nil {
			return nil, err
		}

		// Serialize the length of the bet data
		betLength := uint32(len(betData))
		if err := binary.Write(&buffer, binary.BigEndian, betLength); err != nil {
			return nil, err
		}

		// Serialize the bet data itself
		if _, err := buffer.Write(betData); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}

// MessageType returns the message type for a BatchMessage
func (b *BatchMessage) MessageType() MsgType {
	return MSG_BATCH
}

// BatchProcessor is responsible for batching bets and sending them.
type BatchProcessor struct {
	MaxBatchSize int // in bytes
	BatchChan    chan BatchMessage
}

func NewBatchProcessor(maxBatchSizeKB int) *BatchProcessor {
	return &BatchProcessor{
		MaxBatchSize: maxBatchSizeKB * 1024, // Convert KB to bytes
		BatchChan:    make(chan BatchMessage),
	}
}

// StartBatching reads bets from the bet channel and groups them into batches.
func (bp *BatchProcessor) StartBatching(betChan <-chan Bet) {
	var batch []Bet
	currentBatchSize := HEADER_LENGTH + 1 + 4 // Header + message type + count

	for bet := range betChan {
		betData, err := bet.Serialize()
		if err != nil {
			log.Errorf("Failed to serialize bet: %v", err)
			continue
		}

		// Calculate the estimated size for the next bet
		estimatedSize := currentBatchSize + 4 + len(betData) // 4 bytes for the length prefix

		if estimatedSize > bp.MaxBatchSize {
			// If adding this bet would exceed the max batch size, send the current batch
			bp.BatchChan <- BatchMessage{Bets: batch}
			batch = nil                              // Reset batch
			currentBatchSize = HEADER_LENGTH + 1 + 4 // Reset size (header + type + count)
		}

		// Add the bet to the batch
		batch = append(batch, bet)
		currentBatchSize += 4 + len(betData) // 4 bytes for the length prefix
	}

	// Send the final batch if there are remaining bets
	if len(batch) > 0 {
		bp.BatchChan <- BatchMessage{Bets: batch}
	}
	close(bp.BatchChan)
}

// SendBatches sends the batches to the server using the provided protocol.
func (bp *BatchProcessor) SendBatches(protocol *Protocol) error {
	for batchMsg := range bp.BatchChan {
		log.Infof("Sending Batch of %d bets", len(batchMsg.Bets))
		if err := protocol.SendMessage(&batchMsg); err != nil {
			log.Errorf("Failed to send batch: %v", err)
			return err // Return the error if it occurs
		}
	}
	return nil // Return nil if all batches are sent successfully
}
