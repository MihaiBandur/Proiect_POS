// Package service contine regulile care implica mai multe colectii: legaturile dintre resurse,
// stergerea in cascada si ce vede un maintainer. Operatiile pe o singura colectie (POST, PUT)
// raman in handlere, direct pe repository.
package service

import (
	"job-management-service/internal/auth"
	"job-management-service/internal/executor"
	"job-management-service/internal/repository"
)

type Service struct {
	tx        repository.Transactor
	scripts   repository.ScriptRepository
	computers repository.ComputerRepository
	clusters  repository.ClusterRepository
	users     repository.UserRepository
	tasks     repository.TaskRepository
	accounts  auth.AccountDirectory
	executor  executor.Dispatcher
}

// Config sunt dependentele serviciului; numele campurilor fac apelul usor de citit.
type Config struct {
	Tx        repository.Transactor
	Scripts   repository.ScriptRepository
	Computers repository.ComputerRepository
	Clusters  repository.ClusterRepository
	Users     repository.UserRepository
	Tasks     repository.TaskRepository
	Accounts  auth.AccountDirectory
	Executor  executor.Dispatcher
}

func New(cfg Config) *Service {
	return &Service{
		tx:        cfg.Tx,
		scripts:   cfg.Scripts,
		computers: cfg.Computers,
		clusters:  cfg.Clusters,
		users:     cfg.Users,
		tasks:     cfg.Tasks,
		accounts:  cfg.Accounts,
		executor:  cfg.Executor,
	}
}

// paginate intoarce pagina ceruta dintr-o lista deja filtrata, plus numarul total de elemente.
// Listele de legaturi (scripturile unui calculator, resursele unui maintainer) sunt mici,
// asa ca le filtram si paginam in serviciu, la fel pentru memorie si pentru MongoDB.
func paginate[T any](items []T, offset, limit int) ([]T, int) {
	total := len(items)
	start := min(offset, total)
	end := min(start+limit, total)
	return items[start:end], total
}
