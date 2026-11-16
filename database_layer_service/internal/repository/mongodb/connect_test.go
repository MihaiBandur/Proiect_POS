package mongodb_test

import (
	"context"
	"strings"
	"testing"

	"job-management-service/internal/config"
	"job-management-service/internal/repository/mongodb"
	"job-management-service/internal/repository/mongodb/mongotest"
)

func TestConnectMissingSettings(t *testing.T) {
	_, err := mongodb.Connect(context.Background(), config.MongoConfig{Addr: "127.0.0.1:27017", Database: "x"})
	if err == nil || !strings.Contains(err.Error(), "MONGO_APP_USERNAME, MONGO_APP_PASSWORD") {
		t.Errorf("err = %v, vrem lista variabilelor lipsa", err)
	}
}

func TestConnect(t *testing.T) {
	if !mongotest.Enabled() {
		t.Skip("ruleaza cu MONGO_TESTS=1 si containerul de test pornit")
	}
	cfg, err := mongotest.Config()
	if err != nil {
		t.Fatal(err)
	}

	db, err := mongodb.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer db.Client().Disconnect(context.Background())
	if db.Name() != cfg.Database {
		t.Errorf("baza = %q, vrem %q", db.Name(), cfg.Database)
	}

	// O parola gresita trebuie sa fie refuzata chiar la pornire, nu la prima cerere.
	cfg.Password += "-gresita"
	if _, err := mongodb.Connect(context.Background(), cfg); err == nil {
		t.Error("Connect a reusit cu o parola gresita")
	}
}
