import socket
import logging
import signal
import threading
from common.protocol import Protocol, ProtocolError
from common.utils import store_bets


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._shutdown_event = threading.Event()

        # Register the signal handler for graceful shutdown
        signal.signal(signal.SIGTERM, self.__handle_sigterm)

    def __handle_sigterm(self, signum, frame):
        """
        Handle SIGTERM signal to initiate a graceful shutdown
        """
        logging.info('action: exit | result: success | message: SIGINT received')
        self._shutdown_event.set()
        self._server_socket.close()

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        while not self._shutdown_event.is_set():
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError:
                break  # server socket being closed

    def __handle_client_connection(self, client_sock: socket.socket):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        protocol = Protocol(client_sock)
        try:
            self.handle_client(protocol)
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]}')
        except (ConnectionError, OSError, ProtocolError) as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def handle_client(self, protocol: Protocol) -> None:
        """Handles client communication."""
        message_data: bytes = b''

        try:
            message_data = protocol.receive_message()
            message_type = message_data[0]
            offset = 1

            if message_type != protocol.consts.MSG_BET:
                raise ProtocolError("Unknown message type", protocol.consts.REJECT_INVALID)
            bet = protocol.deserialize_bet(message_data[offset:])
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')
            store_bets([bet])
            protocol.send_response(protocol.consts.MSG_SUCCESS, "Bet stored successfully")
        except ProtocolError as e:
            if e.code == protocol.consts.MSG_ECHO:
                logging.info("Handling echo message due to malformed protocol message")
                protocol.handle_echo_message(message_data)
            else:
                logging.error(f'Protocol error: {e}')
                protocol.send_response(e.code, "Protocol error occurred")
        except Exception as e:
            logging.error(f'Unexpected error: {e}')
            protocol.send_response(protocol.consts.REJECT_INVALID, "Unexpected error occurred")
