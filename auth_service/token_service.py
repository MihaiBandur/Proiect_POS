import jwt
import threading
import datetime
from uuid6 import uuid7
import os
import sys
from dotenv import load_dotenv

blacklist = {}

load_dotenv()
SECRET_KEY = os.getenv("SECRET_KEY")
if SECRET_KEY is None:
    sys.exit("Cheie pentru semnarea token JWT nu exista.")
algorithm = "HS256"
ISSUER = "http://127.0.0.1:50051"
TOKEN_TTL = datetime.timedelta(hours=1)
_lock = threading.Lock()

def generate_token(uid, rol):
    now = datetime.datetime.now(datetime.timezone.utc)
    payload = {
        "iss": ISSUER,
        "sub": str(uid),
        "exp": now + TOKEN_TTL,
        "jti": str(uuid7()),
        "role": rol,
    }
    return jwt.encode(payload, SECRET_KEY, algorithm=algorithm)

def add_to_blacklist(token: str):
    with _lock:
        blacklist[token] = datetime.datetime.now(datetime.timezone.utc)


def unparse_token(token: str):
    with _lock:
        if token in blacklist:
            raise jwt.InvalidTokenError("Token-ul a fost revocat.")
    return jwt.decode(
        token,
        SECRET_KEY,
        algorithms=[algorithm],
        issuer=ISSUER,
        options={"require": ["iss", "sub", "exp", "jti", "role"]},
    )



