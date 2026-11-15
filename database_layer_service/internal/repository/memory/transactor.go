package memory

import (
	"context"
	"sync"

	"job-management-service/internal/repository"
)

// transactor serializeaza operatiile care scriu in mai multe colectii (asignari, stergeri in
// cascada): doar una ruleaza la un moment dat, deci nu se pot intercala. Spre deosebire de o
// tranzactie MongoDB nu poate anula scrierile deja facute, asa ca serviciul verifica totul
// inainte sa scrie.
type transactor struct {
	mu sync.Mutex
}

func NewTransactor() repository.Transactor {
	return &transactor{}
}

func (t *transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fn(ctx)
}
