package auth

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"job-management-service/internal/gen/authpb"
)

const validateTimeout = 3 * time.Second

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
	ctx, cancel := context.WithTimeout(ctx, validateTimeout)
	defer cancel()

	resp, err := v.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: token})
	if err != nil {
		switch status.Code(err) {
		case codes.Unauthenticated:
			return Identity{}, ErrInvalidToken
		case codes.Unavailable, codes.DeadlineExceeded:
			return Identity{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		default:
			return Identity{}, fmt.Errorf("ValidateToken: %w", err)
		}
	}
	return Identity{UID: resp.GetUid(), Role: roleFromProto(resp.GetRole())}, nil
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
