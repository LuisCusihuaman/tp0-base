import threading
import logging
from common.utils import store_bets, Bet
from typing import List


class LotteryManager:
    def __init__(self):
        # Diccionario para rastrear notificaciones de agencias
        self.lock: threading.Lock = threading.Lock()

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
