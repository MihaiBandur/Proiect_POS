package handler

import (
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// scriptBody este reprezentarea primita de client, citita din JSON.
type scriptBody struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
	Links    map[string]struct {
		Href string `json:"href"`
		Type string `json:"type"`
	} `json:"_links"`
}

const validScript = `{"name":"backup","store_path":"/srv/scripts/backup.sh","store_comp":"/opt/backup.sh"}`

func TestScriptsCRUD(t *testing.T) {
	router := newTestRouter()

	// POST: 201, Location, is_active implicit false, actiuni pentru admin.
	rec := send(router, http.MethodPost, scriptsPath, adminToken, validScript)
	expectStatus(t, rec, http.StatusCreated)
	created := decode[scriptBody](t, rec)
	self := scriptsPath + "/" + created.ID
	if got := rec.Header().Get("Location"); got != self {
		t.Errorf("Location = %q, vrem %q", got, self)
	}
	if created.IsActive || created.Links["self"].Href != self || created.Links["parent"].Href != scriptsPath {
		t.Errorf("reprezentare gresita: %+v", created)
	}
	if created.Links["edit"].Type != http.MethodPut || created.Links["delete"].Type != http.MethodDelete {
		t.Errorf("adminul trebuie sa vada actiunile edit si delete: %+v", created.Links)
	}

	// GET ca maintainer: vede scriptul, dar fara actiuni.
	rec = send(router, http.MethodGet, self, maintainerToken, "")
	expectStatus(t, rec, http.StatusOK)
	if links := decode[scriptBody](t, rec).Links; len(links) != 2 {
		t.Errorf("maintainerul trebuie sa vada doar self si parent: %+v", links)
	}

	// Lista ca maintainer: un element, fara link-ul create.
	rec = send(router, http.MethodGet, scriptsPath, maintainerToken, "")
	expectStatus(t, rec, http.StatusOK)
	list := decode[struct {
		Items      []scriptBody              `json:"items"`
		TotalItems int                       `json:"total_items"`
		Links      map[string]map[string]any `json:"_links"`
	}](t, rec)
	if list.TotalItems != 1 || len(list.Items) != 1 || list.Links["create"] != nil || list.Links["parent"] == nil {
		t.Errorf("lista gresita: %+v", list)
	}

	// PUT pe un script existent: 204; inlocuirea e completa (is_active lipsa -> false).
	rec = send(router, http.MethodPut, self, adminToken,
		`{"name":"backup2","store_path":"/srv/b.sh","store_comp":"/opt/b.sh","is_active":true}`)
	expectStatus(t, rec, http.StatusNoContent)
	rec = send(router, http.MethodPut, self, adminToken, validScript)
	expectStatus(t, rec, http.StatusNoContent)
	if got := decode[scriptBody](t, send(router, http.MethodGet, self, adminToken, "")); got.Name != "backup" || got.IsActive {
		t.Errorf("dupa PUT: %+v, vrem name=backup, is_active=false", got)
	}

	// PUT pe un id nou: 201.
	newSelf := scriptsPath + "/" + bson.NewObjectID().Hex()
	expectStatus(t, send(router, http.MethodPut, newSelf, adminToken, validScript), http.StatusCreated)

	// DELETE: 200 cu ultima reprezentare si doar link-ul parent; apoi 404.
	rec = send(router, http.MethodDelete, self, adminToken, "")
	expectStatus(t, rec, http.StatusOK)
	deleted := decode[scriptBody](t, rec)
	if deleted.Name != "backup" || len(deleted.Links) != 1 || deleted.Links["parent"].Href != scriptsPath {
		t.Errorf("reprezentarea dupa DELETE: %+v", deleted)
	}
	expectStatus(t, send(router, http.MethodGet, self, adminToken, ""), http.StatusNotFound)
	expectStatus(t, send(router, http.MethodDelete, self, adminToken, ""), http.StatusNotFound)
}

func TestScriptsErrors(t *testing.T) {
	router := newTestRouter()
	someID := scriptsPath + "/" + bson.NewObjectID().Hex()

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		body       string
		wantStatus int
	}{
		{"maintainerul nu poate crea", http.MethodPost, scriptsPath, maintainerToken, validScript, http.StatusForbidden},
		{"maintainerul nu poate inlocui", http.MethodPut, someID, maintainerToken, validScript, http.StatusForbidden},
		{"maintainerul nu poate sterge", http.MethodDelete, someID, maintainerToken, "", http.StatusForbidden},
		{"camp obligatoriu lipsa", http.MethodPost, scriptsPath, adminToken, `{"name":"x","store_path":"/a"}`, http.StatusUnsupportedMediaType},
		{"cale relativa", http.MethodPost, scriptsPath, adminToken, `{"name":"x","store_path":"a.sh","store_comp":"/a"}`, http.StatusUnprocessableEntity},
		{"cale cu ..", http.MethodPost, scriptsPath, adminToken, `{"name":"x","store_path":"/srv/../etc/passwd","store_comp":"/a"}`, http.StatusUnprocessableEntity},
		{"nume gol", http.MethodPost, scriptsPath, adminToken, `{"name":" ","store_path":"/a","store_comp":"/a"}`, http.StatusUnprocessableEntity},
		{"id malformat", http.MethodGet, scriptsPath + "/123", adminToken, "", http.StatusBadRequest},
		{"id plin de zero", http.MethodGet, scriptsPath + "/000000000000000000000000", adminToken, "", http.StatusBadRequest},
		{"script inexistent", http.MethodGet, someID, adminToken, "", http.StatusNotFound},
		{"paginare invalida", http.MethodGet, scriptsPath + "?page=0", adminToken, "", http.StatusBadRequest},
		{"metoda nepermisa", http.MethodPatch, scriptsPath, adminToken, "", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := send(router, tt.method, tt.path, tt.token, tt.body)
			expectStatus(t, rec, tt.wantStatus)
			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, vrem application/problem+json", got)
			}
		})
	}
}
