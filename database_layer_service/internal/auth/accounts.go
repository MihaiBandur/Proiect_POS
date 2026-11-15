package auth

import (
	"context"
	"errors"
)

type Account struct {
	UID      string
	Username string
	Role     Role
	IsActive bool
}

var ErrAccountNotFound = errors.New("contul nu exista in A&A Services")

type AccountDirectory interface {
	FindAccount(ctx context.Context, token, uid string) (Account, error)
}
