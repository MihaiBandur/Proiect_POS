package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type testRequest struct {
	Name  *string `json:"name"`
	Count *int    `json:"count"`
}

func (r *testRequest) Validate(v *Validation) {
	if r.Name == nil {
		v.Missing("name")
	} else if strings.TrimSpace(*r.Name) == "" {
		v.Invalid("name", "nu poate fi gol")
	}
	if r.Count != nil && *r.Count < 0 {
		v.Invalid("count", "trebuie sa fie cel putin 0")
	}
}

func TestDecodeJSON(t *testing.T) {
	tooLarge := `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`

	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantField   string // primul camp din "errors"
		wantAccept  string
	}{
		{"cerere corecta", "application/json", `{"name":"a","count":1}`, http.StatusCreated, "", ""},
		{"content-type cu parametri si majuscule", "Application/JSON; charset=utf-8", `{"name":"a"}`, http.StatusCreated, "", ""},
		{"fara content-type", "", `{"name":"a"}`, http.StatusUnsupportedMediaType, "", "application/json"},
		{"alt content-type", "text/plain", `{"name":"a"}`, http.StatusUnsupportedMediaType, "", "application/json"},
		{"corp gol", "application/json", ``, http.StatusBadRequest, "", ""},
		{"virgula in plus", "application/json", `{"name":"a",}`, http.StatusBadRequest, "", ""},
		{"obiect neterminat", "application/json", `{"name":"a"`, http.StatusBadRequest, "", ""},
		{"doua obiecte", "application/json", `{"name":"a"} {"name":"b"}`, http.StatusBadRequest, "", ""},
		{"corp prea mare", "application/json", tooLarge, http.StatusRequestEntityTooLarge, "", ""},
		{"lista in loc de obiect", "application/json", `[1]`, http.StatusUnsupportedMediaType, "", ""},
		{"camp necunoscut", "application/json", `{"name":"a","nou":1}`, http.StatusUnsupportedMediaType, "nou", ""},
		{"tip gresit", "application/json", `{"name":5}`, http.StatusUnsupportedMediaType, "name", ""},
		{"camp obligatoriu lipsa", "application/json", `{"count":1}`, http.StatusUnsupportedMediaType, "name", ""},
		{"valoare nepermisa", "application/json", `{"name":"  "}`, http.StatusUnprocessableEntity, "name", ""},
		{"lipsa are prioritate fata de nepermis", "application/json", `{"count":-1}`, http.StatusUnsupportedMediaType, "name", ""},
	}

	engine := newTestEngine()
	engine.POST("/decode", func(c *gin.Context) {
		var req testRequest
		if !DecodeJSON(c, &req) {
			return
		}
		c.JSON(http.StatusCreated, gin.H{"name": *req.Name})
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, vrem %d (corp: %.200s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := rec.Header().Get("Accept"); got != tt.wantAccept {
				t.Errorf("Accept = %q, vrem %q", got, tt.wantAccept)
			}
			if rec.Code < 400 {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != problemContentType {
				t.Errorf("Content-Type = %q, vrem %s", got, problemContentType)
			}
			var body struct {
				Status int          `json:"status"`
				Errors []FieldError `json:"errors"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("corpul nu e JSON valid: %v", err)
			}
			if body.Status != tt.wantStatus {
				t.Errorf("status din corp = %d, vrem %d", body.Status, tt.wantStatus)
			}
			if tt.wantField != "" && (len(body.Errors) == 0 || body.Errors[0].Field != tt.wantField) {
				t.Errorf("errors = %+v, vrem primul camp %q", body.Errors, tt.wantField)
			}
		})
	}
}
