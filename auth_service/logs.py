import functools
import logging
import os
from logging.handlers import RotatingFileHandler

import grpc
import pymysql

logger = logging.getLogger("auth_service")


def setup_logging():
    log_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "logs")
    os.makedirs(log_dir, exist_ok=True)
    handler = RotatingFileHandler(
        filename=os.path.join(log_dir, "auth_service.log"),
        maxBytes=5_000_000, backupCount=3, encoding="utf-8"
    )

    handler.setFormatter(logging.Formatter("%(asctime)s %(levelname)s %(name)s: %(message)s"))
    logging.basicConfig(level=logging.INFO, handlers=[handler])


def handle_errors(method):
    @functools.wraps(method)
    def wrapper(self, request, context):
        try:
            return method(self, request, context)
        except pymysql.err.OperationalError:
            logger.exception("Baza de date nu raspunde in %s", method.__name__)
            context.abort(grpc.StatusCode.UNAVAILABLE, "Serviciul nu este disponibil momentan.")
        except Exception:
            # context.abort() arunca si el o exceptie, dar seteaza inainte codul de eroare.
            if context.code() is not None:
                # INFO, nu WARNING: o cerere respinsa e un raspuns normal, nu o problema a serverului.
                logger.info("%s respins pentru '%s': %s - %s", method.__name__,
                            getattr(request, "username", "-"), context.code().name,
                            (context.details() or b"").decode())
                raise
            logger.exception("Eroare neasteptata in %s", method.__name__)
            context.abort(grpc.StatusCode.INTERNAL, "Eroare interna a serverului.")
    return wrapper
