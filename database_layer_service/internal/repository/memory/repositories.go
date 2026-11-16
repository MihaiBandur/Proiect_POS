package memory

import (
	"context"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/model"
	"job-management-service/internal/repository"
)

// Cate un constructor pentru fiecare colectie din SP. Diferenta dintre ele: cum se ajunge
// la id, care campuri sunt unice (coloana "Constrangeri" din Tabelele 2-5) si ce legaturi au.

func NewScriptRepository() repository.ScriptRepository {
	return newCollection(spec[model.Script]{
		idOf:  func(s model.Script) bson.ObjectID { return s.ID },
		setID: func(s *model.Script, id bson.ObjectID) { s.ID = id },
		// scripturile nu au campuri unice si nici legaturi (SP, Tabelul 5)
	})
}

// ---------- calculatoare ----------

type computerRepository struct {
	*collection[model.Computer]
}

func NewComputerRepository() repository.ComputerRepository {
	return &computerRepository{newCollection(spec[model.Computer]{
		idOf:  func(c model.Computer) bson.ObjectID { return c.ID },
		setID: func(c *model.Computer, id bson.ObjectID) { c.ID = id },
		unique: func(c model.Computer) []uniqueField {
			return []uniqueField{{"IP", c.IP}, {"hostname", c.Hostname}}
		},
		keepLinks: func(next *model.Computer, prev model.Computer) {
			next.IDScript, next.IDUsers = prev.IDScript, prev.IDUsers
		},
	})}
}

func (r *computerRepository) AddScript(_ context.Context, computerID, scriptID bson.ObjectID) (bool, error) {
	return r.update(computerID, func(c *model.Computer) bool { return addTo(&c.IDScript, scriptID) })
}

func (r *computerRepository) RemoveScript(_ context.Context, computerID, scriptID bson.ObjectID) (bool, error) {
	return r.update(computerID, func(c *model.Computer) bool { return removeFrom(&c.IDScript, scriptID) })
}

func (r *computerRepository) RemoveScriptEverywhere(_ context.Context, scriptID bson.ObjectID) error {
	r.updateAll(func(c *model.Computer) bool { return removeFrom(&c.IDScript, scriptID) })
	return nil
}

func (r *computerRepository) AddUser(_ context.Context, computerID, userID bson.ObjectID) (bool, error) {
	return r.update(computerID, func(c *model.Computer) bool { return addTo(&c.IDUsers, userID) })
}

func (r *computerRepository) RemoveUser(_ context.Context, computerID, userID bson.ObjectID) (bool, error) {
	return r.update(computerID, func(c *model.Computer) bool { return removeFrom(&c.IDUsers, userID) })
}

func (r *computerRepository) RemoveUserEverywhere(_ context.Context, userID bson.ObjectID) error {
	r.updateAll(func(c *model.Computer) bool { return removeFrom(&c.IDUsers, userID) })
	return nil
}

// ---------- clustere ----------

type clusterRepository struct {
	*collection[model.Cluster]
}

func NewClusterRepository() repository.ClusterRepository {
	return &clusterRepository{newCollection(spec[model.Cluster]{
		idOf:   func(c model.Cluster) bson.ObjectID { return c.ID },
		setID:  func(c *model.Cluster, id bson.ObjectID) { c.ID = id },
		unique: func(c model.Cluster) []uniqueField { return []uniqueField{{"nume", c.Nume}} },
		keepLinks: func(next *model.Cluster, prev model.Cluster) {
			next.IDComp, next.IDUsers = prev.IDComp, prev.IDUsers
		},
	})}
}

func (r *clusterRepository) AddComputer(_ context.Context, clusterID, computerID bson.ObjectID) (bool, error) {
	return r.update(clusterID, func(c *model.Cluster) bool { return addTo(&c.IDComp, computerID) })
}

func (r *clusterRepository) RemoveComputer(_ context.Context, clusterID, computerID bson.ObjectID) (bool, error) {
	return r.update(clusterID, func(c *model.Cluster) bool { return removeFrom(&c.IDComp, computerID) })
}

func (r *clusterRepository) RemoveComputerEverywhere(_ context.Context, computerID bson.ObjectID) error {
	r.updateAll(func(c *model.Cluster) bool { return removeFrom(&c.IDComp, computerID) })
	return nil
}

func (r *clusterRepository) AddUser(_ context.Context, clusterID, userID bson.ObjectID) (bool, error) {
	return r.update(clusterID, func(c *model.Cluster) bool { return addTo(&c.IDUsers, userID) })
}

func (r *clusterRepository) RemoveUser(_ context.Context, clusterID, userID bson.ObjectID) (bool, error) {
	return r.update(clusterID, func(c *model.Cluster) bool { return removeFrom(&c.IDUsers, userID) })
}

func (r *clusterRepository) RemoveUserEverywhere(_ context.Context, userID bson.ObjectID) error {
	r.updateAll(func(c *model.Cluster) bool { return removeFrom(&c.IDUsers, userID) })
	return nil
}

func (r *clusterRepository) FindByComputer(_ context.Context, computerID bson.ObjectID) (model.Cluster, error) {
	return r.find(func(c model.Cluster) bool { return slices.Contains(c.IDComp, computerID) })
}

// ---------- utilizatori ----------

type userRepository struct {
	*collection[model.User]
}

func NewUserRepository() repository.UserRepository {
	return &userRepository{newCollection(spec[model.User]{
		idOf:  func(u model.User) bson.ObjectID { return u.ID },
		setID: func(u *model.User, id bson.ObjectID) { u.ID = id },
		// Acelasi cont din A&A Services nu poate avea doua documente.
		unique: func(u model.User) []uniqueField { return []uniqueField{{"id_user_grpc", u.IDUserGRPC}} },
		keepLinks: func(next *model.User, prev model.User) {
			next.IDCompClust = prev.IDCompClust
		},
	})}
}

func (r *userRepository) FindByGRPCID(_ context.Context, uid string) (model.User, error) {
	return r.find(func(u model.User) bool { return u.IDUserGRPC == uid })
}

func (r *userRepository) AddManaged(_ context.Context, userID bson.ObjectID, ref model.ManagedRef) (bool, error) {
	return r.update(userID, func(u *model.User) bool { return addTo(&u.IDCompClust, ref) })
}

func (r *userRepository) RemoveManaged(_ context.Context, userID bson.ObjectID, ref model.ManagedRef) (bool, error) {
	return r.update(userID, func(u *model.User) bool { return removeFrom(&u.IDCompClust, ref) })
}

func (r *userRepository) RemoveManagedEverywhere(_ context.Context, ref model.ManagedRef) error {
	r.updateAll(func(u *model.User) bool { return removeFrom(&u.IDCompClust, ref) })
	return nil
}

// ---------- task-uri ----------

type taskRepository struct {
	*collection[model.Task]
}

func NewTaskRepository() repository.TaskRepository {
	return &taskRepository{newCollection(spec[model.Task]{
		idOf:  func(t model.Task) bson.ObjectID { return t.ID },
		setID: func(t *model.Task, id bson.ObjectID) { t.ID = id },
	})}
}

func (r *taskRepository) ListByUser(_ context.Context, userID bson.ObjectID, offset, limit int) ([]model.Task, int, error) {
	tasks, total := r.listWhere(func(t model.Task) bool { return t.IDUser == userID }, offset, limit)
	return tasks, total, nil
}
