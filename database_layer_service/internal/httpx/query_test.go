package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQueryParams(t *testing.T) {
	engine := newTestEngine()
	engine.GET("/filtre", func(c *gin.Context) {
		isActive, ok := QueryBool(c, "isActive")
		if !ok {
			return
		}
		verbose, ok := QueryFlag(c, "verbose")
		if !ok {
			return
		}
		active := "lipsa"
		if isActive != nil {
			active = fmt.Sprint(*isActive)
		}
		c.String(http.StatusOK, "%s %v", active, verbose)
	})

	tests := []struct {
		query      string
		wantStatus int
		wantBody   string
	}{
		{"", http.StatusOK, "lipsa false"},
		{"?isActive=true", http.StatusOK, "true false"},
		{"?isActive=false&verbose", http.StatusOK, "false true"},
		{"?verbose=true", http.StatusOK, "lipsa true"},
		{"?verbose=false", http.StatusOK, "lipsa false"},
		{"?isActive=da", http.StatusBadRequest, ""},
		{"?isActive=", http.StatusBadRequest, ""},
		{"?verbose=2", http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/filtre"+tt.query, nil))
			if rec.Code != tt.wantStatus || (tt.wantBody != "" && rec.Body.String() != tt.wantBody) {
				t.Errorf("= %d %q, vrem %d %q", rec.Code, rec.Body, tt.wantStatus, tt.wantBody)
			}
		})
	}
}
