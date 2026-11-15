package handler

import (
	"net/http"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
)

// Id-uri fixe pentru scenarii (calculatoarele si clusterele sunt in computers_test.go).
const (
	script1   = scriptsPath + "/6730f1a2c3d4e5f6a7b8c9a1"
	script2   = scriptsPath + "/6730f1a2c3d4e5f6a7b8c9a2"
	computer3 = computersPath + "/6730f1a2c3d4e5f6a7b8c903"
	missingID = "/6730f1a2c3d4e5f6a7b8c999" // nu e creat niciodata

	activeScript   = `{"name":"activ","store_path":"/srv/a.sh","store_comp":"/opt/a.sh","is_active":true}`
	inactiveScript = `{"name":"inactiv","store_path":"/srv/b.sh","store_comp":"/opt/b.sh"}`
)

// idOf intoarce ultima parte a unei adrese, adica id-ul resursei.
func idOf(path string) string {
	return path[len(path)-24:]
}

// listBody este o pagina dintr-o colectie, cu elementele citite ca map-uri.
type listBody struct {
	Items      []map[string]any          `json:"items"`
	TotalItems int                       `json:"total_items"`
	Links      map[string]map[string]any `json:"_links"`
}

func getList(t *testing.T, router *gin.Engine, path, token string) listBody {
	t.Helper()
	rec := send(router, http.MethodGet, path, token, "")
	expectStatus(t, rec, http.StatusOK)
	return decode[listBody](t, rec)
}

// getIDs intoarce o lista de id-uri din reprezentarea unei resurse (ex. id_script).
func getIDs(t *testing.T, router *gin.Engine, path, field string) []string {
	t.Helper()
	rec := send(router, http.MethodGet, path, adminToken, "")
	expectStatus(t, rec, http.StatusOK)
	var ids []string
	for _, id := range decode[map[string]any](t, rec)[field].([]any) {
		ids = append(ids, id.(string))
	}
	return ids
}

// createUser adauga un cont din A&A in /users si intoarce adresa documentului.
func createUser(t *testing.T, router *gin.Engine, uid string) string {
	t.Helper()
	rec := send(router, http.MethodPost, usersPath, adminToken, `{"id_user_grpc":"`+uid+`"}`)
	expectStatus(t, rec, http.StatusCreated)
	return rec.Header().Get("Location")
}

