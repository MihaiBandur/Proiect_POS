package service

import (
	"context"
	"errors"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/model"
)

// Asignarea unui maintainer (SP 1.3) se scrie in doua locuri, ca legatura sa poata fi
// parcursa din ambele sensuri: id_users din calculator/cluster si id_comp_clust din utilizator.
// target descrie partea calculator/cluster; restul logicii e aceeasi pentru amandoua.
type assignTarget struct {
	ref      model.ManagedRef                                             // ce se scrie in id_comp_clust
	notFound string                                                       // mesajul cand calculatorul/clusterul lipseste
	get      func(ctx context.Context) (users []bson.ObjectID, err error) // lista id_users curenta
	add      func(ctx context.Context, userID bson.ObjectID) (bool, error)
	remove   func(ctx context.Context, userID bson.ObjectID) (bool, error)
}

func (s *Service) computerTarget(id bson.ObjectID) assignTarget {
	return assignTarget{
		ref:      model.ManagedRef{ID: id, Tip: model.TipCalculator},
		notFound: "Calculatorul cerut nu exista.",
		get: func(ctx context.Context) ([]bson.ObjectID, error) {
			computer, err := s.computers.Get(ctx, id)
			return computer.IDUsers, err
		},
		add: func(ctx context.Context, userID bson.ObjectID) (bool, error) {
			return s.computers.AddUser(ctx, id, userID)
		},
		remove: func(ctx context.Context, userID bson.ObjectID) (bool, error) {
			return s.computers.RemoveUser(ctx, id, userID)
		},
	}
}

func (s *Service) clusterTarget(id bson.ObjectID) assignTarget {
	return assignTarget{
		ref:      model.ManagedRef{ID: id, Tip: model.TipCluster},
		notFound: "Clusterul cerut nu exista.",
		get: func(ctx context.Context) ([]bson.ObjectID, error) {
			cluster, err := s.clusters.Get(ctx, id)
			return cluster.IDUsers, err
		},
		add: func(ctx context.Context, userID bson.ObjectID) (bool, error) {
			return s.clusters.AddUser(ctx, id, userID)
		},
		remove: func(ctx context.Context, userID bson.ObjectID) (bool, error) {
			return s.clusters.RemoveUser(ctx, id, userID)
		},
	}
}

// requireMaintainer verifica in A&A Services ca utilizatorul are rolul de maintainer
// (SP 1.3: "asigneaza utilizatori cu rol de maintainer"). token e token-ul adminului.
func (s *Service) requireMaintainer(ctx context.Context, token string, user model.User) error {
	account, err := s.accounts.FindAccount(ctx, token, user.IDUserGRPC)
	switch {
	case errors.Is(err, auth.ErrAccountNotFound):
		return &ConflictError{Detail: "Contul utilizatorului nu mai exista in A&A Services."}
	case err != nil:
		return err
	case account.Role != auth.RoleMaintainer:
		return &ConflictError{Detail: "Utilizatorul nu are rolul de maintainer in A&A Services."}
	}
	return nil
}

func (s *Service) assign(ctx context.Context, token string, target assignTarget, userID bson.ObjectID) (created bool, err error) {
	// Rolul il verificam inaintea tranzactiei: e un apel prin retea si nu vrem sa tinem
	// celelalte asignari blocate cat asteptam raspunsul de la A&A Services.
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return false, notFoundAs(err, "Utilizatorul cerut nu exista.")
	}
	if err := s.requireMaintainer(ctx, token, user); err != nil {
		return false, err
	}

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := target.get(ctx); err != nil {
			return notFoundAs(err, target.notFound)
		}
		if _, err := s.users.Get(ctx, userID); err != nil { // poate fi sters intre timp
			return notFoundAs(err, "Utilizatorul cerut nu exista.")
		}
		if created, err = target.add(ctx, userID); err != nil {
			return err
		}
		_, err = s.users.AddManaged(ctx, userID, target.ref)
		return err
	})
	return created, err
}

func (s *Service) unassign(ctx context.Context, target assignTarget, userID bson.ObjectID) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		removed, err := target.remove(ctx, userID)
		if err != nil {
			return notFoundAs(err, target.notFound)
		}
		if !removed {
			return &NotFoundError{Detail: "Utilizatorul nu este asignat aici."}
		}
		// Utilizatorul poate lipsi daca e sters chiar acum; atunci nu mai avem ce actualiza.
		if _, err := s.users.RemoveManaged(ctx, userID, target.ref); err != nil && !isNotFound(err) {
			return err
		}
		return nil
	})
}

// assigned intoarce maintainerii asignati (documentele din /users), paginati.
func (s *Service) assigned(ctx context.Context, target assignTarget, offset, limit int) ([]model.User, int, error) {
	ids, err := target.get(ctx)
	if err != nil {
		return nil, 0, notFoundAs(err, target.notFound)
	}
	users, err := s.users.GetMany(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	page, total := paginate(users, offset, limit)
	return page, total, nil
}

// assignedOne intoarce un maintainer asignat; 404 daca nu e asignat aici.
func (s *Service) assignedOne(ctx context.Context, target assignTarget, userID bson.ObjectID) (model.User, error) {
	ids, err := target.get(ctx)
	if err != nil {
		return model.User{}, notFoundAs(err, target.notFound)
	}
	if !slices.Contains(ids, userID) {
		return model.User{}, &NotFoundError{Detail: "Utilizatorul nu este asignat aici."}
	}
	user, err := s.users.Get(ctx, userID)
	return user, notFoundAs(err, "Utilizatorul cerut nu exista.")
}

// Operatiile publice, pe calculatoare si pe clustere.

func (s *Service) AssignToComputer(ctx context.Context, token string, computerID, userID bson.ObjectID) (bool, error) {
	return s.assign(ctx, token, s.computerTarget(computerID), userID)
}

func (s *Service) UnassignFromComputer(ctx context.Context, computerID, userID bson.ObjectID) error {
	return s.unassign(ctx, s.computerTarget(computerID), userID)
}

func (s *Service) ComputerUsers(ctx context.Context, computerID bson.ObjectID, offset, limit int) ([]model.User, int, error) {
	return s.assigned(ctx, s.computerTarget(computerID), offset, limit)
}

func (s *Service) ComputerUser(ctx context.Context, computerID, userID bson.ObjectID) (model.User, error) {
	return s.assignedOne(ctx, s.computerTarget(computerID), userID)
}

func (s *Service) AssignToCluster(ctx context.Context, token string, clusterID, userID bson.ObjectID) (bool, error) {
	return s.assign(ctx, token, s.clusterTarget(clusterID), userID)
}

func (s *Service) UnassignFromCluster(ctx context.Context, clusterID, userID bson.ObjectID) error {
	return s.unassign(ctx, s.clusterTarget(clusterID), userID)
}

func (s *Service) ClusterUsers(ctx context.Context, clusterID bson.ObjectID, offset, limit int) ([]model.User, int, error) {
	return s.assigned(ctx, s.clusterTarget(clusterID), offset, limit)
}

func (s *Service) ClusterUser(ctx context.Context, clusterID, userID bson.ObjectID) (model.User, error) {
	return s.assignedOne(ctx, s.clusterTarget(clusterID), userID)
}
