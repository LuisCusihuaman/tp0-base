package common

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

// CSVReader structure that encapsulates reading CSV files from a ZIP.
type CSVReader struct {
	ZipPath  string
	AgencyID string
}

// NewCSVReader constructor for CSVReader.
func NewCSVReader(zipPath string, agencyID string) *CSVReader {
	return &CSVReader{
		ZipPath:  zipPath,
		AgencyID: agencyID,
	}
}

// ReadBets reads the bets from the CSV file inside the ZIP and sends them to the bets channel.
func (r *CSVReader) ReadBets(betChan chan<- Bet) error {
	zipReader, err := zip.OpenReader(r.ZipPath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	expectedFileName := fmt.Sprintf("agency-%s.csv", r.AgencyID)
	var csvFile *zip.File
	for _, file := range zipReader.File {
		if file.Name == expectedFileName {
			csvFile = file
			break
		}
	}

	if csvFile == nil {
		return fmt.Errorf("CSV file for agency ID %s not found in the archive", r.AgencyID)
	}

	rc, err := csvFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Parse the record into a bet
		bet, err := r.parseRecordToBet(record)
		if err != nil {
			return err
		}

		betChan <- bet
	}

	return nil
}

// parseRecordToBet converts a CSV record into a Bet structure.
func (r *CSVReader) parseRecordToBet(record []string) (Bet, error) {
	if len(record) < 5 {
		return Bet{}, fmt.Errorf("invalid record format")
	}

	// Parse the Document field from string to uint32
	document, err := strconv.ParseUint(record[2], 10, 32)
	if err != nil {
		return Bet{}, fmt.Errorf("invalid document: %v", err)
	}

	// Parse the Number field from string to int
	number, err := strconv.Atoi(record[4])
	if err != nil {
		return Bet{}, fmt.Errorf("invalid number: %v", err)
	}

	// Parse the BirthDate field
	birthDate, err := time.Parse("2006-01-02", record[3])
	if err != nil {
		return Bet{}, fmt.Errorf("invalid birth date: %v", err)
	}

	// Parse the AgencyID from the CSVReader's AgencyID field
	agencyID, _ := strconv.Atoi(r.AgencyID)

	// Return the constructed Bet object
	return Bet{
		Agency:    agencyID,
		FirstName: record[0],
		LastName:  record[1],
		Document:  uint32(document),
		BirthDate: birthDate,
		Number:    number,
	}, nil
}
