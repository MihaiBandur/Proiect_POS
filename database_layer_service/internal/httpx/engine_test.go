package httpx

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	// Ascundem in teste log-ul cererilor si stiva afisata la panic.
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	engine := NewEngine()
	engine.GET("/resursa", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.PUT("/resursa", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/panica", func(c *gin.Context) { panic("ceva neprevazut") })
	return engine
}

func TestEngine(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{"ruta existenta", http.MethodGet, "/resursa", http.StatusOK, ""},
		{"ruta inexistenta", http.MethodGet, "/nu-exista", http.StatusNotFound, ""},
		{"metoda nepermisa", http.MethodDelete, "/resursa", http.StatusMethodNotAllowed, "GET, PUT"},
		{"panic in handler", http.MethodGet, "/panica", http.StatusInternalServerError, ""},
	}

	engine := newTestEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, vrem %d (corp: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, vrem %q", got, tt.wantAllow)
			}
			if rec.Code < 400 {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != problemContentType {
				t.Errorf("Content-Type = %q, vrem %s", got, problemContentType)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("corpul nu e JSON valid: %v", err)
			}
			if body["type"] != "about:blank" || body["status"] != float64(tt.wantStatus) || body["instance"] != tt.path {
				t.Errorf("corp problem+json gresit: %s", rec.Body)
			}
		})
	}
}
