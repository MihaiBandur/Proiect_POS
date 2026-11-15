package dto

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/httpx"
)

// decodeScript trimite corpul prin httpx.DecodeJSON, ca in controller.
// Intoarce cererea decodata si codul de eroare (0 daca a fost acceptata).
func decodeScript(body string) (ScriptRequest, int) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/scripts", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	var req ScriptRequest
	if !httpx.DecodeJSON(c, &req) {
		return req, rec.Code
	}
	return req, 0
}

func TestScriptRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int // 0 = cerere acceptata
	}{
		{"complet", `{"name":"backup","store_path":"/srv/b.sh","store_comp":"/opt/b.sh","is_active":true}`, 0},
		{"fara is_active", `{"name":"backup","store_path":"/srv/b.sh","store_comp":"/opt/b.sh"}`, 0},
		{"lipseste name", `{"store_path":"/srv/b.sh","store_comp":"/opt/b.sh"}`, http.StatusUnsupportedMediaType},
		{"lipseste store_comp", `{"name":"b","store_path":"/srv/b.sh"}`, http.StatusUnsupportedMediaType},
		{"nume gol", `{"name":"  ","store_path":"/srv/b.sh","store_comp":"/opt/b.sh"}`, http.StatusUnprocessableEntity},
		{"cale relativa", `{"name":"b","store_path":"srv/b.sh","store_comp":"/opt/b.sh"}`, http.StatusUnprocessableEntity},
		{"cale cu ..", `{"name":"b","store_path":"/srv/../etc/passwd","store_comp":"/opt/b.sh"}`, http.StatusUnprocessableEntity},
		{"cale cu / la final", `{"name":"b","store_path":"/srv/b.sh","store_comp":"/opt/"}`, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, status := decodeScript(tt.body); status != tt.wantStatus {
				t.Errorf("status = %d, vrem %d", status, tt.wantStatus)
			}
		})
	}
}

func TestScriptRequestToModel(t *testing.T) {
	req, status := decodeScript(`{"name":"  backup ","store_path":"/srv/b.sh","store_comp":"/opt/b.sh"}`)
	if status != 0 {
		t.Fatalf("cererea a fost respinsa cu %d", status)
	}
	script := req.ToModel()
	// Numele e salvat fara spatii la capete; is_active lipsa inseamna false (SP, Tabelul 5).
	if script.Name != "backup" || script.IsActive || !script.ID.IsZero() {
		t.Errorf("ToModel = %+v", script)
	}
}