func TestComputerScripts(t *testing.T) {
	router := newTestRouter()
	scripts := computer1 + "/scripts"
	runSteps(t, router, []step{
		{"script activ", http.MethodPut, script1, adminToken, activeScript, http.StatusCreated, ""},
		{"script inactiv", http.MethodPut, script2, adminToken, inactiveScript, http.StatusCreated, ""},
		{"calculator", http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.1"}`, http.StatusCreated, ""},
		{"instalare", http.MethodPut, scripts + "/" + idOf(script1), adminToken, "", http.StatusCreated, ""},
		{"instalare repetata (idempotenta)", http.MethodPut, scripts + "/" + idOf(script1), adminToken, "", http.StatusNoContent, ""},
		{"al doilea script", http.MethodPut, scripts + "/" + idOf(script2), adminToken, "", http.StatusCreated, ""},
		{"script inexistent", http.MethodPut, scripts + missingID, adminToken, "", http.StatusNotFound, ""},
		{"calculator inexistent", http.MethodPut, computersPath + missingID + "/scripts/" + idOf(script1), adminToken, "", http.StatusNotFound, ""},
		{"maintainerul nu instaleaza", http.MethodPut, scripts + "/" + idOf(script1), maintainerToken, "", http.StatusForbidden, ""},
		{"un script instalat", http.MethodGet, scripts + "/" + idOf(script1), adminToken, "", http.StatusOK, ""},
		{"filtru invalid", http.MethodGet, scripts + "?isActive=poate", adminToken, "", http.StatusBadRequest, ""},
		{"revocare", http.MethodDelete, scripts + "/" + idOf(script2), adminToken, "", http.StatusNoContent, ""},
		{"revocare repetata", http.MethodDelete, scripts + "/" + idOf(script2), adminToken, "", http.StatusNotFound, ""},
		{"scriptul revocat nu mai e pe calculator", http.MethodGet, scripts + "/" + idOf(script2), adminToken, "", http.StatusNotFound, ""},
		{"reinstalare", http.MethodPut, scripts + "/" + idOf(script2), adminToken, "", http.StatusCreated, ""},
	})

	// Fara verbose: doar id si link-uri. Cu verbose: si numele si caile (L3 2.1).
	list := getList(t, router, scripts, adminToken)
	if list.TotalItems != 2 || list.Items[0]["name"] != nil || list.Items[0]["_links"] == nil {
		t.Errorf("lista scurta: %+v", list)
	}
	if list := getList(t, router, scripts+"?verbose", adminToken); list.Items[0]["name"] != "activ" {
		t.Errorf("lista cu verbose: %+v", list.Items)
	}
	// Filtrarea dupa is_active (L3 2.1: ?isActive=...).
	if list := getList(t, router, scripts+"?isActive=true&verbose", adminToken); list.TotalItems != 1 || list.Items[0]["name"] != "activ" {
		t.Errorf("isActive=true: %+v", list.Items)
	}
	if list := getList(t, router, scripts+"?isActive=false&verbose", adminToken); list.TotalItems != 1 || list.Items[0]["name"] != "inactiv" {
		t.Errorf("isActive=false: %+v", list.Items)
	}
	// Link-urile colectiei: parent e calculatorul; adminul primeste sablonul pentru instalare.
	if list.Links["parent"]["href"] != computer1 || list.Links["add"]["templated"] != true {
		t.Errorf("link-urile sub-colectiei: %+v", list.Links)
	}

	// PUT pe calculator schimba IP-ul, dar nu atinge scripturile instalate.
	expectStatus(t, send(router, http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.5"}`), http.StatusNoContent)
	if ids := getIDs(t, router, computer1, "id_script"); len(ids) != 2 {
		t.Errorf("dupa PUT pe calculator, id_script = %v", ids)
	}

	// Stergerea scriptului il scoate si de pe calculator (cascada).
	expectStatus(t, send(router, http.MethodDelete, script1, adminToken, ""), http.StatusOK)
	if ids := getIDs(t, router, computer1, "id_script"); slices.Contains(ids, idOf(script1)) || len(ids) != 1 {
		t.Errorf("dupa stergerea scriptului, id_script = %v", ids)
	}
}

func TestClusterComputers(t *testing.T) {
	router := newTestRouter()
	runSteps(t, router, []step{
		{"cluster 1", http.MethodPut, cluster1, adminToken, `{"nume":"laborator"}`, http.StatusCreated, ""},
		{"cluster 2", http.MethodPut, cluster2, adminToken, `{"nume":"productie"}`, http.StatusCreated, ""},
		{"calculator 1", http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.1"}`, http.StatusCreated, ""},
		{"calculator 2", http.MethodPut, computer2, adminToken, `{"IP":"10.0.0.2"}`, http.StatusCreated, ""},
		{"adaugare in cluster", http.MethodPut, cluster1 + "/computers/" + idOf(computer1), adminToken, "", http.StatusCreated, ""},
		{"adaugare repetata", http.MethodPut, cluster1 + "/computers/" + idOf(computer1), adminToken, "", http.StatusNoContent, ""},
		{"calculatorul e deja in alt cluster", http.MethodPut, cluster2 + "/computers/" + idOf(computer1), adminToken, "", http.StatusConflict, ""},
		{"al doilea calculator", http.MethodPut, cluster1 + "/computers/" + idOf(computer2), adminToken, "", http.StatusCreated, ""},
		{"eliminare din cluster", http.MethodDelete, cluster1 + "/computers/" + idOf(computer2), adminToken, "", http.StatusNoContent, ""},
		{"eliminare repetata", http.MethodDelete, cluster1 + "/computers/" + idOf(computer2), adminToken, "", http.StatusNotFound, ""},
		{"calculatorul liber poate intra in alt cluster", http.MethodPut, cluster2 + "/computers/" + idOf(computer2), adminToken, "", http.StatusCreated, ""},
		{"un calculator din cluster", http.MethodGet, cluster1 + "/computers/" + idOf(computer1), adminToken, "", http.StatusOK, ""},
		{"calculator care nu e in cluster", http.MethodGet, cluster1 + "/computers/" + idOf(computer2), adminToken, "", http.StatusNotFound, ""},
	})

	if list := getList(t, router, cluster1+"/computers", adminToken); list.TotalItems != 1 {
		t.Errorf("calculatoarele din cluster1: %+v", list)
	}

	// Stergerea calculatorului il scoate si din cluster (cascada).
	expectStatus(t, send(router, http.MethodDelete, computer1, adminToken, ""), http.StatusOK)
	if ids := getIDs(t, router, cluster1, "id_comp"); len(ids) != 0 {
		t.Errorf("dupa stergerea calculatorului, id_comp = %v", ids)
	}
}

func TestAssignments(t *testing.T) {
	router := newTestRouter()
	expectStatus(t, send(router, http.MethodPut, computer1, adminToken, `{"IP":"10.0.0.1"}`), http.StatusCreated)
	expectStatus(t, send(router, http.MethodPut, cluster1, adminToken, `{"nume":"laborator"}`), http.StatusCreated)
	ana := createUser(t, router, anaUID)   // maintainer
	boss := createUser(t, router, bossUID) // administrator

	runSteps(t, router, []step{
		{"asignare pe calculator", http.MethodPut, computer1 + "/users/" + idOf(ana), adminToken, "", http.StatusCreated, ""},
		{"asignare repetata", http.MethodPut, computer1 + "/users/" + idOf(ana), adminToken, "", http.StatusNoContent, ""},
		{"un admin nu poate fi asignat ca maintainer", http.MethodPut, computer1 + "/users/" + idOf(boss), adminToken, "", http.StatusConflict, ""},
		{"utilizator inexistent", http.MethodPut, computer1 + "/users" + missingID, adminToken, "", http.StatusNotFound, ""},
		{"asignare pe cluster", http.MethodPut, cluster1 + "/users/" + idOf(ana), adminToken, "", http.StatusCreated, ""},
		{"maintainerul nu vede asignarile", http.MethodGet, computer1 + "/users", maintainerToken, "", http.StatusForbidden, ""},
		{"un maintainer asignat", http.MethodGet, computer1 + "/users/" + idOf(ana), adminToken, "", http.StatusOK, ""},
	})

	// Asignarea se vede din ambele parti: id_users la calculator si id_comp_clust la utilizator.
	if ids := getIDs(t, router, computer1, "id_users"); !slices.Equal(ids, []string{idOf(ana)}) {
		t.Errorf("id_users al calculatorului = %v", ids)
	}
	user := decode[map[string]any](t, send(router, http.MethodGet, ana, adminToken, ""))
	if refs := user["id_comp_clust"].([]any); len(refs) != 2 {
		t.Errorf("id_comp_clust = %v, vrem calculatorul si clusterul", refs)
	}

	// Revocarea scoate legatura din ambele parti.
	expectStatus(t, send(router, http.MethodDelete, computer1+"/users/"+idOf(ana), adminToken, ""), http.StatusNoContent)
	if ids := getIDs(t, router, computer1, "id_users"); len(ids) != 0 {
		t.Errorf("dupa revocare, id_users = %v", ids)
	}
	user = decode[map[string]any](t, send(router, http.MethodGet, ana, adminToken, ""))
	if refs := user["id_comp_clust"].([]any); len(refs) != 1 || refs[0].(map[string]any)["tip"] != "cluster" {
		t.Errorf("dupa revocare, id_comp_clust = %v", refs)
	}

	// Stergerea utilizatorului il scoate si din cluster (cascada).
	expectStatus(t, send(router, http.MethodDelete, ana, adminToken, ""), http.StatusOK)
	if ids := getIDs(t, router, cluster1, "id_users"); len(ids) != 0 {
		t.Errorf("dupa stergerea utilizatorului, id_users al clusterului = %v", ids)
	}
}

func TestMaintainerAccess(t *testing.T) {
	router := newTestRouter()
	// Adminul pregateste: ana (maintainerToken) primeste computer1 direct si cluster1,
	// in care e computer2. computer3 si cluster2 nu sunt ale ei.
	for _, path := range []string{computer1, computer2, computer3} {
		expectStatus(t, send(router, http.MethodPut, path, adminToken, `{"IP":"10.0.0.`+path[len(path)-1:]+`"}`), http.StatusCreated)
	}
	expectStatus(t, send(router, http.MethodPut, cluster1, adminToken, `{"nume":"laborator"}`), http.StatusCreated)
	expectStatus(t, send(router, http.MethodPut, cluster2, adminToken, `{"nume":"productie"}`), http.StatusCreated)
	expectStatus(t, send(router, http.MethodPut, script1, adminToken, activeScript), http.StatusCreated)
	ana := createUser(t, router, anaUID)
	for _, path := range []string{
		cluster1 + "/computers/" + idOf(computer2),
		computer1 + "/scripts/" + idOf(script1),
		computer1 + "/users/" + idOf(ana),
		cluster1 + "/users/" + idOf(ana),
	} {
		expectStatus(t, send(router, http.MethodPut, path, adminToken, ""), http.StatusCreated)
	}

	// Lista contine computer1 (asignat direct) si computer2 (din cluster1), dar nu computer3.
	list := getList(t, router, computersPath, maintainerToken)
	if list.TotalItems != 2 || list.Links["create"] != nil {
		t.Errorf("calculatoarele maintainerului: %+v", list)
	}
	// Maintainerul nu primeste actiuni de admin, doar navigarea catre scripturi.
	links := list.Items[0]["_links"].(map[string]any)
	if links["edit"] != nil || links["users"] != nil || links["scripts"] == nil {
		t.Errorf("link-urile vazute de maintainer: %v", links)
	}

	runSteps(t, router, []step{
		{"calculator asignat direct", http.MethodGet, computer1, maintainerToken, "", http.StatusOK, ""},
		{"calculator din clusterul asignat", http.MethodGet, computer2, maintainerToken, "", http.StatusOK, ""},
		{"calculator neasignat", http.MethodGet, computer3, maintainerToken, "", http.StatusForbidden, ""},
		{"calculator inexistent", http.MethodGet, computersPath + missingID, maintainerToken, "", http.StatusNotFound, ""},
		{"scripturile calculatorului asignat", http.MethodGet, computer1 + "/scripts", maintainerToken, "", http.StatusOK, ""},
		{"scripturile calculatorului neasignat", http.MethodGet, computer3 + "/scripts", maintainerToken, "", http.StatusForbidden, ""},
		{"clusterul asignat", http.MethodGet, cluster1, maintainerToken, "", http.StatusOK, ""},
		{"clusterul neasignat", http.MethodGet, cluster2, maintainerToken, "", http.StatusForbidden, ""},
		{"calculatoarele clusterului asignat", http.MethodGet, cluster1 + "/computers", maintainerToken, "", http.StatusOK, ""},
		{"maintainerul nu modifica legaturi", http.MethodDelete, computer1 + "/scripts/" + idOf(script1), maintainerToken, "", http.StatusForbidden, ""},
	})

	if list := getList(t, router, clustersPath, maintainerToken); list.TotalItems != 1 {
		t.Errorf("clusterele maintainerului: %+v", list)
	}
}
