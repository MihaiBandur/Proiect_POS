package service

import (
	"context"
	"log"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/model"
)

// mockStatus e mesajul unui task simulat: fara executor sau fara Job Status Service, serviciul
// "va furniza raspunsuri dummy/mock" (SP 1.3).
const mockStatus = "mock: rulare simulata (lipsesc IPC_SOCKET_PATH sau JOB_STATUS_URL)"

// LaunchOnComputer porneste rularea unui script pe un calculator (SP 1.3: maintainerul
// "initializeaza rularea task-urilor noi"). Ruta permite doar maintainerii; aici verificam ca
// calculatorul exista (404) si e al maintainerului (403), iar scriptul e instalat pe el (409).
func (s *Service) LaunchOnComputer(ctx context.Context, identity auth.Identity, computerID, scriptID bson.ObjectID) (model.Task, error) {
	computer, err := s.GetComputer(ctx, identity, computerID)
	if err != nil {
		return model.Task{}, err
	}
	script, err := s.runnableScript(ctx, scriptID)
	if err != nil {
		return model.Task{}, err
	}
	if !slices.Contains(computer.IDScript, scriptID) {
		return model.Task{}, &ConflictError{Detail: "Scriptul nu este instalat pe acest calculator."}
	}
	return s.launch(ctx, identity, model.ManagedRef{ID: computerID, Tip: model.TipCalculator}, script)
}

// LaunchOnCluster porneste rularea pe toate calculatoarele unui cluster, in paralel
// (executorul din kit stie sa ruleze pe un cluster). Toate calculatoarele trebuie sa aiba
// scriptul instalat: SP spune ca au "aceeasi configuratie".
func (s *Service) LaunchOnCluster(ctx context.Context, identity auth.Identity, clusterID, scriptID bson.ObjectID) (model.Task, error) {
	cluster, err := s.GetCluster(ctx, identity, clusterID)
	if err != nil {
		return model.Task{}, err
	}
	script, err := s.runnableScript(ctx, scriptID)
	if err != nil {
		return model.Task{}, err
	}
	if len(cluster.IDComp) == 0 {
		return model.Task{}, &ConflictError{Detail: "Clusterul nu are niciun calculator."}
	}
	computers, err := s.computers.GetMany(ctx, cluster.IDComp)
	if err != nil {
		return model.Task{}, err
	}
	for _, computer := range computers {
		if !slices.Contains(computer.IDScript, scriptID) {
			return model.Task{}, &ConflictError{Detail: "Scriptul nu este instalat pe toate calculatoarele din cluster (lipseste pe " + computer.IP + ")."}
		}
	}
	return s.launch(ctx, identity, model.ManagedRef{ID: clusterID, Tip: model.TipCluster}, script)
}

// runnableScript intoarce scriptul daca exista (altfel 404) si e activ (altfel 409).
func (s *Service) runnableScript(ctx context.Context, scriptID bson.ObjectID) (model.Script, error) {
	script, err := s.scripts.Get(ctx, scriptID)
	if err != nil {
		return model.Script{}, notFoundAs(err, "Scriptul cerut nu exista.")
	}
	if !script.IsActive {
		// Executorul ruleaza doar scripturi active (SP, Tabelul 5: lipsa lui is_active = inactiv).
		return model.Script{}, &ConflictError{Detail: "Scriptul nu este activ (is_active: false)."}
	}
	return script, nil
}

// launch creeaza task-ul (pending) si anunta executorul prin socket (student-kit/README.md).
func (s *Service) launch(ctx context.Context, identity auth.Identity, target model.ManagedRef, script model.Script) (model.Task, error) {
	// Daca maintainerul vede tinta, are un document in /users (altfel nu ar vedea nimic).
	user, err := s.users.FindByGRPCID(ctx, identity.UID)
	if err != nil {
		return model.Task{}, notFoundAs(err, "Utilizatorul nu este inregistrat in /users.")
	}

	task := model.Task{
		IDUser:   user.ID,
		IDScript: script.ID,
		Target:   target,
		// MongoDB pastreaza datele cu precizie de milisecunda; rotunjim la fel si in memorie.
		Date:  time.Now().UTC().Truncate(time.Millisecond),
		State: model.TaskPending,
	}
	if s.executor.Mock() {
		task.State, task.Status = model.TaskFinished, mockStatus
		return s.tasks.Create(ctx, task)
	}

	task, err = s.tasks.Create(ctx, task)
	if err != nil {
		return model.Task{}, err
	}
	// Pe socket trimitem doar id-ul. Daca mesajul se pierde (executor oprit sau repornit),
	// task-ul ramane pending si executorul il gaseste la urmatoarea scanare (la 10 s), deci
	// raspundem tot 202: cererea a fost acceptata si va fi rulata.
	if err := s.executor.Run(ctx, task.ID.Hex()); err != nil {
		log.Printf("task %s: executorul nu a fost anuntat (il va gasi la scanare): %v", task.ID.Hex(), err)
	}
	return task, nil
}

// ListTasks: adminul vede toate task-urile, maintainerul doar pe ale lui (SP 1.4).
func (s *Service) ListTasks(ctx context.Context, identity auth.Identity, offset, limit int) ([]model.Task, int, error) {
	if identity.IsAdmin() {
		return s.tasks.List(ctx, offset, limit)
	}
	user, err := s.users.FindByGRPCID(ctx, identity.UID)
	if isNotFound(err) {
		return nil, 0, nil // nu e in /users, deci n-a pornit niciun task
	}
	if err != nil {
		return nil, 0, err
	}
	return s.tasks.ListByUser(ctx, user.ID, offset, limit)
}

// GetTask: 404 daca nu exista; 403 pentru task-ul altui maintainer.
func (s *Service) GetTask(ctx context.Context, identity auth.Identity, id bson.ObjectID) (model.Task, error) {
	task, err := s.tasks.Get(ctx, id)
	if err != nil {
		return model.Task{}, notFoundAs(err, "Task-ul cerut nu exista.")
	}
	if identity.IsAdmin() {
		return task, nil
	}
	user, err := s.users.FindByGRPCID(ctx, identity.UID)
	if err != nil || user.ID != task.IDUser {
		return model.Task{}, ErrNotAssigned
	}
	return task, nil
}
