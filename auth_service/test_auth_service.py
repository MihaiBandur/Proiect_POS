"""
Teste de integrare pentru serviciul gRPC de autentificare (Laborator 2).

Ruleaza serverul gRPC pe un port liber (nu intra in conflict cu serverul pornit
manual pe 50051) si foloseste baza de date MariaDB reala din docker-compose.
Conturile create de teste au prefixul "zz_test_" si sunt sterse la final.

Rulare (din folderul auth_service, cu containerul mariadb pornit):
    python -m unittest test_auth_service -v

Parola contului admin din baza de date se poate da prin TEST_ADMIN_PASSWORD
(implicit "parola", cea din db/init/01_schema.sql).
"""
import datetime
import os
import unittest
import uuid
from concurrent import futures

import bcrypt
import grpc
import jwt
from uuid6 import uuid7

import auth_pb2
import auth_pb2_grpc
import auth_service
import db_connection
import token_service

ADMIN_USERNAME = "admin"
ADMIN_PASSWORD = os.getenv("TEST_ADMIN_PASSWORD", "parola")
TEST_PREFIX = "zz_test_"

_server = None
_stub = None
_created_usernames = []


def setUpModule():
    global _server, _stub
    _server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    auth_pb2_grpc.add_AuthServicer_to_server(auth_service.AuthService(), _server)
    port = _server.add_insecure_port("127.0.0.1:0")
    _server.start()
    _stub = auth_pb2_grpc.AuthStub(grpc.insecure_channel(f"127.0.0.1:{port}"))


def tearDownModule():
    with db_connection.get_connection() as connection:
        with connection.cursor() as cursor:
            for username in _created_usernames:
                cursor.execute("DELETE FROM `users` WHERE `username` = %s", (username,))
    _server.stop(0)


# ---------------------------------------------------------------- utilitare

def new_username():
    username = TEST_PREFIX + uuid.uuid4().hex[:10]
    _created_usernames.append(username)
    return username


def make_user(state, password="parola_test", rol=2):
    """Creeaza direct in baza un utilizator intr-una din starile:
    'new'         - is_active=FALSE, password NULL (creat de admin, neactivat)
    'active'      - is_active=TRUE, cu parola
    'deactivated' - is_active=FALSE, cu parola (dezactivat de admin)
    """
    username = new_username()
    db_connection.insert_user_in_db(uid=uuid7(), username=username, rol=rol)
    if state in ("active", "deactivated"):
        db_connection.activate_user_in_db(username=username, password=password)
    if state == "deactivated":
        db_connection.set_active_in_db(username=username, is_active=False)
    return username


def login(username, password):
    return _stub.Login(auth_pb2.LoginRequest(username=username, password=password)).token


def admin_token():
    return login(ADMIN_USERNAME, ADMIN_PASSWORD)


def maintainer_token():
    password = "parola_test"
    return login(make_user("active", password=password), password)


def expired_token():
    payload = {
        "iss": token_service.ISSUER,
        "sub": str(uuid7()),
        "exp": datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(minutes=1),
        "jti": str(uuid7()),
        "role": 1,
    }
    return jwt.encode(payload, token_service.SECRET_KEY, algorithm="HS256")


def tampered_token():
    # Payload modificat (role=1) cu semnatura veche: integritatea nu mai e valida.
    header, _, signature = admin_token().split(".")
    fake_payload = jwt.utils.base64url_encode(
        b'{"iss":"%s","sub":"x","exp":9999999999,"jti":"x","role":1}' % token_service.ISSUER.encode()
    ).decode()
    return f"{header}.{fake_payload}.{signature}"


def forged_token():
    # Semnat cu alta cheie decat cea a serviciului.
    payload = {"iss": token_service.ISSUER, "sub": "x", "jti": "x", "role": 1,
               "exp": datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=1)}
    return jwt.encode(payload, "o-cheie-gresita-de-cel-putin-32-de-bytes!!", algorithm="HS256")


