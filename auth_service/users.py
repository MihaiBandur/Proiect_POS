import bcrypt
from uuid6 import uuid7
import threading

ROLE_ADMIN = 1
ROLE_MAINTAINER = 2



def _new_user(username: str, rol: int, is_active:bool = False, password = None):
    return{
        "uid": uuid7(),
        "username": username,
        "password": password,
        "rol": rol,
        "is_active": is_active
    }

_users = {}
for _u in (
    _new_user("admin", ROLE_ADMIN, is_active= True, password = "$2b$12$NCreP1CpHpdXCbicttpKv.rjUPfoz/4K9ZE0GTT9lhSbpPAPuJoI2"),
    # maintainer activ - pentru testarea Login
    _new_user("maintainer1", ROLE_MAINTAINER, is_active = False, password = "$2b$12$NCreP1CpHpdXCbicttpKv.rjUPfoz/4K9ZE0GTT9lhSbpPAPuJoI2"),
    # maintainer creat de admin, inca neactivat - pentru testarea ActivateAccount
    _new_user("maintainer2", ROLE_MAINTAINER, is_active = True, password = "$2b$12$NCreP1CpHpdXCbicttpKv.rjUPfoz/4K9ZE0GTT9lhSbpPAPuJoI2"),
):
    _users[_u["uid"]] = _u

_lock = threading.Lock()

def search_user(username: str):
    with _lock:
        for user in _users.values():
            if user["username"] == username:
                return dict(user)
        return None


def verify_password(entered_password: str, hashed_password: str):
        if bcrypt.checkpw(entered_password.encode(), hashed_password.encode()):
            return True
        return False

def list_users():
    with _lock:
        return [dict(user) for user in _users.values()]


def create_user(username: str, rol: int):
    with _lock:
        if any(u["username"] == username for u in _users.values()):
            return None
        user = _new_user(username, rol)
        _users[user["uid"]] = user
        return dict(user)


def set_active(username: str, is_active: bool) -> bool:
    with _lock:
        for user in _users.values():
            if user["username"] == username:
                user["is_active"] = is_active
                return True
    return False

def set_password(username: str, password: str) -> bool:
    s = bcrypt.gensalt()
    hash_password = bcrypt.hashpw(password.encode(), s)
    with _lock:
        for user in _users.values():
            if user["username"] == username:
                user["password"] = hash_password.decode()
                return True
        return False