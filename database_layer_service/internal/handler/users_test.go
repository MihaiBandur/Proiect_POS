package handler

import (
	"net/http"
	"strings"
	"testing"
)

const user1 = usersPath + "/6730f1a2c3d4e5f6a7b8c9e1"

func TestUsers(t *testing.T) {
	router := newTestRouter()

	// POST: uid-ul e cautat in A&A Services, iar userName e copiat de acolo.
	rec := send(router, http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"`+strings.ToUpper(anaUID)+`"}`)
	expectStatus(t, rec, http.StatusCreated)
	created := decode[struct {
		ID         string `json:"id"`
		IDUserGRPC string `json:"id_user_grpc"`
		UserName   string `json:"userName"`
	}](t, rec)
	if created.IDUserGRPC != anaUID || created.UserName != "ana" {
		t.Errorf("utilizator creat: %+v", created)
	}
	if got := rec.Header().Get("Location"); got != usersPath+"/"+created.ID {
		t.Errorf("Location = %q", got)
	}

	runSteps(t, router, []step{
		{"acelasi cont a doua oara", http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"` + anaUID + `"}`, http.StatusConflict, "id_user_grpc"},
		{"uid inexistent in A&A", http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"0190f2a8-ffff-7fff-8fff-ffffffffffff"}`, http.StatusUnprocessableEntity, "id_user_grpc"},
		{"A&A oprit", http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"` + unavailableUID + `"}`, http.StatusServiceUnavailable, ""},
		{"uid care nu e UUID", http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"ana"}`, http.StatusUnprocessableEntity, "id_user_grpc"},
		{"userName trimis de client", http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"` + anaUID + `","userName":"x"}`, http.StatusUnsupportedMediaType, "userName"},
		{"maintainerul nu vede utilizatorii", http.MethodGet, usersPath, maintainerToken, "", http.StatusForbidden, ""},
		{"PUT pe acelasi document", http.MethodPut, usersPath + "/" + created.ID, adminToken, `{"id_user_grpc":"` + anaUID + `"}`, http.StatusNoContent, ""},
		{"PUT cu un cont deja folosit", http.MethodPut, user1, adminToken, `{"id_user_grpc":"` + anaUID + `"}`, http.StatusConflict, "id_user_grpc"},
		{"DELETE", http.MethodDelete, usersPath + "/" + created.ID, adminToken, "", http.StatusOK, ""},
		{"contul poate fi adaugat din nou", http.MethodPut, user1, adminToken, `{"id_user_grpc":"` + anaUID + `"}`, http.StatusCreated, ""},
	})
}
