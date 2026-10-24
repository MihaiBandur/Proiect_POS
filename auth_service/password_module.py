import bcrypt
from uuid6 import uuid7
import hashlib

ROLE_ADMIN = 1
ROLE_MAINTAINER = 2


def verify_password(entered_password: str, hashed_password: str):
        prehash_password = pre_hash_for_password(password=entered_password)
        if bcrypt.checkpw(prehash_password, hashed_password.encode()):
            return True
        return False

def pre_hash_for_password(password: str) -> bytes:
      m = hashlib.sha256()
      m.update(password.encode())
      return m.digest()


def generate_password(entered_password: str) -> str:
      s = bcrypt.gensalt()
      pre_hash_password = pre_hash_for_password(password=entered_password)
      return bcrypt.hashpw(password=pre_hash_password, salt = s).decode()




