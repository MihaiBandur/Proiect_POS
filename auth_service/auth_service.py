import auth_pb2_grpc
import auth_pb2
import grpc
from concurrent import futures
import db_connection
import users
import token_service
import jwt
from google.protobuf import empty_pb2
from uuid6 import uuid7


class AuthService(auth_pb2_grpc.AuthServicer):
    def _decode_token(self, token, context, error_code):
        if len(token) == 0:
            context.abort(error_code, "Token-ul lipseste.")
        try:
            return token_service.unparse_token(token)
        except jwt.ExpiredSignatureError:
            token_service.add_to_blacklist(token)
            context.abort(error_code, "Token-ul a expirat.")
        except jwt.InvalidTokenError:
            token_service.add_to_blacklist(token)
            context.abort(error_code, "Token-ul nu este valid.")

    def _require_admin(self, token, context):
        payload = self._decode_token(token, context, grpc.StatusCode.PERMISSION_DENIED)
        if payload["role"] != users.ROLE_ADMIN:
            context.abort(grpc.StatusCode.PERMISSION_DENIED, "Operatia este permisa doar administratorului.")
        return payload

    def Login(self, request, context):
        username = request.username
        password = request.password
        if(len(password) == 0 or len(username) == 0):
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "User-ul si parola trebuie sa fie campuri valide.")
        user = db_connection.find_user_in_db(username=username)
        if user is None:
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "User-ul sau parola nu sunt corecte.")
        if user["is_active"] == False:
            context.abort(grpc.StatusCode.PERMISSION_DENIED, "User-ul nu este activ.")
        if users.verify_password(password, user["password"]) == False:
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "User-ul sau parola nu sunt corecte.")
        token_jws = token_service.generate_token(user["uid"], user["rol"])
        return auth_pb2.LoginResponse(token=token_jws)




    def ListUsers(self, request, context):
        self._require_admin(request.token, context)
        accounts = [
            auth_pb2.Account(uid=str(u["uid"]), username=u["username"],
                     role=u["rol"], is_active=u["is_active"])
            for u in db_connection.list_users_in_db()
]
        return auth_pb2.ListUsersResponse(accounts=accounts)

    def AddAccount(self, request, context):
        self._require_admin(token=request.token, context=context)
        username = request.username
        rol=request.role
        if(len(username)) == 0:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "User-ul trebuie sa fie valid.")
        if request.role not in (users.ROLE_ADMIN, users.ROLE_MAINTAINER):
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Alegeti un rol de maintainer sau de admin.")
        uid = uuid7()
        is_added = db_connection.insert_user_in_db(uid=uid, username=username, rol=rol)
        if is_added == False:
            context.abort(grpc.StatusCode.ALREADY_EXISTS, f"Deja exista cineva cu acest username {username}.")
        return auth_pb2.CreateAccountResponse(uid=str(uid))




    def ActivateAccount(self, request, context):
        username = request.username
        password = request.password
        if len(username) == 0  or len(password) == 0:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Introduceti un camp valid pentru username sau pentru parola")
        user_to_activate = db_connection.find_user_in_db(username=username)
        if user_to_activate is None:
            context.abort(grpc.StatusCode.NOT_FOUND, "Nu exista acest user in baza de date.")
        if user_to_activate["is_active"] == True:
            context.abort(grpc.StatusCode.PERMISSION_DENIED, "Acest user are deja contul activ.")
        if user_to_activate["is_active"] == False and user_to_activate["password"] != None:
            context.abort(grpc.StatusCode.PERMISSION_DENIED, "Acest user are deja o parola setat si are contul dezactivat")
        if db_connection.activate_user_in_db(username=username, password=password) == False:
            context.abort(grpc.StatusCode.PERMISSION_DENIED, "Contul nu mai poate fi activat.")

        return empty_pb2.Empty()


    def DeactivateAccount(self, request, context):
        token_admin = request.token 
        username = request.username
        payload = self._require_admin(token=token_admin, context=context)
        if len(username) == 0:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "User-ul trebuie sa fie valid.")
        user_to_inactivate = db_connection.find_user_in_db(username=username)
        if user_to_inactivate is None:
            context.abort(grpc.StatusCode.NOT_FOUND, "Nu exista acest user in baza de date")
        if str(user_to_inactivate["uid"]) == payload["sub"]:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "Nu va puteti dezactiva propriul cont.")
        succeded = db_connection.set_active_in_db(username=username, is_active=False)
        if succeded == False:
            context.abort(grpc.StatusCode.NOT_FOUND, "Nu exista acest user in baza de date")
        
        return empty_pb2.Empty()
            

    def ValidateToken(self, request, context):
        payload = self._decode_token(request.token, context, grpc.StatusCode.UNAUTHENTICATED)
        return auth_pb2.ValidateTokenResponse(uid=payload["sub"], role=payload["role"])

    def DestroyToken(self, request, context):
        self._decode_token(request.token, context, grpc.StatusCode.UNAUTHENTICATED)
        token_service.add_to_blacklist(request.token)
        return empty_pb2.Empty()


def serve():
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10)
    )

    auth_pb2_grpc.add_AuthServicer_to_server(
        AuthService(),
        server=server
    )

    server.add_insecure_port("127.0.0.1:50051")

    server.start()

    print("Auth grpc server started")
    print("Listenig on 127.0.0.1:50051")

    try:
        server.wait_for_termination()
    except  KeyboardInterrupt:
        server.stop(grace=2).wait()

if __name__== "__main__":
    serve()