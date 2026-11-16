package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCORSOnAPI verifica CORS pe API-ul complet, asa cum il foloseste aplicatia React din browser.
func TestCORSOnAPI(t *testing.T) {
	router := newTestRouter()

	// 1. Preflight: fara token, pe o ruta care cere autentificare -> 204, nu 401.
	req := httptest.NewRequest(http.MethodOptions, scriptsPath, nil)
	req.Header.Set("Origin", webOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	expectStatus(t, rec, http.StatusNoContent)
	if rec.Header().Get("Access-Control-Allow-Origin") != webOrigin {
		t.Errorf("preflight fara Access-Control-Allow-Origin: %v", rec.Header())
	}

	// 2. Cererea propriu-zisa: 201, iar JavaScript poate citi header-ul Location.
	req = httptest.NewRequest(http.MethodPost, scriptsPath, strings.NewReader(validScript))
	req.Header.Set("Origin", webOrigin)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	expectStatus(t, rec, http.StatusCreated)
	if rec.Header().Get("Access-Control-Allow-Origin") != webOrigin || rec.Header().Get("Access-Control-Expose-Headers") == "" {
		t.Errorf("raspunsul 201 nu are header-ele CORS: %v", rec.Header())
	}

	// 3. Si erorile au header-ele CORS: altfel aplicatia nu ar putea citi un 401 ca sa
	//    trimita utilizatorul la login.
	req = httptest.NewRequest(http.MethodGet, scriptsPath, nil)
	req.Header.Set("Origin", webOrigin)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	expectStatus(t, rec, http.StatusUnauthorized)
	if rec.Header().Get("Access-Control-Allow-Origin") != webOrigin {
		t.Errorf("raspunsul 401 nu are Access-Control-Allow-Origin: %v", rec.Header())
	}
}
