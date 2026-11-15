package service

import (
	"context"
	"errors"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/model"
	"job-management-service/internal/repository"
)

// scope este ce poate vedea utilizatorul curent (L3 2.1: un maintainer vede doar
// calculatoarele si clusterele care i-au fost repartizate).
type scope struct {
	all       bool // administratorul vede tot
	computers []bson.ObjectID
	clusters  []bson.ObjectID
}

func (s scope) canSeeComputer(id bson.ObjectID) bool {
	return s.all || slices.Contains(s.computers, id)
}
func (s scope) canSeeCluster(id bson.ObjectID) bool { return s.all || slices.Contains(s.clusters, id) }

// scopeOf calculeaza ce vede utilizatorul:
//   - adminul vede tot;
//   - un maintainer vede calculatoarele si clusterele din id_comp_clust-ul documentului sau
//     si, in plus, calculatoarele din clusterele asignate lui (cine gestioneaza un cluster
//     gestioneaza si calculatoarele din el).
//
// Legatura dintre token si documentul din MongoDB e uid-ul: identity.UID == user.id_user_grpc.
func (s *Service) scopeOf(ctx context.Context, identity auth.Identity) (scope, error) {
	if identity.IsAdmin() {
		return scope{all: true}, nil
	}
	user, err := s.users.FindByGRPCID(ctx, identity.UID)
	if errors.Is(err, repository.ErrNotFound) {
		return scope{}, nil // maintainer care nu a fost adaugat in /users: nu vede nimic
	}
	if err != nil {
		return scope{}, err
	}

	var result scope
	for _, ref := range user.IDCompClust {
		switch ref.Tip {
		case model.TipCalculator:
			result.computers = appendUnique(result.computers, ref.ID)
		case model.TipCluster:
			result.clusters = appendUnique(result.clusters, ref.ID)
		}
	}
	clusters, err := s.clusters.GetMany(ctx, result.clusters)
	if err != nil {
		return scope{}, err
	}
	for _, cluster := range clusters {
		for _, computerID := range cluster.IDComp {
			result.computers = appendUnique(result.computers, computerID)
		}
	}
	return result, nil
}

func appendUnique(ids []bson.ObjectID, id bson.ObjectID) []bson.ObjectID {
	if slices.Contains(ids, id) {
		return ids
	}
	return append(ids, id)
}

// ListComputers: adminul primeste pagina direct din baza; maintainerul doar calculatoarele lui.
func (s *Service) ListComputers(ctx context.Context, identity auth.Identity, offset, limit int) ([]model.Computer, int, error) {
	visible, err := s.scopeOf(ctx, identity)
	if err != nil {
		return nil, 0, err
	}
	if visible.all {
		return s.computers.List(ctx, offset, limit)
	}
	computers, err := s.computers.GetMany(ctx, visible.computers)
	if err != nil {
		return nil, 0, err
	}
	page, total := paginate(computers, offset, limit)
	return page, total, nil
}

// GetComputer: 404 daca nu exista, 403 daca exista dar nu e asignat maintainerului.
func (s *Service) GetComputer(ctx context.Context, identity auth.Identity, id bson.ObjectID) (model.Computer, error) {
	computer, err := s.computers.Get(ctx, id)
	if err != nil {
		return model.Computer{}, notFoundAs(err, "Calculatorul cerut nu exista.")
	}
	visible, err := s.scopeOf(ctx, identity)
	if err != nil {
		return model.Computer{}, err
	}
	if !visible.canSeeComputer(id) {
		return model.Computer{}, ErrNotAssigned
	}
	return computer, nil
}

func (s *Service) ListClusters(ctx context.Context, identity auth.Identity, offset, limit int) ([]model.Cluster, int, error) {
	visible, err := s.scopeOf(ctx, identity)
	if err != nil {
		return nil, 0, err
	}
	if visible.all {
		return s.clusters.List(ctx, offset, limit)
	}
	clusters, err := s.clusters.GetMany(ctx, visible.clusters)
	if err != nil {
		return nil, 0, err
	}
	page, total := paginate(clusters, offset, limit)
	return page, total, nil
}

func (s *Service) GetCluster(ctx context.Context, identity auth.Identity, id bson.ObjectID) (model.Cluster, error) {
	cluster, err := s.clusters.Get(ctx, id)
	if err != nil {
		return model.Cluster{}, notFoundAs(err, "Clusterul cerut nu exista.")
	}
	visible, err := s.scopeOf(ctx, identity)
	if err != nil {
		return model.Cluster{}, err
	}
	if !visible.canSeeCluster(id) {
		return model.Cluster{}, ErrNotAssigned
	}
	return cluster, nil
}
