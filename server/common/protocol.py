import logging
import socket
import struct
from types import SimpleNamespace
from typing import Optional, Tuple, List
from common.utils import Bet

# Constants for message types and protocol errors
consts = SimpleNamespace(
    MSG_SUCCESS=0x00,  # Success message
    MSG_ERROR=0x01,  # Error message
    MSG_BET=0x10,  # Bet message
    MSG_BATCH=0x11,  # Batch message
    MSG_ECHO=0x12,  # Echo message
    SUCCESS_BATCH_PROCESSED=0x01,  # Success code for successful batch processing
    SUCCESS_BET_PROCESSED=0x02,  # Success code for successful bet processing
    ERROR_BATCH_FAILED=0x01,  # Error code for batch processing failure
    ERROR_BET_FAILED=0x02,  # Error code for bet processing failure
    ERROR_MALFORMED_MESSAGE=0x03,  # Error code for malformed message
    ERROR_INVALID_MESSAGE=0x04,  # Error code for invalid message
    MAX_ECHO_MSG_LENGTH=1024,  # Maximum echo message length
    MAX_BATCH_MSG_LENGTH=8192,  # Maximum batch message length
    MIN_PROTOCOL_MESSAGE_LENGTH=5  # Minimum protocol message length
)


class ProtocolError(Exception):
    def __init__(self, message: str, code: int) -> None:
        super().__init__(message)
        self.code: int = code


class Protocol:
    def __init__(self, conn: socket.socket):
        self.conn: socket.socket = conn
        self.consts: SimpleNamespace = consts  # Include the consts inside the class

    def read_exactly(self, n: int) -> bytes:
        """Ensures that exactly n bytes are read from the socket."""
        data: bytearray = bytearray()
        while len(data) < n:
            packet: Optional[bytes] = self.conn.recv(n - len(data))
            if not packet:
                raise ConnectionError("Short read")
            data.extend(packet)
        return bytes(data)

    def read_with_timeout(self, n: int, timeout: float = 0.5) -> bytes:
        """Reads up to n bytes with a timeout to avoid blocking indefinitely."""
        data: bytearray = bytearray()
        self.conn.settimeout(timeout)
        try:
            while len(data) < n:
                packet: Optional[bytes] = self.conn.recv(n - len(data))
                if not packet:
                    break
                data.extend(packet)
        except socket.timeout:
            pass  # Stop reading if timeout occurs
        finally:
            self.conn.settimeout(None)  # Reset the timeout to default (blocking mode)
        return bytes(data)

    def send_all(self, data: bytes) -> None:
        """Ensures that all data is sent over the socket."""
        total_sent: int = 0
        while total_sent < len(data):
            sent: int = self.conn.send(data[total_sent:])
            if sent == 0:
                raise ConnectionError("Socket connection broken")
            total_sent += sent

    def receive_message(self, timeout: float = 0.5) -> bytes:
        """Reads the length header and the entire message with a timeout."""
        header: bytes = self.conn.recv(4, socket.MSG_PEEK)  # Peek to check if it's a valid header
        if header == b'':
            raise ConnectionError("Connection closed by client")
        if all(32 <= byte <= 126 for byte in header):
            raise ProtocolError("Error unpacking header", self.consts.MSG_ECHO)

        # Otherwise, it's likely a structured protocol message
        header = self.read_with_timeout(4, timeout)

        # If header is valid, unpack the message length
        body_length: int = struct.unpack('>I', header)[0]

        if not (self.consts.MIN_PROTOCOL_MESSAGE_LENGTH <= body_length < self.consts.MAX_BATCH_MSG_LENGTH):
            raise ProtocolError(f"Invalid message length {body_length}",
                                self.consts.ERROR_MALFORMED_MESSAGE)

        message_data: bytes = self.read_with_timeout(body_length, timeout)

        return message_data

    def send_response(self, msg_type: int, code: int) -> None:
        """Sends a fixed-length response to the client with a message type and code."""
        body_length = 1 + 1  # 1 byte for message type + 1 byte for code
        header = struct.pack('>I', body_length)
        response = header + bytes([msg_type, code])
        self.send_all(response)

    def handle_echo_message(self, initial_data: bytes) -> None:
        """Handles echo messages by reading the rest of the message and sending it back."""
        remaining_length = self.consts.MAX_ECHO_MSG_LENGTH - len(initial_data)
        remaining_data = self.read_with_timeout(remaining_length)
        self.send_all(initial_data + remaining_data)

    def deserialize_bet(self, data: bytes) -> Bet:
        """Deserializes a Bet object from binary data according to the protocol."""
        try:
            offset: int = 0

            # Deserialize Agency (4 bytes, uint32)
            agency: int = struct.unpack('>I', data[offset:offset + 4])[0]
            offset += 4

            # Deserialize First Name (variable length string)
            first_name, offset = self.deserialize_string(data, offset)

            # Deserialize Last Name (variable length string)
            last_name, offset = self.deserialize_string(data, offset)

            # Deserialize Document (4 bytes, uint32)
            document: int = struct.unpack('>I', data[offset:offset + 4])[0]
            offset += 4

            # Deserialize Birth Date (10 bytes string)
            birthdate: str = data[offset:offset + 10].decode('utf-8')
            offset += 10

            # Deserialize Number (4 bytes, uint32)
            number: int = struct.unpack('>I', data[offset:offset + 4])[0]

            # Convert the document from int to string as expected by the Bet class
            return Bet(str(agency), first_name, last_name, str(document), birthdate, str(number))

        except Exception as e:
            logging.error(f"Unexpected error during Bet deserialization: {str(e)}")
            raise ProtocolError(f"Unexpected error during Bet deserialization: {str(e)}", self.consts.ERROR_BET_FAILED)

    def deserialize_batch(self, data: bytes) -> List[Bet]:
        """Deserializes a batch of Bet objects from binary data according to the protocol."""
        offset = 0

        # Deserialize the count of bets in the batch
        bet_count = struct.unpack('>I', data[offset:offset + 4])[0]
        offset += 4
        bets = []
        try:
            for _ in range(bet_count):
                # Deserialize the length of the bet data
                bet_length = struct.unpack('>I', data[offset:offset + 4])[0]
                offset += 4

                # Deserialize the bet data using the length
                bet_data = data[offset:offset + bet_length]
                offset += bet_length

                # Deserialize the bet object
                bet = self.deserialize_bet(bet_data)
                bets.append(bet)
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | cantidad: {len(bets)}")
            raise ProtocolError(f"Failed to deserialize batch: {str(e)}", self.consts.ERROR_BATCH_FAILED)
        return bets

    def deserialize_string(self, data: bytes, offset: int) -> Tuple[str, int]:
        """Deserializes a string from binary data, reading the length prefix first."""
        str_length: int = struct.unpack('>I', data[offset:offset + 4])[0]
        offset += 4
        string: str = data[offset:offset + str_length].decode('utf-8')
        offset += str_length
        return string, offset
