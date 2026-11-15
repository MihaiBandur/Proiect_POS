package memory

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/model"
	"job-management-service/internal/repository"
)

func TestCollectionCRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewScriptRepository()

	// Create: id nou pentru fiecare document, in ordinea adaugarii.
	var ids []bson.ObjectID
	for _, name := range []string{"a", "b", "c"} {
		script, err := repo.Create(ctx, model.Script{Name: name})
		if err != nil || script.ID.IsZero() {
			t.Fatalf("Create(%s) = %+v, %v", name, script, err)
		}
		ids = append(ids, script.ID)
	}

	// List: paginile urmeaza ordinea id-urilor; dupa final, lista e goala.
	page, total, _ := repo.List(ctx, 1, 2)
	if total != 3 || len(page) != 2 || page[0].Name != "b" || page[1].Name != "c" {
		t.Errorf("List(1, 2) = %+v, total %d", page, total)
	}
	if page, _, _ := repo.List(ctx, 10, 2); len(page) != 0 {
		t.Errorf("List dupa final = %+v, vrem gol", page)
	}

	// Get: gaseste un document existent; ErrNotFound pentru un id necunoscut.
	if script, err := repo.Get(ctx, ids[0]); err != nil || script.Name != "a" {
		t.Errorf("Get = %+v, %v", script, err)
	}
	if _, err := repo.Get(ctx, bson.NewObjectID()); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Get(id necunoscut) err = %v, vrem ErrNotFound", err)
	}

	// Replace: false pentru un document existent, true cand il creeaza.
	if created, _ := repo.Replace(ctx, model.Script{ID: ids[0], Name: "a2"}); created {
		t.Error("Replace pe un id existent a raportat creare")
	}
	if script, _ := repo.Get(ctx, ids[0]); script.Name != "a2" {
		t.Errorf("dupa Replace, Name = %q, vrem a2", script.Name)
	}
	newID := bson.NewObjectID()
	if created, _ := repo.Replace(ctx, model.Script{ID: newID, Name: "d"}); !created {
		t.Error("Replace pe un id nou nu a raportat creare")
	}

	// Delete: intoarce documentul sters; a doua oara, ErrNotFound.
	if script, err := repo.Delete(ctx, newID); err != nil || script.Name != "d" {
		t.Errorf("Delete = %+v, %v", script, err)
	}
	if _, err := repo.Delete(ctx, newID); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("al doilea Delete err = %v, vrem ErrNotFound", err)
	}
}

// duplicateField intoarce campul raportat de un DuplicateError, sau "" pentru alta eroare.
func duplicateField(err error) string {
	var dup *repository.DuplicateError
	if errors.As(err, &dup) {
		return dup.Field
	}
	return ""
}

func TestCollectionUnique(t *testing.T) {
	ctx := context.Background()
	repo := NewComputerRepository()

	first, err := repo.Create(ctx, model.Computer{IP: "10.0.0.1", Hostname: "node-1"})
	if err != nil {
		t.Fatal(err)
	}

	// Hostname-ul poate lipsi la mai multe calculatoare: valorile goale nu se compara.
	if _, err := repo.Create(ctx, model.Computer{IP: "10.0.0.2"}); err != nil {
		t.Errorf("primul calculator fara hostname: %v", err)
	}
	second, err := repo.Create(ctx, model.Computer{IP: "10.0.0.3"})
	if err != nil {
		t.Errorf("al doilea calculator fara hostname: %v", err)
	}

	tests := []struct {
		name      string
		do        func() error
		wantField string // "" = fara conflict
	}{
		{"IP duplicat la Create", func() error {
			_, err := repo.Create(ctx, model.Computer{IP: "10.0.0.1"})
			return err
		}, "IP"},
		{"hostname duplicat la Create", func() error {
			_, err := repo.Create(ctx, model.Computer{IP: "10.0.0.9", Hostname: "node-1"})
			return err
		}, "hostname"},
		{"Replace cu propriile valori", func() error {
			_, err := repo.Replace(ctx, first)
			return err
		}, ""},
		{"Replace cu IP-ul altui calculator", func() error {
			_, err := repo.Replace(ctx, model.Computer{ID: second.ID, IP: "10.0.0.1"})
			return err
		}, "IP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.do()
			if got := duplicateField(err); got != tt.wantField || (tt.wantField == "" && err != nil) {
				t.Errorf("err = %v, vrem conflict pe %q", err, tt.wantField)
			}
		})
	}

	// Un conflict nu modifica documentul existent.
	if got, _ := repo.Get(ctx, second.ID); got.IP != "10.0.0.3" {
		t.Errorf("dupa un Replace respins, IP = %q, vrem 10.0.0.3", got.IP)
	}
}