class GrpcTestCase(unittest.TestCase):
    def assertGrpcError(self, code, fn, *args, **kwargs):
        with self.assertRaises(grpc.RpcError) as cm:
            fn(*args, **kwargs)
        self.assertEqual(cm.exception.code(), code,
                         f"Asteptat {code.name}, primit {cm.exception.code().name}: {cm.exception.details()}")
        return cm.exception

    def assertBlacklisted(self, token):
        self.assertIn(token, token_service.blacklist, "Token-ul invalid nu a fost trecut in blacklist")


# ---------------------------------------------------------------- 1. Login

class TestLogin(GrpcTestCase):
    def test_admin_primeste_token_cu_claim_urile_cerute(self):
        token = admin_token()
        claims = jwt.decode(token, token_service.SECRET_KEY, algorithms=["HS256"],
                            issuer=token_service.ISSUER)
        admin = db_connection.find_user_in_db(ADMIN_USERNAME)
        self.assertEqual(claims["iss"], token_service.ISSUER)
        self.assertEqual(claims["sub"], str(admin["uid"]))
        self.assertEqual(claims["role"], auth_pb2.ADMIN)
        uuid.UUID(claims["jti"])  # jti trebuie sa fie UUID
        ttl = claims["exp"] - datetime.datetime.now(datetime.timezone.utc).timestamp()
        self.assertAlmostEqual(ttl, 3600, delta=60, msg="Token-ul trebuie sa fie valid o ora")

    def test_doua_login_uri_au_jti_diferit(self):
        a = jwt.decode(admin_token(), options={"verify_signature": False})
        b = jwt.decode(admin_token(), options={"verify_signature": False})
        self.assertNotEqual(a["jti"], b["jti"])

    def test_maintainer_activ_primeste_token_cu_rol_maintainer(self):
        claims = jwt.decode(maintainer_token(), options={"verify_signature": False})
        self.assertEqual(claims["role"], auth_pb2.MAINTAINER)

    def test_parola_gresita(self):
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, login, ADMIN_USERNAME, "gresita")

    def test_user_inexistent(self):
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, login, TEST_PREFIX + "nu_exista", "x")

    def test_campuri_goale(self):
        self.assertGrpcError(grpc.StatusCode.INVALID_ARGUMENT, login, "", "")

    def test_cont_nou_neactivat_fara_parola(self):
        # Laborator: is_active=FALSE -> PERMISSION_DENIED
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, login, make_user("new"), "orice")

    def test_cont_dezactivat_cu_parola_corecta(self):
        username = make_user("deactivated", password="p")
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, login, username, "p")


# ---------------------------------------------------------------- 2. ListUsers

class TestListUsers(GrpcTestCase):
    def list_users(self, token):
        return _stub.ListUsers(auth_pb2.ListUsersRequest(token=token)).accounts

    def test_admin_primeste_lista_cu_campurile_cerute(self):
        username = make_user("new")
        accounts = {a.username: a for a in self.list_users(admin_token())}

        admin = accounts[ADMIN_USERNAME]
        self.assertEqual(admin.role, auth_pb2.ADMIN)
        self.assertTrue(admin.is_active)
        self.assertEqual(admin.uid, str(db_connection.find_user_in_db(ADMIN_USERNAME)["uid"]))

        self.assertIn(username, accounts, "Contul creat in baza nu apare in lista")
        self.assertEqual(accounts[username].role, auth_pb2.MAINTAINER)
        self.assertFalse(accounts[username].is_active)

    def test_maintainer(self):
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.list_users, maintainer_token())

    def test_token_gol(self):
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.list_users, "")

    def test_token_corupt_ajunge_in_blacklist(self):
        token = "abc.def.ghi"
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.list_users, token)
        self.assertBlacklisted(token)

    def test_token_alterat_ajunge_in_blacklist(self):
        token = tampered_token()
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.list_users, token)
        self.assertBlacklisted(token)

    def test_token_expirat_ajunge_in_blacklist(self):
        token = expired_token()
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.list_users, token)
        self.assertBlacklisted(token)


