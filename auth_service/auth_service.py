import auth_pb2_grpc
import auth_pb2
import grpc


class Auth_Service(auth_pb2_grpc.AuthServicer):
    def Login(self, request, context):
        return super().Login(request, context)

    def ListUsers(self, request, context):
        return super().ListUsers(request, context)

    def AddAccount(self, request, context):
        return super().AddAccount(request, context)

    def ActivateAccount(self, request, context):
        return super().ActivateAccount(request, context)

    def DeactivateAccount(self, request, context):
        return super().DeactivateAccount(request, context)

    def ValidateToken(self, request, context):
        return super().ValidateToken(request, context)

    def DestroyToken(self, request, context):
        return super().DestroyToken(request, context)

