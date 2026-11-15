package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/httpx"
	"job-management-service/internal/repository/memory"
)

const (
	adminToken      = "token-admin"
	maintainerToken = "token-maintainer"

	// Conturi din A&A Services simulate de fakeAccounts.
	anaUID         = "0190f2a8-7c1e-7a3b-9d4e-2f6a8b1c3d5e" // maintainer; e si contul din maintainerToken
	bossUID        = "0190f2a8-b055-7000-8000-000000000001" // un al doilea administrator
	unavailableUID = "0190f2a8-dead-7000-8000-000000000000" // simuleaza A&A oprit
)

// fakeValidator raspunde in locul ValidateToken, fara sa apeleze A&A Services.
type fakeValidator map[string]auth.Identity

func (f fakeValidator) Validate(_ context.Context, token string) (auth.Identity, error) {
	if identity, ok := f[token]; ok {
		return identity, nil
	}
	return auth.Identity{}, auth.ErrInvalidToken
}

// fakeAccounts raspunde in locul ListUsers: uid -> cont.
type fakeAccounts map[string]auth.Account

func (f fakeAccounts) FindAccount(_ context.Context, token, uid string) (auth.Account, error) {
	switch {
	case token != adminToken:
		// Verifica si ca handler-ul trimite mai departe token-ul adminului.
		return auth.Account{}, auth.ErrInvalidToken
	case uid == unavailableUID:
		return auth.Account{}, auth.ErrUnavailable
	}
	if account, ok := f[uid]; ok {
		return account, nil
	}
	return auth.Account{}, auth.ErrAccountNotFound
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	return NewRouter(Dependencies{
		Validator: fakeValidator{
			adminToken:      {UID: "uid-admin", Role: auth.RoleAdmin},
			maintainerToken: {UID: anaUID, Role: auth.RoleMaintainer},
		},
		Accounts: fakeAccounts{
			anaUID:  {UID: anaUID, Username: "ana", Role: auth.RoleMaintainer, IsActive: true},
			bossUID: {UID: bossUID, Username: "boss", Role: auth.RoleAdmin, IsActive: true},
		},
		Tx:        memory.NewTransactor(),
		Scripts:   memory.NewScriptRepository(),
		Users:     memory.NewUserRepository(),
		Computers: memory.NewComputerRepository(),
		Clusters:  memory.NewClusterRepository(),
	})
}

// send trimite o cerere; un corp nevid e trimis ca application/json.
func send(router *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("corpul nu e JSON valid: %v (%s)", err, rec.Body)
	}
	return value
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, vrem %d (corp: %s)", rec.Code, want, rec.Body)
	}
}

// step este o cerere dintr-un scenariu. Pasii ruleaza in ordine, pe acelasi router,
// deci un pas vede ce au creat pasii dinaintea lui.
type step struct {
	name       string
	method     string
	path       string
	token      string
	body       string
	wantStatus int
	wantField  string // primul camp din "errors" (la 409, 415, 422); "" = nu verificam
}

func runSteps(t *testing.T, router *gin.Engine, steps []step) {
	t.Helper()
	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			rec := send(router, s.method, s.path, s.token, s.body)
			expectStatus(t, rec, s.wantStatus)
			if s.wantField == "" {
				return
			}
			body := decode[struct {
				Errors []httpx.FieldError `json:"errors"`
			}](t, rec)
			if len(body.Errors) == 0 || body.Errors[0].Field != s.wantField {
				t.Errorf("errors = %+v, vrem primul camp %q", body.Errors, s.wantField)
			}
		})
	}
}
