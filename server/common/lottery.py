import threading
import logging
from common.utils import store_bets, load_bets, has_won, Bet
from typing import List, Union

# Constante que define cuántas agencias se van a esperar antes de realizar el sorteo
AGENCY_COUNT: int = 5


class LotteryManager:
    def __init__(self):
        # Diccionario para rastrear notificaciones de agencias
        self.agency_notifications: dict[int, bool] = {i: False for i in range(1, AGENCY_COUNT + 1)}
        self.lock: threading.Lock = threading.Lock()
        self.lottery_done: bool = False
        self.winners: List[Bet] = []

    def register_bet(self, bet: Bet) -> None:
        """
        Registra una apuesta y la persiste en el almacenamiento.
        """
        logging.info(f'Registering bet: {bet}')
        with self.lock:
            store_bets([bet])
        logging.info('Bet registered successfully.')

    def register_batch(self, bets: List[Bet]) -> None:
        """
        Registra un lote de apuestas y las persiste en el almacenamiento.
        """
        logging.info(f'Registering batch of {len(bets)} bets.')
        with self.lock:
            store_bets(bets)
        logging.info(f'Batch of {len(bets)} bets registered successfully.')

    def notify_agency(self, agency_id: int) -> None:
        """
        Marca a una agencia como notificada. Si ya ha sido notificada previamente, no se vuelve a procesar.
        Cuando todas las agencias hayan notificado, realiza el sorteo y procesa las apuestas para determinar los ganadores.
        """
        with self.lock:
            if agency_id not in self.agency_notifications:
                logging.error(f"Unknown agency ID: {agency_id}")
                return

            if self.agency_notifications[agency_id]:
                logging.warning(f'Agency {agency_id} has already notified. Skipping.')
                return

            self.agency_notifications[agency_id] = True
            logging.info(f'Agency {agency_id} has notified.')

    def all_agencies_notified(self) -> bool:
        """
        Verifica si todas las agencias han notificado.
        """
        return all(self.agency_notifications.values())

    def perform_lottery(self) -> None:
        """
        Realiza el sorteo, carga las apuestas y determina los ganadores.
        """
        with self.lock:
            if self.lottery_done:
                logging.info("Lottery already done. Skipping re-execution.")
                return
            self.winners = [bet for bet in load_bets() if has_won(bet)]
            logging.info(f'action: sorteo | result: success | cant_ganadores: {len(self.winners)}')
            self.lottery_done = True

    def query_winners(self, agency_id: int) -> Union[List[int], None]:
        """
        Retorna la lista de documentos de los ganadores de una agencia específica después de que se haya realizado el sorteo.
        Retorna None si la lotería no ha sido realizada.
        """
        with self.lock:
            if not self.lottery_done:
                logging.error('Lottery not done yet. Cannot query winners.')
                return None  # Lotería no realizada

            logging.info(f'Agency {agency_id} querying winners.')
            agency_winners = [int(winner.document) for winner in self.winners if winner.agency == agency_id]

            logging.info(f'action: consulta_ganadores | result: success | cant_ganadores: {len(agency_winners)}')
            return agency_winners  # Podría ser una lista vacía si no hay ganadores
