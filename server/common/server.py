import socket
import logging
import signal
import threading
from typing import Callable

from common import protocol
from common.lottery import LotteryManager
from common.protocol import Protocol, ProtocolError


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._shutdown_event = threading.Event()

        # Register the signal handler for graceful shutdown
        signal.signal(signal.SIGTERM, self.__handle_sigterm)

        # Initialize the LotteryManager
        self.lottery_manager = LotteryManager()

        # Create handlers dictionary
        self.message_handlers = {
            protocol.consts.MSG_BET: self.handle_bet,
            protocol.consts.MSG_BATCH: self.handle_batch,
            protocol.consts.MSG_NOTIFY: self.handle_notify,
            protocol.consts.MSG_WINNERS_QUERY: self.handle_winners_query
        }

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
        client_threads = []
        while not self._shutdown_event.is_set():
            try:
                client_sock = self.__accept_new_connection()
                if client_sock:
                    thread = threading.Thread(target=self.__handle_client_connection, args=(client_sock,))
                    client_threads.append(thread)
                    thread.start()

            except OSError:
                break  # server socket being closed
        logging.info('action: server_shutdown | result: success')
        self.__shutdown_workers(client_threads)

    def __handle_client_connection(self, client_sock: socket.socket):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        proto = Protocol(client_sock)
        try:
            self.handle_client(proto)
        except (ConnectionError, OSError) as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            if client_sock:
                client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
        try:
            client_sock, addr = self._server_socket.accept()
            logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
            return client_sock
        except BlockingIOError:
            return None

    @staticmethod
    def __shutdown_workers(client_threads):
        for thread in client_threads:
            thread.join()
        logging.info("All client threads have finished.")

    def handle_client(self, proto: Protocol) -> None:
        """Handles client communication."""
        while True:
            try:
                message_data: bytes = proto.receive_message()
                message_type = int(message_data[0])  # Ensure message_type is an integer

                # Look for the handler using an integer key
                handler: Callable[[Protocol, bytes], None] = self.message_handlers.get(message_type)

                if handler:
                    handler(proto, message_data[1:])
                else:
                    raise ProtocolError(f"Unknown message type: {message_type}", proto.consts.ERROR_INVALID_MESSAGE)

            except ProtocolError as e:
                logging.error(f'Protocol error: {e}')
                proto.send_response(proto.consts.MSG_ERROR, e.code)
            except ConnectionError:
                logging.info('Client disconnected, closing connection')
                break
            except OSError as e:
                logging.error(f'OSError during client handling: {e}')
                break
            except Exception as e:
                logging.error(f'Unexpected error: {e}')
                proto.send_response(proto.consts.MSG_ERROR, proto.consts.ERROR_INVALID_MESSAGE)
                break

    def handle_bet(self, proto: Protocol, message_data: bytes):
        logging.info('Received MSG_BET')
        bet = proto.deserialize_bet(message_data)
        self.lottery_manager.register_bet(bet)
        proto.send_response(proto.consts.MSG_SUCCESS, proto.consts.SUCCESS_BET_PROCESSED)

    def handle_batch(self, proto: Protocol, message_data: bytes):
        logging.info('Received MSG_BATCH')
        bets = proto.deserialize_batch(message_data)
        self.lottery_manager.register_batch(bets)
        logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
        proto.send_response(proto.consts.MSG_SUCCESS, proto.consts.SUCCESS_BATCH_PROCESSED)

    def handle_notify(self, proto: Protocol, message_data: bytes):
        logging.info('Received MSG_NOTIFY')
        agency_id = proto.deserialize_agency_id(message_data)
        self.lottery_manager.notify_agency(agency_id)

    def handle_winners_query(self, proto: Protocol, message_data: bytes):
        logging.info('Received MSG_WINNERS_QUERY')
        agency_id = proto.deserialize_agency_id(message_data)
        winners = self.lottery_manager.query_winners(agency_id)

        if winners is not None:
            proto.send_winners_list(winners)
            logging.info('action: winners_query | result: success')
        else:
            proto.send_response(proto.consts.MSG_ERROR, proto.consts.ERROR_LOTTERY_NOT_DONE)
            logging.info('action: winners_query | result: fail | reason: lottery not done')
