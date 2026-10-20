import os
import uuid
import bcrypt
import pymysql
from pymysql.constants import CLIENT, ER
from dotenv import load_dotenv

load_dotenv()

DB_CONFIG = {
    "host": os.getenv("DB_HOST", "127.0.0.1"),
    "port": int(os.getenv("DB_PORT", "3306")),
    "user": os.getenv("DB_USER", "python_user"),
    "password": os.getenv("MARIADB_PASSWORD"),
    "database": os.getenv("DB_NAME", "users"),
}

# In cod rolurile sunt 1/2 (ca enum-ul Role din auth.proto),
# in baza de date sunt ENUM('administrator', 'maintainer').
_ROLE_TO_DB = {1: "administrator", 2: "maintainer"}
_ROLE_FROM_DB = {v: k for k, v in _ROLE_TO_DB.items()}

def get_connection():
    return pymysql.connect(
        **DB_CONFIG,
        cursorclass=pymysql.cursors.DictCursor,
        autocommit=True,
        # Fara FOUND_ROWS, un UPDATE care nu schimba nimic (ex. is_active era deja FALSE)
        # raporteaza 0 randuri si nu s-ar putea distinge de "user inexistent".
        client_flag=CLIENT.FOUND_ROWS,
    )


def _row_to_user(row):
    # Acelasi format de dictionar ca in users.py.
    return {
        "uid": uuid.UUID(bytes=row["uid"]),
        "username": row["username"],
        "password": row["password"],
        "rol": _ROLE_FROM_DB[row["role"]],
        "is_active": bool(row["is_active"]),
    }


def find_user_in_db(username: str):
    """Intoarce dictionarul utilizatorului sau None daca nu exista."""
    with get_connection() as connection:
        with connection.cursor() as cursor:
            sql = "SELECT * FROM `users` WHERE `username` = %s"
            cursor.execute(sql, (username,))
            result = cursor.fetchone()
            return _row_to_user(result) if result else None


def list_users_in_db():
    with get_connection() as connection:
        with connection.cursor() as cursor:
            cursor.execute("SELECT * FROM `users`")
            return [_row_to_user(row) for row in cursor.fetchall()]


def insert_user_in_db(uid: uuid.UUID, username: str, rol: int) -> bool:
    """Cont nou cu password NULL si is_active FALSE. False daca username-ul exista deja."""
    with get_connection() as connection:
        with connection.cursor() as cursor:
            sql = (
                "INSERT INTO `users` (`uid`, `username`, `password`, `role`, `is_active`) "
                "VALUES (%s, %s, NULL, %s, FALSE)"
            )
            try:
                cursor.execute(sql, (uid.bytes, username, _ROLE_TO_DB[rol]))
            except pymysql.err.IntegrityError as e:
                # Constrangerea UNIQUE pe username verifica duplicatele atomic,
                # deci nu e nevoie de un SELECT inainte.
                if e.args[0] == ER.DUP_ENTRY:
                    return False
                raise
            return True


def set_active_in_db(username: str, is_active: bool) -> bool:
    with get_connection() as connection:
        with connection.cursor() as cursor:
            sql = "UPDATE `users` SET `is_active` = %s WHERE `username` = %s"
            return cursor.execute(sql, (is_active, username)) == 1


def activate_user_in_db(username: str, password: str) -> bool:
    """Seteaza parola si is_active=TRUE intr-un singur UPDATE, doar daca
    contul este inca inactiv si fara parola. False in orice alt caz."""
    s = bcrypt.gensalt()
    password_hash = bcrypt.hashpw(password=password.encode(), salt=s).decode()
    with get_connection() as connection:
        with connection.cursor() as cursor:
            sql = (
                "UPDATE `users` SET `password` = %s, `is_active` = TRUE "
                "WHERE `username` = %s AND `is_active` = FALSE AND `password` IS NULL"
            )
            return cursor.execute(sql, (password_hash, username)) == 1


def set_password_in_db(username: str, password_hash: str) -> bool:
    """Primeste hash-ul bcrypt deja calculat. False daca utilizatorul nu exista."""
    with get_connection() as connection:
        with connection.cursor() as cursor:
            sql = "UPDATE `users` SET `password` = %s WHERE `username` = %s"
            return cursor.execute(sql, (password_hash, username)) == 1


