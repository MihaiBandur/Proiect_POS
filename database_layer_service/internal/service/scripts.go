package service

import (
	"context"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/model"
)

// InstallScript instaleaza scriptul pe calculator (SP 1.3). created e false daca era deja instalat.
func (s *Service) InstallScript(ctx context.Context, computerID, scriptID bson.ObjectID) (created bool, err error) {
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := s.computers.Get(ctx, computerID); err != nil {
			return notFoundAs(err, "Calculatorul cerut nu exista.")
		}
		// SP, Tabelul 3: valorile din id_script trebuie sa existe in colectia de scripturi.
		if _, err := s.scripts.Get(ctx, scriptID); err != nil {
			return notFoundAs(err, "Scriptul cerut nu exista.")
		}
		created, err = s.computers.AddScript(ctx, computerID, scriptID)
		return err
	})
	return created, err
}

// UninstallScript revoca scriptul de pe calculator (SP 1.3).
func (s *Service) UninstallScript(ctx context.Context, computerID, scriptID bson.ObjectID) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		removed, err := s.computers.RemoveScript(ctx, computerID, scriptID)
		if err != nil {
			return notFoundAs(err, "Calculatorul cerut nu exista.")
		}
		if !removed {
			return &NotFoundError{Detail: "Scriptul nu este instalat pe acest calculator."}
		}
		return nil
	})
}

// ComputerScripts: scripturile instalate pe un calculator (L3 2.1: /computers/{id}/scripts),
// filtrate optional dupa is_active (?isActive=true|false).
func (s *Service) ComputerScripts(ctx context.Context, identity auth.Identity, computerID bson.ObjectID, isActive *bool, offset, limit int) ([]model.Script, int, error) {
	computer, err := s.GetComputer(ctx, identity, computerID)
	if err != nil {
		return nil, 0, err
	}
	scripts, err := s.scripts.GetMany(ctx, computer.IDScript)
	if err != nil {
		return nil, 0, err
	}
	if isActive != nil {
		filtered := make([]model.Script, 0, len(scripts))
		for _, script := range scripts {
			if script.IsActive == *isActive {
				filtered = append(filtered, script)
			}
		}
		scripts = filtered
	}
	page, total := paginate(scripts, offset, limit)
	return page, total, nil
}

// ComputerScript: un singur script instalat pe calculator; 404 daca nu e instalat.
func (s *Service) ComputerScript(ctx context.Context, identity auth.Identity, computerID, scriptID bson.ObjectID) (model.Script, error) {
	computer, err := s.GetComputer(ctx, identity, computerID)
	if err != nil {
		return model.Script{}, err
	}
	if !slices.Contains(computer.IDScript, scriptID) {
		return model.Script{}, &NotFoundError{Detail: "Scriptul nu este instalat pe acest calculator."}
	}
	script, err := s.scripts.Get(ctx, scriptID)
	return script, notFoundAs(err, "Scriptul cerut nu exista.")
}
