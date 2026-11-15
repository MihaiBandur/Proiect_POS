package service

import (
	"errors"

	"job-management-service/internal/repository"
)

// NotFoundError: o resursa din adresa nu exista (404). Detail spune care, ca la o legatura
// clientul sa afle daca lipseste calculatorul, scriptul sau chiar legatura dintre ele.
type NotFoundError struct {
	Detail string
}

func (e *NotFoundError) Error() string { return e.Detail }

// Unwrap face ca errors.Is(err, repository.ErrNotFound) sa fie adevarat si pentru aceasta eroare.
func (e *NotFoundError) Unwrap() error { return repository.ErrNotFound }

// ConflictError: operatia contrazice starea curenta a resursei (409, RFC 9110, 15.5.10),
// de exemplu un calculator care e deja in alt cluster.
type ConflictError struct {
	Detail string
}

func (e *ConflictError) Error() string { return e.Detail }

// ErrNotAssigned: un maintainer cere o resursa care nu i-a fost asignata (403).
var ErrNotAssigned = errors.New("resursa nu este asignata acestui maintainer")

// notFoundAs inlocuieste ErrNotFound din repository cu un mesaj care spune ce lipseste.
func notFoundAs(err error, detail string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return &NotFoundError{Detail: detail}
	}
	return err
}
