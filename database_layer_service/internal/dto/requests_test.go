package dto

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/httpx"
)

// decodeInto trimite corpul prin httpx.DecodeJSON in dst, ca in controller.
// Intoarce codul de eroare, sau 0 daca cererea a fost acceptata.
func decodeInto(dst httpx.Validator, body string) int {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if !httpx.DecodeJSON(c, dst) {
		return rec.Code
	}
	return 0
}

func TestRequestValidate(t *testing.T) {
	const (
		missing = http.StatusUnsupportedMediaType // 415: camp obligatoriu lipsa (L3)
		invalid = http.StatusUnprocessableEntity  // 422: valoare nepermisa
	)
	computer := func() httpx.Validator { return &ComputerRequest{} }
	cluster := func() httpx.Validator { return &ClusterRequest{} }
	user := func() httpx.Validator { return &UserRequest{} }

	tests := []struct {
		name       string
		newRequest func() httpx.Validator
		body       string
		wantStatus int // 0 = cerere acceptata
	}{
		{"calculator cu IPv4 si hostname", computer, `{"IP":"192.168.1.10","hostname":"node-01.cluster.local"}`, 0},
		{"calculator fara hostname", computer, `{"IP":"10.0.0.1"}`, 0},
		{"calculator cu IPv6", computer, `{"IP":"2001:db8::1"}`, 0},
		{"calculator fara IP", computer, `{"hostname":"node-01"}`, missing},
		{"IP invalid", computer, `{"IP":"300.1.1.1"}`, invalid},
		{"IP cu zerouri in fata", computer, `{"IP":"010.0.0.1"}`, invalid},
		{"hostname gol", computer, `{"IP":"10.0.0.1","hostname":""}`, invalid},
		{"hostname cu underscore", computer, `{"IP":"10.0.0.1","hostname":"node_01"}`, invalid},
		{"hostname cu - la inceput", computer, `{"IP":"10.0.0.1","hostname":"-node"}`, invalid},
		{"cluster corect", cluster, `{"nume":"laborator"}`, 0},
		{"cluster fara nume", cluster, `{}`, missing},
		{"cluster cu nume gol", cluster, `{"nume":"  "}`, invalid},
		{"utilizator corect", user, `{"id_user_grpc":"0190f2a8-7c1e-7a3b-9d4e-2f6a8b1c3d5e"}`, 0},
		{"utilizator fara uid", user, `{}`, missing},
		{"uid care nu e UUID", user, `{"id_user_grpc":"admin"}`, invalid},
		{"userName trimis de client", user, `{"id_user_grpc":"0190f2a8-7c1e-7a3b-9d4e-2f6a8b1c3d5e","userName":"x"}`, missing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status := decodeInto(tt.newRequest(), tt.body); status != tt.wantStatus {
				t.Errorf("status = %d, vrem %d", status, tt.wantStatus)
			}
		})
	}
}

func TestRequestToModel(t *testing.T) {
	var computer ComputerRequest
	if decodeInto(&computer, `{"IP":"2001:DB8:0:0::0001","hostname":"NODE-01"}`) != 0 {
		t.Fatal("calculatorul a fost respins")
	}
	// Forma standard, ca doua scrieri ale aceleiasi adrese sa fie detectate ca duplicat (409).
	if got := computer.ToModel(); got.IP != "2001:db8::1" || got.Hostname != "node-01" {
		t.Errorf("ComputerRequest.ToModel = %+v", got)
	}

	var cluster ClusterRequest
	if decodeInto(&cluster, `{"nume":"  laborator "}`) != 0 {
		t.Fatal("clusterul a fost respins")
	}
	if got := cluster.ToModel(); got.Nume != "laborator" {
		t.Errorf("ClusterRequest.ToModel = %+v", got)
	}

	var user UserRequest
	if decodeInto(&user, `{"id_user_grpc":"0190F2A8-7C1E-7A3B-9D4E-2F6A8B1C3D5E"}`) != 0 {
		t.Fatal("utilizatorul a fost respins")
	}
	if got := user.ToModel("ana"); got.IDUserGRPC != "0190f2a8-7c1e-7a3b-9d4e-2f6a8b1c3d5e" || got.UserName != "ana" {
		t.Errorf("UserRequest.ToModel = %+v", got)
	}
}
