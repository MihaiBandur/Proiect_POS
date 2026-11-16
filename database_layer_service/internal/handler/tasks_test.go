package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// prepareRun pregateste un scenariu de rulare: computer1 si computer3 sunt in cluster1 si au
// instalat script1 (activ); computer1 are si script2 (inactiv). ana (maintainerToken) e
// asignata pe computer1 si pe cluster1. computer2 nu e al ei.
func prepareRun(t *testing.T, router *gin.Engine) {
	t.Helper()
	for _, s := range []struct{ method, path, body string }{
		{http.MethodPut, script1, activeScript},
		{http.MethodPut, script2, inactiveScript},
		{http.MethodPut, computer1, `{"IP":"10.0.0.1","hostname":"node-1"}`},
		{http.MethodPut, computer2, `{"IP":"10.0.0.2"}`},
		{http.MethodPut, computer3, `{"IP":"10.0.0.3"}`},
		{http.MethodPut, cluster1, `{"nume":"laborator"}`},
		{http.MethodPut, computer1 + "/scripts/" + idOf(script1), ""},
		{http.MethodPut, computer1 + "/scripts/" + idOf(script2), ""},
		{http.MethodPut, computer2 + "/scripts/" + idOf(script1), ""},
		{http.MethodPut, computer3 + "/scripts/" + idOf(script1), ""},
		{http.MethodPut, cluster1 + "/computers/" + idOf(computer1), ""},
		{http.MethodPut, cluster1 + "/computers/" + idOf(computer3), ""},
	} {
		rec := send(router, s.method, s.path, adminToken, s.body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s %s: %d %s", s.method, s.path, rec.Code, rec.Body)
		}
	}
	ana := createUser(t, router, anaUID)
	expectStatus(t, send(router, http.MethodPut, computer1+"/users/"+idOf(ana), adminToken, ""), http.StatusCreated)
	expectStatus(t, send(router, http.MethodPut, cluster1+"/users/"+idOf(ana), adminToken, ""), http.StatusCreated)
}

// taskBody este reprezentarea unui task, citita din JSON.
type taskBody struct {
	ID     string `json:"id"`
	Target struct {
		ID  string `json:"id"`
		Tip string `json:"tip"`
	} `json:"target"`
	State  string `json:"state"`
	Status string `json:"status"`
	Links  map[string]struct {
		Href string `json:"href"`
	} `json:"_links"`
}

