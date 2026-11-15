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

// AddComputerToCluster adauga calculatorul in cluster (SP 1.3). created e false daca era deja acolo.
// Un calculator poate fi intr-un singur cluster: SP spune ca "toate calculatoarele inglobate in
// acelasi cluster au aceeasi configuratie", iar doua clustere i-ar putea cere configuratii diferite.
func (s *Service) AddComputerToCluster(ctx context.Context, clusterID, computerID bson.ObjectID) (created bool, err error) {
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := s.clusters.Get(ctx, clusterID); err != nil {
			return notFoundAs(err, "Clusterul cerut nu exista.")
		}
		if _, err := s.computers.Get(ctx, computerID); err != nil {
			return notFoundAs(err, "Calculatorul cerut nu exista.")
		}
		current, err := s.clusters.FindByComputer(ctx, computerID)
		switch {
		case err == nil && current.ID != clusterID:
			return &ConflictError{Detail: "Calculatorul face deja parte din clusterul " + current.Nume + "; scoate-l intai de acolo."}
		case err != nil && !errors.Is(err, repository.ErrNotFound):
			return err
		}
		created, err = s.clusters.AddComputer(ctx, clusterID, computerID)
		return err
	})
	return created, err
}

// RemoveComputerFromCluster elimina calculatorul din cluster (SP 1.3).
func (s *Service) RemoveComputerFromCluster(ctx context.Context, clusterID, computerID bson.ObjectID) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		removed, err := s.clusters.RemoveComputer(ctx, clusterID, computerID)
		if err != nil {
			return notFoundAs(err, "Clusterul cerut nu exista.")
		}
		if !removed {
			return &NotFoundError{Detail: "Calculatorul nu face parte din acest cluster."}
		}
		return nil
	})
}

// ClusterComputers: calculatoarele dintr-un cluster.
func (s *Service) ClusterComputers(ctx context.Context, identity auth.Identity, clusterID bson.ObjectID, offset, limit int) ([]model.Computer, int, error) {
	cluster, err := s.GetCluster(ctx, identity, clusterID)
	if err != nil {
		return nil, 0, err
	}
	computers, err := s.computers.GetMany(ctx, cluster.IDComp)
	if err != nil {
		return nil, 0, err
	}
	page, total := paginate(computers, offset, limit)
	return page, total, nil
}

// ClusterComputer: un calculator din cluster; 404 daca nu face parte din el.
func (s *Service) ClusterComputer(ctx context.Context, identity auth.Identity, clusterID, computerID bson.ObjectID) (model.Computer, error) {
	cluster, err := s.GetCluster(ctx, identity, clusterID)
	if err != nil {
		return model.Computer{}, err
	}
	if !slices.Contains(cluster.IDComp, computerID) {
		return model.Computer{}, &NotFoundError{Detail: "Calculatorul nu face parte din acest cluster."}
	}
	computer, err := s.computers.Get(ctx, computerID)
	return computer, notFoundAs(err, "Calculatorul cerut nu exista.")
}
