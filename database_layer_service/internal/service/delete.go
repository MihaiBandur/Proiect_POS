package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/model"
	"job-management-service/internal/repository"
)

// Stergerea in cascada: SP cere ca valorile din listele de legaturi sa existe in colectiile
// tinta (Tabelele 2-4), deci cand stergem o resursa o scoatem si din listele care o refera.
// Totul ruleaza intr-o singura tranzactie, ca nimeni sa nu vada o legatura catre o resursa stearsa.

func (s *Service) DeleteScript(ctx context.Context, id bson.ObjectID) (script model.Script, err error) {
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if script, err = s.scripts.Delete(ctx, id); err != nil {
			return notFoundAs(err, "Scriptul cerut nu exista.")
		}
		return s.computers.RemoveScriptEverywhere(ctx, id)
	})
	return script, err
}

func (s *Service) DeleteComputer(ctx context.Context, id bson.ObjectID) (computer model.Computer, err error) {
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if computer, err = s.computers.Delete(ctx, id); err != nil {
			return notFoundAs(err, "Calculatorul cerut nu exista.")
		}
		if err := s.clusters.RemoveComputerEverywhere(ctx, id); err != nil {
			return err
		}
		return s.users.RemoveManagedEverywhere(ctx, model.ManagedRef{ID: id, Tip: model.TipCalculator})
	})
	return computer, err
}

func (s *Service) DeleteCluster(ctx context.Context, id bson.ObjectID) (cluster model.Cluster, err error) {
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if cluster, err = s.clusters.Delete(ctx, id); err != nil {
			return notFoundAs(err, "Clusterul cerut nu exista.")
		}
		return s.users.RemoveManagedEverywhere(ctx, model.ManagedRef{ID: id, Tip: model.TipCluster})
	})
	return cluster, err
}

// DeleteUser sterge documentul din MongoDB si asignarile lui; contul din A&A Services ramane.
func (s *Service) DeleteUser(ctx context.Context, id bson.ObjectID) (user model.User, err error) {
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if user, err = s.users.Delete(ctx, id); err != nil {
			return notFoundAs(err, "Utilizatorul cerut nu exista.")
		}
		if err := s.computers.RemoveUserEverywhere(ctx, id); err != nil {
			return err
		}
		return s.clusters.RemoveUserEverywhere(ctx, id)
	})
	return user, err
}

func isNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}
