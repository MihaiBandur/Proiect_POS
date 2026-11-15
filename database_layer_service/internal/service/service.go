// Package service contine regulile care implica mai multe colectii: legaturile dintre resurse,
// stergerea in cascada si ce vede un maintainer. Operatiile pe o singura colectie (POST, PUT)
// raman in handlere, direct pe repository.
package service

import (
	"job-management-service/internal/auth"
	"job-management-service/internal/repository"
)

type Service struct {
	tx        repository.Transactor
	scripts   repository.ScriptRepository
	computers repository.ComputerRepository
	clusters  repository.ClusterRepository
	users     repository.UserRepository
	accounts  auth.AccountDirectory
}

func New(
	tx repository.Transactor,
	scripts repository.ScriptRepository,
	computers repository.ComputerRepository,
	clusters repository.ClusterRepository,
	users repository.UserRepository,
	accounts auth.AccountDirectory,
) *Service {
	return &Service{tx: tx, scripts: scripts, computers: computers, clusters: clusters, users: users, accounts: accounts}
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
