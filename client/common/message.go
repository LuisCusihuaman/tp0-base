package common

// MsgType is an enumeration of the different message types and protocol errors
type MsgType int

const (
	MSG_SUCCESS MsgType = 0x00 // 0x00, Success message
	MSG_ERROR   MsgType = 0x01 // 0x01, Error message
	MSG_BET     MsgType = 0x10 // 0x10, Bet message
	MSG_BATCH   MsgType = 0x11 // 0x11, Batch message
	MSG_ECHO    MsgType = 0x12 // 0x12, Echo message
)

func (m MsgType) String() string {
	switch m {
	case MSG_SUCCESS:
		return "MSG_SUCCESS"
	case MSG_ERROR:
		return "MSG_ERROR"
	case MSG_BET:
		return "MSG_BET"
	case MSG_BATCH:
		return "MSG_BATCH"
	case MSG_ECHO:
		return "MSG_ECHO"
	default:
		return "UNKNOWN"
	}
}

// SuccessCode defines specific success codes for the MSG_SUCCESS message type
type SuccessCode int

const (
	SUCCESS_BATCH_PROCESSED SuccessCode = 0x01 // 0x01, Batch processed successfully
	SUCCESS_BET_PROCESSED   SuccessCode = 0x02 // 0x02, Bet processed successfully
)

func (sc SuccessCode) String() string {
	switch sc {
	case SUCCESS_BATCH_PROCESSED:
		return "SUCCESS_BATCH_PROCESSED"
	case SUCCESS_BET_PROCESSED:
		return "SUCCESS_BET_PROCESSED"
	default:
		return "UNKNOWN_SUCCESS_CODE"
	}
}

// ErrorCode defines specific error codes for the MSG_ERROR message type
type ErrorCode int

const (
	ERROR_BATCH_FAILED      ErrorCode = 0x01 // 0x01, Failed to process batch
	ERROR_BET_FAILED        ErrorCode = 0x02 // 0x02, Failed to process bet
	ERROR_MALFORMED_MESSAGE ErrorCode = 0x03 // 0x03, Message was malformed
	ERROR_INVALID_MESSAGE   ErrorCode = 0x04 // 0x04, Message was invalid
)

func (ec ErrorCode) String() string {
	switch ec {
	case ERROR_BATCH_FAILED:
		return "ERROR_BATCH_FAILED"
	case ERROR_BET_FAILED:
		return "ERROR_BET_FAILED"
	case ERROR_MALFORMED_MESSAGE:
		return "ERROR_MALFORMED_MESSAGE"
	case ERROR_INVALID_MESSAGE:
		return "ERROR_INVALID_MESSAGE"
	default:
		return "UNKNOWN_ERROR_CODE"
	}
}

// Message interface to be implemented by entities that need to be serialized and sent
type Message interface {
	Serialize() ([]byte, error)
	MessageType() MsgType
}
