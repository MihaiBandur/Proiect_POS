package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{"id valid", "6730f1a2c3d4e5f6a7b8c9d0", http.StatusOK},
		{"litere mari", "6730F1A2C3D4E5F6A7B8C9D0", http.StatusOK},
		{"prea scurt", "6730f1a2", http.StatusBadRequest},
		{"caractere care nu sunt hex", "6730f1a2c3d4e5f6a7b8c9zz", http.StatusBadRequest},
		{"doar zerouri", "000000000000000000000000", http.StatusBadRequest},
	}

	engine := newTestEngine()
	engine.GET("/resurse/:id", func(c *gin.Context) {
		id, ok := ParseID(c, "id")
		if !ok {
			return
		}
		c.String(http.StatusOK, id.Hex())
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/resurse/"+tt.id, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, vrem %d (corp: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}
