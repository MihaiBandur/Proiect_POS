package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"job-management-service/internal/model"
	"job-management-service/internal/repository"
)

// downScripts simuleaza o baza de date oprita: List intoarce ErrUnavailable. Celelalte metode
// vin din interfata inclusa (nil) si nu sunt apelate in test.
type downScripts struct {
	repository.ScriptRepository
}

func (downScripts) List(context.Context, int, int) ([]model.Script, int, error) {
	return nil, 0, fmt.Errorf("%w: server selection timeout", repository.ErrUnavailable)
}

func TestStorageUnavailable(t *testing.T) {
	router := newTestRouterWith(func(deps *Dependencies) { deps.Scripts = downScripts{} })

	rec := send(router, http.MethodGet, scriptsPath, adminToken, "")
	expectStatus(t, rec, http.StatusServiceUnavailable)
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, vrem application/problem+json", got)
	}
}
