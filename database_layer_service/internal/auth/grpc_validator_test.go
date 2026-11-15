package auth

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"job-management-service/internal/gen/authpb"
)

// fakeAuthClient inlocuieste clientul gRPC generat. Implementeaza doar ListUsers;
// celelalte metode vin din interfata inclusa (nil) si nu sunt apelate in test.
type fakeAuthClient struct {
	authpb.AuthClient
	accounts []*authpb.Account
	err      error
}

func (f fakeAuthClient) ListUsers(context.Context, *authpb.ListUsersRequest, ...grpc.CallOption) (*authpb.ListUsersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &authpb.ListUsersResponse{Accounts: f.accounts}, nil
}

func TestFindAccount(t *testing.T) {
	const uid = "0190f2a8-7c1e-7a3b-9d4e-2f6a8b1c3d5e"
	accounts := []*authpb.Account{
		{Uid: "0190f2a8-0000-7000-8000-000000000001", Username: "admin", Role: authpb.Role_ADMIN, IsActive: true},
		{Uid: uid, Username: "ana", Role: authpb.Role_MAINTAINER, IsActive: true},
	}

	tests := []struct {
		name     string
		client   fakeAuthClient
		uid      string
		wantUser string
		wantErr  error
	}{
		{"cont gasit", fakeAuthClient{accounts: accounts}, uid, "ana", nil},
		{"uid cu litere mari", fakeAuthClient{accounts: accounts}, "0190F2A8-7C1E-7A3B-9D4E-2F6A8B1C3D5E", "ana", nil},
		{"cont inexistent", fakeAuthClient{accounts: accounts}, "0190f2a8-ffff-7fff-8fff-ffffffffffff", "", ErrAccountNotFound},
		{"A&A oprit", fakeAuthClient{err: status.Error(codes.Unavailable, "oprit")}, uid, "", ErrUnavailable},
		{"token respins", fakeAuthClient{err: status.Error(codes.PermissionDenied, "nu e admin")}, uid, "", ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &GRPCValidator{client: tt.client}
			account, err := validator.FindAccount(context.Background(), "token-admin", tt.uid)
			if !errors.Is(err, tt.wantErr) || account.Username != tt.wantUser {
				t.Errorf("FindAccount = %+v, %v; vrem %q, %v", account, err, tt.wantUser, tt.wantErr)
			}
		})
	}
}
