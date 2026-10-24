import re

DOMAIN = "academic.tuiasi.ro"

MAX_LOCAL_PART = 64

_NAME = r"[a-z]+(?:-[a-z]+)*"

NEW_USERNAME_RE = re.compile(rf"{_NAME}\.{_NAME}@academic\.tuiasi\.ro")

USERNAME_RE = re.compile(rf"{_NAME}\.{_NAME}(?:[1-9][0-9]*)?@academic\.tuiasi\.ro")


def clean(username: str) -> str:
    return username.strip().lower()

def add_domain(username: str) -> str:
    if "@" not in username:
        return f"{username}@{DOMAIN}"
    return username

def _short_enough(username: str) -> bool:
    return len(username.partition("@")[0]) <= MAX_LOCAL_PART

def is_valid_new_username(username: str) -> bool:
    return _short_enough(username) and NEW_USERNAME_RE.fullmatch(username) is not None

def is_valid_username(username: str) -> bool:
    return _short_enough(username) and USERNAME_RE.fullmatch(username) is not None

