package repository

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/model"
)

// ErrNotFound: documentul cerut nu exista.
var ErrNotFound = errors.New("documentul nu exista")

// DuplicateError: un camp unic (SP, Tabelele 2-4) are deja aceeasi valoare in alt document.
type DuplicateError struct {
	Field string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("exista deja un document cu aceeasi valoare pentru %s", e.Field)
}

// Repository este accesul la o colectie de documente de tip T.
// Implementari: memory (acum) si mongo (Faza 5); handlerele nu stiu care e folosita.
type Repository[T any] interface {
	// List intoarce cel mult limit documente, sarind primele offset, plus numarul lor total.
	List(ctx context.Context, offset, limit int) ([]T, int, error)
	Get(ctx context.Context, id bson.ObjectID) (T, error)
	// GetMany intoarce documentele cu id-urile date, in ordinea id-urilor.
	// Id-urile care nu exista sunt ignorate.
	GetMany(ctx context.Context, ids []bson.ObjectID) ([]T, error)
	// Create genereaza un id nou, salveaza documentul si il intoarce cu id-ul completat.
	Create(ctx context.Context, doc T) (T, error)
	// Replace inlocuieste campurile proprii ale documentului cu acelasi id sau il creeaza
	// daca nu exista. Listele de legaturi (id_script, id_users etc.) raman neschimbate.
	// created spune care dintre cazuri a avut loc (201 sau 204 in handler).
	Replace(ctx context.Context, doc T) (created bool, err error)
	// Delete sterge documentul si il intoarce asa cum era inainte de stergere.
	Delete(ctx context.Context, id bson.ObjectID) (T, error)
}

// ScriptRepository: scripturile nu au liste de legaturi (SP, Tabelul 5).
type ScriptRepository = Repository[model.Script]

// Metodele de legatura urmeaza operatiile MongoDB pe care le vom folosi in Faza 5:
// Add... = $addToSet (nu adauga de doua ori), Remove... = $pull.
// added/removed spun daca lista s-a schimbat; ErrNotFound daca documentul nu exista.

// ComputerRepository: calculatoarele si listele lor id_script si id_users (SP, Tabelul 3).
type ComputerRepository interface {
	Repository[model.Computer]
	AddScript(ctx context.Context, computerID, scriptID bson.ObjectID) (added bool, err error)
	RemoveScript(ctx context.Context, computerID, scriptID bson.ObjectID) (removed bool, err error)
	// RemoveScriptEverywhere scoate scriptul de pe toate calculatoarele (stergere in cascada).
	RemoveScriptEverywhere(ctx context.Context, scriptID bson.ObjectID) error
	AddUser(ctx context.Context, computerID, userID bson.ObjectID) (added bool, err error)
	RemoveUser(ctx context.Context, computerID, userID bson.ObjectID) (removed bool, err error)
	RemoveUserEverywhere(ctx context.Context, userID bson.ObjectID) error
}

// ClusterRepository: clusterele si listele lor id_comp si id_users (SP, Tabelul 4).
type ClusterRepository interface {
	Repository[model.Cluster]
	AddComputer(ctx context.Context, clusterID, computerID bson.ObjectID) (added bool, err error)
	RemoveComputer(ctx context.Context, clusterID, computerID bson.ObjectID) (removed bool, err error)
	RemoveComputerEverywhere(ctx context.Context, computerID bson.ObjectID) error
	AddUser(ctx context.Context, clusterID, userID bson.ObjectID) (added bool, err error)
	RemoveUser(ctx context.Context, clusterID, userID bson.ObjectID) (removed bool, err error)
	RemoveUserEverywhere(ctx context.Context, userID bson.ObjectID) error
	// FindByComputer intoarce clusterul care contine calculatorul, sau ErrNotFound.
	FindByComputer(ctx context.Context, computerID bson.ObjectID) (model.Cluster, error)
}

// UserRepository: utilizatorii si lista lor id_comp_clust (SP, Tabelul 2).
type UserRepository interface {
	Repository[model.User]
	// FindByGRPCID cauta documentul contului din A&A Services cu uid-ul dat, sau ErrNotFound.
	FindByGRPCID(ctx context.Context, uid string) (model.User, error)
	AddManaged(ctx context.Context, userID bson.ObjectID, ref model.ManagedRef) (added bool, err error)
	RemoveManaged(ctx context.Context, userID bson.ObjectID, ref model.ManagedRef) (removed bool, err error)
	// RemoveManagedEverywhere scoate ref din lista tuturor utilizatorilor (stergere in cascada).
	RemoveManagedEverywhere(ctx context.Context, ref model.ManagedRef) error
}

// Transactor ruleaza mai multe operatii, pe colectii diferite, ca un tot unitar:
// o asignare scrie si in calculator, si in utilizator. In memorie e un mutex;
// in MongoDB (Faza 5) va fi o tranzactie.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
