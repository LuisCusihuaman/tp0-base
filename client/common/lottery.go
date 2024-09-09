package common

import (
	"encoding/binary"
	"strconv"
)

// NotifyMessage represents the MSG_NOTIFY message
type NotifyMessage struct {
	AgencyID uint32
}

func (m *NotifyMessage) Serialize() ([]byte, error) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, m.AgencyID)
	return data, nil
}

func (m *NotifyMessage) MessageType() MsgType {
	return MSG_NOTIFY
}

// WinnersQueryMessage represents the MSG_WINNERS_QUERY message
type WinnersQueryMessage struct {
	AgencyID uint32
}

func (m *WinnersQueryMessage) Serialize() ([]byte, error) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, m.AgencyID)
	return data, nil
}

func (m *WinnersQueryMessage) MessageType() MsgType {
	return MSG_WINNERS_QUERY
}

// WinnersListMessage represents the MSG_WINNERS_LIST message
type WinnersListMessage struct {
	Winners []uint32
}

func (m *WinnersListMessage) Serialize() ([]byte, error) {
	winnerCount := len(m.Winners)
	data := make([]byte, 4*winnerCount)
	for i, winner := range m.Winners {
		binary.BigEndian.PutUint32(data[4*i:], winner)
	}
	return data, nil
}

func (m *WinnersListMessage) MessageType() MsgType {
	return MSG_WINNERS_LIST
}

func NewNotifyMessage(agencyID string) *NotifyMessage {
	id, _ := strconv.ParseUint(agencyID, 10, 32)
	return &NotifyMessage{AgencyID: uint32(id)}
}

func NewWinnersQueryMessage(agencyID string) *WinnersQueryMessage {
	id, _ := strconv.ParseUint(agencyID, 10, 32)
	return &WinnersQueryMessage{AgencyID: uint32(id)}
}
