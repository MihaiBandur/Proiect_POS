package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"job-management-service/internal/gen/authpb"
)

const rpcTimeout = 3 * time.Second

type GRPCValidator struct {
	conn   *grpc.ClientConn
	client authpb.AuthClient
}

func NewGRPCValidator(addr string) (*GRPCValidator, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("client gRPC pentru %s: %w", addr, err)
	}
	return &GRPCValidator{conn: conn, client: authpb.NewAuthClient(conn)}, nil
}

func (v *GRPCValidator) Close() error {
	return v.conn.Close()
}

func (v *GRPCValidator) Validate(ctx context.Context, token string) (Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	resp, err := v.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: token})
	if err != nil {
		return Identity{}, grpcError("ValidateToken", err)
	}
	return Identity{UID: resp.GetUid(), Role: roleFromProto(resp.GetRole())}, nil
}

func (v *GRPCValidator) FindAccount(ctx context.Context, token, uid string) (Account, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	resp, err := v.client.ListUsers(ctx, &authpb.ListUsersRequest{Token: token})
	if err != nil {
		return Account{}, grpcError("ListUsers", err)
	}
	for _, account := range resp.GetAccounts() {
		if strings.EqualFold(account.GetUid(), uid) {
			return Account{
				UID:      account.GetUid(),
				Username: account.GetUsername(),
				Role:     roleFromProto(account.GetRole()),
				IsActive: account.GetIsActive(),
			}, nil
		}
	}
	return Account{}, ErrAccountNotFound
}

// grpcError traduce codul unei erori gRPC in erorile pachetului auth.
func grpcError(method string, err error) error {
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied:
		return ErrInvalidToken
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	default:
		return fmt.Errorf("%s: %w", method, err)
	}
}

func roleFromProto(role authpb.Role) Role {
	switch role {
	case authpb.Role_ADMIN:
		return RoleAdmin
	case authpb.Role_MAINTAINER:
		return RoleMaintainer
	default:
		return RoleUnknown
	}
}
