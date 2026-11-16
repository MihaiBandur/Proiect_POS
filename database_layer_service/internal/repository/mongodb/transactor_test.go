package mongodb_test

import (
	"context"
	"errors"
	"testing"

	"job-management-service/internal/model"
	"job-management-service/internal/repository/mongodb"
	"job-management-service/internal/repository/mongodb/mongotest"
)

func TestTransaction(t *testing.T) {
	db := mongotest.Require(t)
	ctx := context.Background()
	scripts := mongodb.NewScriptRepository(db)
	tx := mongodb.NewTransactor(db.Client())
	script := model.Script{Name: "backup", StorePath: "/srv/b.sh", StoreComp: "/opt/b.sh"}

	// Anulare: primul pas scrie, al doilea esueaza, deci nici scrierea nu trebuie sa ramana.
	failure := errors.New("al doilea pas a esuat")
	err := tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := scripts.Create(ctx, script); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("err = %v, vrem eroarea din tranzactie", err)
	}
	if _, total, _ := scripts.List(ctx, 0, 10); total != 0 {
		t.Errorf("dupa anulare au ramas %d scripturi, vrem 0", total)
	}

	// Commit: fara eroare, scrierea ramane.
	err = tx.WithinTransaction(ctx, func(ctx context.Context) error {
		_, err := scripts.Create(ctx, script)
		return err
	})
	if err != nil {
		t.Fatalf("tranzactie fara erori: %v", err)
	}
	if _, total, _ := scripts.List(ctx, 0, 10); total != 1 {
		t.Errorf("dupa commit sunt %d scripturi, vrem 1", total)
	}
}