# ---------------------------------------------------------------- 3. AddAccount

class TestAddAccount(GrpcTestCase):
    def add(self, token, username, role=auth_pb2.MAINTAINER):
        return _stub.AddAccount(auth_pb2.CreateAccountRequest(token=token, username=username, role=role))

    def test_cont_nou_inactiv_fara_parola_cu_uuid7(self):
        username = new_username()
        response = self.add(admin_token(), username)

        uid = uuid.UUID(response.uid)
        self.assertEqual(uid.version, 7, "uid-ul trebuie sa fie UUIDv7")

        user = db_connection.find_user_in_db(username)
        self.assertIsNotNone(user, "Contul nu a fost salvat in baza de date")
        self.assertEqual(user["uid"], uid)
        self.assertFalse(user["is_active"])
        self.assertIsNone(user["password"])
        self.assertEqual(user["rol"], auth_pb2.MAINTAINER)

    def test_username_duplicat(self):
        username = make_user("new")
        self.assertGrpcError(grpc.StatusCode.ALREADY_EXISTS, self.add, admin_token(), username)

    def test_username_gol(self):
        self.assertGrpcError(grpc.StatusCode.INVALID_ARGUMENT, self.add, admin_token(), "")

    def test_rol_nespecificat(self):
        self.assertGrpcError(grpc.StatusCode.INVALID_ARGUMENT, self.add,
                             admin_token(), new_username(), auth_pb2.ROLE_UNSPECIFIED)

    def test_maintainer_nu_poate_crea_conturi(self):
        username = new_username()
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.add, maintainer_token(), username)
        self.assertIsNone(db_connection.find_user_in_db(username))

    def test_token_invalid_ajunge_in_blacklist(self):
        token = "q.w.e"
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.add, token, new_username())
        self.assertBlacklisted(token)


# ---------------------------------------------------------------- 4. DeactivateAccount

class TestDeactivateAccount(GrpcTestCase):
    def deactivate(self, token, username):
        return _stub.DeactivateAccount(auth_pb2.DeactivateAccountRequest(token=token, username=username))

    def test_dezactivare_cont_activ(self):
        username = make_user("active", password="p")
        self.deactivate(admin_token(), username)
        self.assertFalse(db_connection.find_user_in_db(username)["is_active"])
        # dupa dezactivare nu se mai poate loga
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, login, username, "p")

    def test_cont_deja_inactiv(self):
        self.deactivate(admin_token(), make_user("deactivated"))

    def test_adminul_nu_se_poate_dezactiva_singur(self):
        self.assertGrpcError(grpc.StatusCode.FAILED_PRECONDITION, self.deactivate,
                             admin_token(), ADMIN_USERNAME)
        self.assertTrue(db_connection.find_user_in_db(ADMIN_USERNAME)["is_active"])

    def test_user_inexistent(self):
        self.assertGrpcError(grpc.StatusCode.NOT_FOUND, self.deactivate,
                             admin_token(), TEST_PREFIX + "nu_exista")

    def test_username_gol(self):
        self.assertGrpcError(grpc.StatusCode.INVALID_ARGUMENT, self.deactivate, admin_token(), "")

    def test_maintainer(self):
        username = make_user("active")
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.deactivate, maintainer_token(), username)
        self.assertTrue(db_connection.find_user_in_db(username)["is_active"])

    def test_token_invalid_ajunge_in_blacklist(self):
        token = expired_token()
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.deactivate, token, make_user("active"))
        self.assertBlacklisted(token)


# ---------------------------------------------------------------- 5. ActivateAccount