func TestLaunchTask(t *testing.T) {
	fake := &fakeExecutor{}
	router := newTestRouterWith(func(deps *Dependencies) { deps.Executor = fake })
	prepareRun(t, router)
	run := computer1 + "/scripts/" + idOf(script1) + "/tasks"

	// 202 Accepted + Location catre task (monitorul de stare, RFC 9110, 15.3.3).
	rec := send(router, http.MethodPost, run, maintainerToken, "")
	expectStatus(t, rec, http.StatusAccepted)
	task := decode[taskBody](t, rec)
	if rec.Header().Get("Location") != tasksPath+"/"+task.ID || task.State != "pending" {
		t.Errorf("task = %+v, Location %q", task, rec.Header().Get("Location"))
	}
	if task.Target.Tip != "calculator" || task.Target.ID != idOf(computer1) || task.Links["computer"].Href == "" {
		t.Errorf("tinta task-ului: %+v", task)
	}
	// Executorul a primit doar id-ul job-ului (contractul din student-kit).
	if len(fake.jobs) != 1 || fake.jobs[0] != task.ID {
		t.Errorf("job-uri anuntate: %v, vrem [%s]", fake.jobs, task.ID)
	}

	// Pe tot clusterul: target = cluster.
	rec = send(router, http.MethodPost, cluster1+"/scripts/"+idOf(script1)+"/tasks", maintainerToken, "")
	expectStatus(t, rec, http.StatusAccepted)
	if task := decode[taskBody](t, rec); task.Target.Tip != "cluster" || task.Links["cluster"].Href == "" {
		t.Errorf("task pe cluster: %+v", task)
	}

	runSteps(t, router, []step{
		{"adminul nu porneste task-uri", http.MethodPost, run, adminToken, "", http.StatusForbidden, ""},
		{"script inactiv", http.MethodPost, computer1 + "/scripts/" + idOf(script2) + "/tasks", maintainerToken, "", http.StatusConflict, ""},
		{"script neinstalat pe tot clusterul", http.MethodPost, cluster1 + "/scripts/" + idOf(script2) + "/tasks", maintainerToken, "", http.StatusConflict, ""},
		{"calculator neasignat", http.MethodPost, computer2 + "/scripts/" + idOf(script1) + "/tasks", maintainerToken, "", http.StatusForbidden, ""},
		{"script inexistent", http.MethodPost, computer1 + "/scripts" + missingID + "/tasks", maintainerToken, "", http.StatusNotFound, ""},
		{"task-ul maintainerului", http.MethodGet, tasksPath + "/" + task.ID, maintainerToken, "", http.StatusOK, ""},
		{"adminul vede orice task", http.MethodGet, tasksPath + "/" + task.ID, adminToken, "", http.StatusOK, ""},
	})

	// Script neinstalat: il scoatem de pe calculator, apoi incercam sa-l rulam.
	expectStatus(t, send(router, http.MethodDelete, computer1+"/scripts/"+idOf(script1), adminToken, ""), http.StatusNoContent)
	expectStatus(t, send(router, http.MethodPost, run, maintainerToken, ""), http.StatusConflict)

	// Lista: maintainerul isi vede cele doua task-uri.
	if list := getList(t, router, tasksPath, maintainerToken); list.TotalItems != 2 {
		t.Errorf("task-urile maintainerului: %+v", list)
	}
}

func TestLaunchTaskExecutorModes(t *testing.T) {
	run := computer1 + "/scripts/" + idOf(script1) + "/tasks"

	// Executor oprit: tot 202; task-ul ramane pending, iar executorul il gaseste la scanare.
	router := newTestRouterWith(func(deps *Dependencies) { deps.Executor = &fakeExecutor{down: true} })
	prepareRun(t, router)
	rec := send(router, http.MethodPost, run, maintainerToken, "")
	expectStatus(t, rec, http.StatusAccepted)
	if task := decode[taskBody](t, rec); task.State != "pending" {
		t.Errorf("task cu executorul oprit = %+v, vrem pending", task)
	}

	// Mod mock (fara executor sau Job Status Service): 202 cu un task simulat, deja terminat.
	router = newTestRouterWith(func(deps *Dependencies) { deps.Executor = &fakeExecutor{mock: true} })
	prepareRun(t, router)
	rec = send(router, http.MethodPost, run, maintainerToken, "")
	expectStatus(t, rec, http.StatusAccepted)
	if task := decode[taskBody](t, rec); task.State != "finished" || task.Status == "" {
		t.Errorf("task mock = %+v, vrem finished cu status", task)
	}
}

func TestRunLinkForMaintainer(t *testing.T) {
	router := newTestRouter()
	prepareRun(t, router)

	// Maintainerul vede actiunea "run" pe scripturile instalate si pe cluster; adminul nu.
	item := func(token string) map[string]any {
		list := getList(t, router, computer1+"/scripts", token)
		return list.Items[0]["_links"].(map[string]any)
	}
	if item(maintainerToken)["run"] == nil || item(adminToken)["run"] != nil {
		t.Errorf("link-ul run: maintainer %v, admin %v", item(maintainerToken), item(adminToken))
	}
	cluster := decode[map[string]any](t, send(router, http.MethodGet, cluster1, maintainerToken, ""))
	if cluster["_links"].(map[string]any)["run"] == nil {
		t.Errorf("clusterul nu are link-ul run pentru maintainer: %v", cluster["_links"])
	}
}
