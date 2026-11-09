package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeValidator map[string]Identity

func (f fakeValidator) Validate(_ context.Context, token string) (Identity, error) {
	if token == "auth-cazut" {
		return Identity{}, ErrUnavailable
	}
	if identity, ok := f[token]; ok {
		return identity, nil
	}
	return Identity{}, ErrInvalidToken
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	validator := fakeValidator{
		"token-admin":      {UID: "uid-admin", Role: RoleAdmin},
		"token-maintainer": {UID: "uid-maintainer", Role: RoleMaintainer},
	}
	router := gin.New()
	group := router.Group("/", Middleware(validator))
	group.GET("/oricine", func(c *gin.Context) { c.String(http.StatusOK, FromContext(c).UID) })
	group.GET("/admin", RequireAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		authorization string
		wantStatus    int
		wantChallenge string
	}{
		{"fara header", "/oricine", "", http.StatusUnauthorized, "Bearer"},
		{"alta schema", "/oricine", "Basic YWRtaW46cGFyb2xh", http.StatusUnauthorized, "Bearer"},
		{"token invalid", "/oricine", "Bearer gresit", http.StatusUnauthorized, `Bearer error="invalid_token"`},
		{"auth indisponibil", "/oricine", "Bearer auth-cazut", http.StatusServiceUnavailable, ""},
		{"maintainer pe ruta comuna", "/oricine", "Bearer token-maintainer", http.StatusOK, ""},
		{"schema scrisa cu litere mici", "/oricine", "bearer token-admin", http.StatusOK, ""},
		{"maintainer pe ruta de admin", "/admin", "Bearer token-maintainer", http.StatusForbidden, ""},
		{"admin pe ruta de admin", "/admin", "Bearer token-admin", http.StatusOK, ""},
	}

	router := newTestRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, vrem %d (corp: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != tt.wantChallenge {
				t.Errorf("WWW-Authenticate = %q, vrem %q", got, tt.wantChallenge)
			}
			if rec.Code >= 400 {
				if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
					t.Errorf("Content-Type = %q, vrem application/problem+json", got)
				}
			}
		})
	}
}
