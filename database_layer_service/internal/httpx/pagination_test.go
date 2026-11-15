package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePage(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantPage   Page
		wantFields []string
	}{
		{"fara parametri", "", http.StatusOK, Page{Number: 1, Size: 10}, nil},
		{"valori date", "?page=3&items_per_page=25", http.StatusOK, Page{Number: 3, Size: 25}, nil},
		{"maximul permis", "?items_per_page=100", http.StatusOK, Page{Number: 1, Size: 100}, nil},
		{"pagina 0", "?page=0", http.StatusBadRequest, Page{}, []string{"page"}},
		{"pagina negativa", "?page=-2", http.StatusBadRequest, Page{}, []string{"page"}},
		{"pagina text", "?page=abc", http.StatusBadRequest, Page{}, []string{"page"}},
		{"parametru gol", "?page=", http.StatusBadRequest, Page{}, []string{"page"}},
		{"pagina uriasa", "?page=99999999999", http.StatusBadRequest, Page{}, []string{"page"}},
		{"peste maxim", "?items_per_page=101", http.StatusBadRequest, Page{}, []string{"items_per_page"}},
		{"ambele invalide", "?page=0&items_per_page=0", http.StatusBadRequest, Page{}, []string{"page", "items_per_page"}},
	}

	engine := newTestEngine()
	engine.GET("/pagina", func(c *gin.Context) {
		page, ok := ParsePage(c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, page)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pagina"+tt.query, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, vrem %d (corp: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if rec.Code == http.StatusOK {
				var page Page
				if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || page != tt.wantPage {
					t.Errorf("pagina = %+v, vrem %+v", page, tt.wantPage)
				}
				return
			}
			var body struct {
				Errors []FieldError `json:"errors"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("corpul nu e JSON valid: %v", err)
			}
			var got []string
			for _, f := range body.Errors {
				got = append(got, f.Field)
			}
			if strings.Join(got, ",") != strings.Join(tt.wantFields, ",") {
				t.Errorf("campuri gresite = %v, vrem %v", got, tt.wantFields)
			}
		})
	}
}

func TestNewCollectionLinks(t *testing.T) {
	const base = "/api/manager/scripts?items_per_page=10&page="

	tests := []struct {
		name       string
		requestURL string
		page       int
		totalItems int
		want       map[string]string // relatie -> href; "" = relatia nu trebuie sa existe
	}{
		{"prima pagina", "/api/manager/scripts", 1, 25, map[string]string{
			"self": base + "1", "first": base + "1", "last": base + "3", "prev": "", "next": base + "2"}},
		{"pagina din mijloc", "/api/manager/scripts", 2, 25, map[string]string{
			"prev": base + "1", "next": base + "3"}},
		{"ultima pagina", "/api/manager/scripts", 3, 25, map[string]string{
			"prev": base + "2", "next": "", "last": base + "3"}},
		{"colectie goala", "/api/manager/scripts", 1, 0, map[string]string{
			"last": base + "1", "prev": "", "next": ""}},
		{"dupa ultima pagina", "/api/manager/scripts", 7, 25, map[string]string{
			"self": base + "7", "prev": base + "3", "next": ""}},
		{"pastreaza filtrele", "/api/manager/computers/1/scripts?isActive=true", 1, 5, map[string]string{
			"self": "/api/manager/computers/1/scripts?isActive=true&items_per_page=10&page=1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestURL, err := url.Parse(tt.requestURL)
			if err != nil {
				t.Fatal(err)
			}
			collection := NewCollection[string](nil, Page{Number: tt.page, Size: 10}, tt.totalItems, requestURL)

			for rel, wantHref := range tt.want {
				link, exists := collection.Links[rel]
				switch {
				case wantHref == "" && exists:
					t.Errorf("%s nu trebuia sa existe, dar e %q", rel, link.Href)
				case wantHref != "" && link.Href != wantHref:
					t.Errorf("%s = %q, vrem %q", rel, link.Href, wantHref)
				}
			}
		})
	}
}

func TestNewCollectionEmptyItemsIsArray(t *testing.T) {
	requestURL, _ := url.Parse("/api/manager/scripts")
	body, err := json.Marshal(NewCollection[string](nil, Page{Number: 1, Size: 10}, 0, requestURL))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"items":[]`) {
		t.Errorf("vrem \"items\":[] in JSON, am primit %s", body)
	}
}
