package memory

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/model"
	"job-management-service/internal/repository"
)

func TestComputerLinks(t *testing.T) {
	ctx := context.Background()
	repo := NewComputerRepository()
	computer, _ := repo.Create(ctx, model.Computer{IP: "10.0.0.1"})
	scriptA, scriptB := bson.NewObjectID(), bson.NewObjectID()

	// AddScript se comporta ca $addToSet: a doua oara nu mai adauga.
	if added, err := repo.AddScript(ctx, computer.ID, scriptA); !added || err != nil {
		t.Fatalf("AddScript = %v, %v", added, err)
	}
	if added, _ := repo.AddScript(ctx, computer.ID, scriptA); added {
		t.Error("al doilea AddScript a adaugat din nou")
	}
	repo.AddScript(ctx, computer.ID, scriptB)

	// Un document citit inainte nu se schimba cand lista e modificata dupa.
	before, _ := repo.Get(ctx, computer.ID)

	// PUT (Replace) schimba campurile proprii, dar pastreaza legaturile.
	repo.Replace(ctx, model.Computer{ID: computer.ID, IP: "10.0.0.99"})
	if got, _ := repo.Get(ctx, computer.ID); got.IP != "10.0.0.99" || len(got.IDScript) != 2 {
		t.Errorf("dupa Replace: %+v, vrem IP nou si 2 scripturi", got)
	}

	// RemoveScript se comporta ca $pull.
	if removed, _ := repo.RemoveScript(ctx, computer.ID, scriptA); !removed {
		t.Error("RemoveScript nu a gasit scriptul")
	}
	if removed, _ := repo.RemoveScript(ctx, computer.ID, scriptA); removed {
		t.Error("al doilea RemoveScript a raportat stergere")
	}
	if len(before.IDScript) != 2 {
		t.Errorf("copia citita inainte s-a modificat: %v", before.IDScript)
	}

	// RemoveScriptEverywhere scoate scriptul de pe toate calculatoarele.
	other, _ := repo.Create(ctx, model.Computer{IP: "10.0.0.2"})
	repo.AddScript(ctx, other.ID, scriptB)
	repo.RemoveScriptEverywhere(ctx, scriptB)
	for _, id := range []bson.ObjectID{computer.ID, other.ID} {
		if got, _ := repo.Get(ctx, id); len(got.IDScript) != 0 {
			t.Errorf("dupa RemoveScriptEverywhere, %s are %v", id.Hex(), got.IDScript)
		}
	}

	// Pe un calculator inexistent: ErrNotFound.
	if _, err := repo.AddScript(ctx, bson.NewObjectID(), scriptA); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("AddScript pe calculator inexistent: %v", err)
	}
}

func TestClusterAndUserLookups(t *testing.T) {
	ctx := context.Background()
	clusters := NewClusterRepository()
	users := NewUserRepository()

	cluster, _ := clusters.Create(ctx, model.Cluster{Nume: "laborator"})
	computerID := bson.NewObjectID()
	clusters.AddComputer(ctx, cluster.ID, computerID)

	if got, err := clusters.FindByComputer(ctx, computerID); err != nil || got.ID != cluster.ID {
		t.Errorf("FindByComputer = %+v, %v", got, err)
	}
	if _, err := clusters.FindByComputer(ctx, bson.NewObjectID()); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindByComputer pentru un calculator din niciun cluster: %v", err)
	}

	user, _ := users.Create(ctx, model.User{IDUserGRPC: "uid-1", UserName: "ana"})
	if got, err := users.FindByGRPCID(ctx, "uid-1"); err != nil || got.ID != user.ID {
		t.Errorf("FindByGRPCID = %+v, %v", got, err)
	}

	ref := model.ManagedRef{ID: cluster.ID, Tip: model.TipCluster}
	users.AddManaged(ctx, user.ID, ref)
	users.RemoveManagedEverywhere(ctx, ref)
	if got, _ := users.Get(ctx, user.ID); len(got.IDCompClust) != 0 {
		t.Errorf("dupa RemoveManagedEverywhere: %v", got.IDCompClust)
	}

	// GetMany ignora id-urile care nu exista si intoarce documentele in ordinea id-urilor.
	second, _ := clusters.Create(ctx, model.Cluster{Nume: "productie"})
	got, _ := clusters.GetMany(ctx, []bson.ObjectID{second.ID, bson.NewObjectID(), cluster.ID})
	if len(got) != 2 || got[0].ID != cluster.ID || got[1].ID != second.ID {
		t.Errorf("GetMany = %+v", got)
	}
}