class TestActivateAccount(GrpcTestCase):
    def activate(self, username, password):
        return _stub.ActivateAccount(auth_pb2.ActivateAccountRequest(username=username, password=password))

    def test_activare_cont_nou(self):
        username = make_user("new")
        response = self.activate(username, "parola_noua")

        # Laborator: NU se genereaza token la activare
        self.assertEqual(response.ByteSize(), 0)

        user = db_connection.find_user_in_db(username)
        self.assertTrue(user["is_active"])
        self.assertIsInstance(user["password"], str)
        self.assertNotEqual(user["password"], "parola_noua", "Parola nu trebuie stocata in clar")
        self.assertTrue(bcrypt.checkpw(b"parola_noua", user["password"].encode()))
        # dupa activare se poate loga
        self.assertTrue(login(username, "parola_noua"))

    def test_cont_deja_activ(self):
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.activate, make_user("active"), "x")

    def test_cont_inactiv_care_are_deja_parola(self):
        username = make_user("deactivated", password="veche")
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, self.activate, username, "noua")
        user = db_connection.find_user_in_db(username)
        self.assertFalse(user["is_active"])
        self.assertTrue(bcrypt.checkpw(b"veche", user["password"].encode()), "Parola veche a fost suprascrisa")

    def test_user_inexistent(self):
        self.assertGrpcError(grpc.StatusCode.NOT_FOUND, self.activate, TEST_PREFIX + "nu_exista", "x")

    def test_campuri_goale(self):
        self.assertGrpcError(grpc.StatusCode.INVALID_ARGUMENT, self.activate, "", "")


# ---------------------------------------------------------------- 6. ValidateToken

class TestValidateToken(GrpcTestCase):
    def validate(self, token):
        return _stub.ValidateToken(auth_pb2.ValidateTokenRequest(token=token))

    def test_token_admin(self):
        response = self.validate(admin_token())
        self.assertEqual(response.uid, str(db_connection.find_user_in_db(ADMIN_USERNAME)["uid"]))
        self.assertEqual(response.role, auth_pb2.ADMIN)

    def test_token_maintainer(self):
        self.assertEqual(self.validate(maintainer_token()).role, auth_pb2.MAINTAINER)

    def test_token_gol(self):
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.validate, "")

    def test_token_expirat_ajunge_in_blacklist(self):
        token = expired_token()
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.validate, token)
        self.assertBlacklisted(token)

    def test_token_alterat_ajunge_in_blacklist(self):
        token = tampered_token()
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.validate, token)
        self.assertBlacklisted(token)

    def test_token_semnat_cu_alta_cheie_ajunge_in_blacklist(self):
        token = forged_token()
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.validate, token)
        self.assertBlacklisted(token)

    def test_token_corupt(self):
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.validate, "nu-e-un-jwt")


# ---------------------------------------------------------------- 7. DestroyToken

class TestDestroyToken(GrpcTestCase):
    def destroy(self, token):
        return _stub.DestroyToken(auth_pb2.DestroyTokenRequest(token=token))

    def test_token_distrus_nu_mai_poate_fi_folosit(self):
        token = admin_token()
        self.destroy(token)
        self.assertBlacklisted(token)
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, _stub.ValidateToken,
                             auth_pb2.ValidateTokenRequest(token=token))
        self.assertGrpcError(grpc.StatusCode.PERMISSION_DENIED, _stub.ListUsers,
                             auth_pb2.ListUsersRequest(token=token))

    def test_distrugere_de_doua_ori(self):
        token = admin_token()
        self.destroy(token)
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.destroy, token)

    def test_alte_token_uri_raman_valide(self):
        token, other = admin_token(), admin_token()
        self.destroy(token)
        self.assertEqual(_stub.ValidateToken(auth_pb2.ValidateTokenRequest(token=other)).role, auth_pb2.ADMIN)

    def test_token_expirat_ajunge_in_blacklist(self):
        token = expired_token()
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.destroy, token)
        self.assertBlacklisted(token)

    def test_token_corupt_ajunge_in_blacklist(self):
        token = "x.y.z"
        self.assertGrpcError(grpc.StatusCode.UNAUTHENTICATED, self.destroy, token)
        self.assertBlacklisted(token)


if __name__ == "__main__":
    unittest.main(verbosity=2)
