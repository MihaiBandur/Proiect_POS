package handler

import (
	"net/http"
	"testing"
)

// Id-uri fixe: le cream prin PUT (201), ca pasii sa le poata folosi in adrese.
const (
	computer1 = computersPath + "/6730f1a2c3d4e5f6a7b8c901"
	computer2 = computersPath + "/6730f1a2c3d4e5f6a7b8c902"
	cluster1  = clustersPath + "/6730f1a2c3d4e5f6a7b8c9c1"
	cluster2  = clustersPath + "/6730f1a2c3d4e5f6a7b8c9c2"
)

func TestComputers(t *testing.T) {
	router := newTestRouter()
	runSteps(t, router, []step{
		{"POST calculator", http.MethodPost, computersPath, adminToken, `{"IP":"10.0.0.1","hostname":"Node-1"}`, http.StatusCreated, ""},
		{"PUT calculator nou", http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.2"}`, http.StatusCreated, ""},
		{"al doilea calculator fara hostname", http.MethodPut, computer2, adminToken, `{"IP":"10.0.0.3"}`, http.StatusCreated, ""},
		{"IP duplicat", http.MethodPost, computersPath, adminToken, `{"IP":"10.0.0.1"}`, http.StatusConflict, "IP"},
		{"aceeasi adresa scrisa altfel", http.MethodPost, computersPath, adminToken, `{"IP":"::ffff:10.0.0.1"}`, http.StatusConflict, "IP"},
		{"hostname duplicat cu alte majuscule", http.MethodPost, computersPath, adminToken, `{"IP":"10.0.0.9","hostname":"NODE-1"}`, http.StatusConflict, "hostname"},
		{"PUT cu IP-ul altui calculator", http.MethodPut, computer2, adminToken, `{"IP":"10.0.0.2"}`, http.StatusConflict, "IP"},
		{"PUT care adauga hostname", http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.2","hostname":"node-2"}`, http.StatusNoContent, ""},
		{"lipseste IP", http.MethodPost, computersPath, adminToken, `{"hostname":"node-5"}`, http.StatusUnsupportedMediaType, "IP"},
		{"IP invalid", http.MethodPost, computersPath, adminToken, `{"IP":"10.0.0"}`, http.StatusUnprocessableEntity, "IP"},
		{"maintainerul fara asignari vede o lista goala", http.MethodGet, computersPath, maintainerToken, "", http.StatusOK, ""},
		{"maintainerul nu poate citi un calculator neasignat", http.MethodGet, computer1, maintainerToken, "", http.StatusForbidden, ""},
		{"DELETE", http.MethodDelete, computer2, adminToken, "", http.StatusOK, ""},
		{"GET dupa DELETE", http.MethodGet, computer2, adminToken, "", http.StatusNotFound, ""},
	})

	// Inlocuire completa: PUT fara hostname il sterge, iar JSON-ul nu mai contine campul.
	expectStatus(t, send(router, http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.2"}`), http.StatusNoContent)
	got := decode[map[string]any](t, send(router, http.MethodGet, computer1, adminToken, ""))
	if _, has := got["hostname"]; has || got["IP"] != "10.0.0.2" {
		t.Errorf("dupa PUT fara hostname: %v", got)
	}
}

func TestClusters(t *testing.T) {
	router := newTestRouter()
	runSteps(t, router, []step{
		{"PUT cluster nou", http.MethodPut, cluster1, adminToken, `{"nume":"laborator"}`, http.StatusCreated, ""},
		{"POST cluster", http.MethodPost, clustersPath, adminToken, `{"nume":"productie"}`, http.StatusCreated, ""},
		{"nume duplicat", http.MethodPost, clustersPath, adminToken, `{"nume":" laborator "}`, http.StatusConflict, "nume"},
		{"PUT cu numele altui cluster", http.MethodPut, cluster2, adminToken, `{"nume":"productie"}`, http.StatusConflict, "nume"},
		{"PUT pe acelasi cluster", http.MethodPut, cluster1, adminToken, `{"nume":"laborator"}`, http.StatusNoContent, ""},
		{"lipseste nume", http.MethodPost, clustersPath, adminToken, `{}`, http.StatusUnsupportedMediaType, "nume"},
		{"maintainerul nu poate crea", http.MethodPost, clustersPath, maintainerToken, `{"nume":"x"}`, http.StatusForbidden, ""},
		{"DELETE", http.MethodDelete, cluster1, adminToken, "", http.StatusOK, ""},
		{"numele e liber dupa DELETE", http.MethodPost, clustersPath, adminToken, `{"nume":"laborator"}`, http.StatusCreated, ""},
	})
}
