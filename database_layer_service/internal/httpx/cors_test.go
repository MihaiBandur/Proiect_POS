package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	const allowed = "http://localhost:5173"
	// Use se aplica doar rutelor inregistrate DUPA el, deci CORS vine inaintea rutelor.
	gin.SetMode(gin.TestMode)
	engine := NewEngine()
	engine.Use(CORS([]string{allowed}))
	engine.GET("/resursa", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.PUT("/resursa", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	tests := []struct {
		name          string
		method        string
		origin        string
		requestMethod string // Access-Control-Request-Method (doar la preflight)
		wantStatus    int
		wantOrigin    string // Access-Control-Allow-Origin
		wantMethods   bool   // Access-Control-Allow-Methods prezent
		wantExposed   bool   // Access-Control-Expose-Headers prezent
	}{
		{"preflight de pe origine permisa", http.MethodOptions, allowed, http.MethodPut, http.StatusNoContent, allowed, true, false},
		{"preflight de pe alta origine", http.MethodOptions, "http://evil.example", http.MethodPut, http.StatusMethodNotAllowed, "", false, false},
		{"cerere de pe origine permisa", http.MethodGet, allowed, "", http.StatusOK, allowed, false, true},
		{"cerere de pe alta origine", http.MethodGet, "http://evil.example", "", http.StatusOK, "", false, false},
		{"cerere fara Origin (curl, Bruno)", http.MethodGet, "", "", http.StatusOK, "", false, false},
		{"OPTIONS care nu e preflight", http.MethodOptions, allowed, "", http.StatusMethodNotAllowed, allowed, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/resursa", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.requestMethod != "" {
				req.Header.Set("Access-Control-Request-Method", tt.requestMethod)
				req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
			}
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			h := rec.Header()
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, vrem %d", rec.Code, tt.wantStatus)
			}
			if got := h.Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Allow-Origin = %q, vrem %q", got, tt.wantOrigin)
			}
			if got := h.Get("Access-Control-Allow-Methods") != ""; got != tt.wantMethods {
				t.Errorf("Allow-Methods prezent = %v, vrem %v", got, tt.wantMethods)
			}
			if got := h.Get("Access-Control-Expose-Headers") != ""; got != tt.wantExposed {
				t.Errorf("Expose-Headers prezent = %v, vrem %v", got, tt.wantExposed)
			}
			if tt.origin != "" && !strings.Contains(strings.Join(h.Values("Vary"), ","), "Origin") {
				t.Errorf("lipseste Vary: Origin (%v)", h.Values("Vary"))
			}
		})
	}
}
