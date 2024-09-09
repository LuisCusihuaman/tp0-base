import socket
import struct
from types import SimpleNamespace
from typing import Optional, Tuple
from common.utils import Bet

# Constants for message types and protocol errors
consts = SimpleNamespace(
    MSG_SUCCESS=0x00,  # Success message
    MSG_BET=0x01,  # Bet message
    MSG_ECHO=0x02,  # Echo message
    MSG_ERROR=0x03,  # Error message
    REJECT_MALFORMED=0x04,  # Malformed message rejection
    REJECT_INVALID=0x05,  # Invalid message rejection
    MAX_MSG_LENGTH=1024,  # Maximum message length
    MIN_PROTOCOL_MESSAGE_LENGTH=6  # Minimum protocol message length
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

    def read_with_timeout(self, n: int, timeout: float = 0.1) -> bytes:
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

    def receive_message(self) -> bytes:
        """Reads the length header and the entire message."""
        header: bytes = self.conn.recv(4, socket.MSG_PEEK)  # Peek to check if it's a valid header

        if all(32 <= byte <= 126 for byte in header):
            raise ProtocolError("Error unpacking header", self.consts.MSG_ECHO)

        # Otherwise, it's likely a structured protocol message
        header = self.read_exactly(4)
        message_length: int = struct.unpack('>I', header)[0]

        if message_length < self.consts.MIN_PROTOCOL_MESSAGE_LENGTH or message_length > self.consts.MAX_MSG_LENGTH:
            raise ProtocolError("Invalid message length", self.consts.REJECT_MALFORMED)

        message_data: bytes = self.read_exactly(message_length)
        return message_data

    def send_response(self, code: int, message: str) -> None:
        """Sends a response to the client with a status code and message."""
        message_length: int = 4 + len(message) + 1  # HEADER_LENGTH + message + 1 byte for code
        header: bytes = struct.pack('>I', message_length)
        response: bytes = header + bytes([code]) + message.encode('utf-8')
        self.send_all(response)

    def handle_echo_message(self, initial_data: bytes) -> None:
        """Handles echo messages by reading the rest of the message and sending it back."""
        remaining_length = self.consts.MAX_MSG_LENGTH - len(initial_data)
        remaining_data = self.read_with_timeout(remaining_length)
        self.send_all(initial_data + remaining_data)

    def deserialize_bet(self, data: bytes) -> Bet:
        """Deserializes a Bet object from binary data according to the protocol."""
        offset: int = 0

        # Deserialize Agency (4 bytes, uint32)
        agency: int = struct.unpack('>I', data[offset:offset + 4])[0]
        offset += 4

        # Deserialize First Name (variable length string)
        first_name, offset = self.deserialize_string(data, offset)

        # Deserialize Last Name (variable length string)
        last_name, offset = self.deserialize_string(data, offset)

        # Deserialize Document (variable length string)
        document, offset = self.deserialize_string(data, offset)

        # Deserialize Birth Date (10 bytes string)
        birthdate: str = data[offset:offset + 10].decode('utf-8')
        offset += 10

        # Deserialize Number (4 bytes, uint32)
        number: int = struct.unpack('>I', data[offset:offset + 4])[0]

        return Bet(str(agency), first_name, last_name, document, birthdate, str(number))

    def deserialize_string(self, data: bytes, offset: int) -> Tuple[str, int]:
        """Deserializes a string from binary data, reading the length prefix first."""
        str_length: int = struct.unpack('>I', data[offset:offset + 4])[0]
        offset += 4
        string: str = data[offset:offset + str_length].decode('utf-8')
        offset += str_length
        return string, offset
