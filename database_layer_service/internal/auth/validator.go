package auth

import (
	"context"
	"errors"
)

var (
	ErrInvalidToken = errors.New("token invalid")

	ErrUnavailable = errors.New("serviciul de autentificare nu este disponibil")
)

type TokenValidator interface {
	Validate(ctx context.Context, token string) (Identity, error)
}
