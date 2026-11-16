package mongodb

import (
	"context"
	"job-management-service/internal/model"
	"job-management-service/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	scriptsCollection   = "scripts"
	usersCollection     = "users"
	computersCollection = "computers"
	clustersCollection  = "clusters"
	tasksCollection     = "tasks"
)

func CollectionNames() []string {
	return []string{scriptsCollection, usersCollection, computersCollection, clustersCollection, tasksCollection}
}

func addToSet(field string, value any) bson.D {
	return bson.D{{
		Key: "$addToSet",
		Value: bson.D{{
			Key:   field,
			Value: value,
		}}}}
}

func pull(field string, value any) bson.D {
	return bson.D{{
		Key: "$pull",
		Value: bson.D{{
			Key:   field,
			Value: value,
		}},
	}}
}

func contains(field string, value any) bson.D {
	return bson.D{{Key: field, Value: value}}
}

func NewScriptRepository(db *mongo.Database) repository.ScriptRepository {
	return newCollection(db.Collection(scriptsCollection), spec[model.Script]{
		idOf:  func(s model.Script) bson.ObjectID { return s.ID },
		setID: func(s *model.Script, id bson.ObjectID) { s.ID = id },
	})
}

type computerRepository struct {
	*collection[model.Computer]
}

func NewComputerRepository(db *mongo.Database) repository.ComputerRepository {
	return &computerRepository{newCollection(db.Collection(computersCollection), spec[model.Computer]{
		idOf:           func(c model.Computer) bson.ObjectID { return c.ID },
		setID:          func(c *model.Computer, id bson.ObjectID) { c.ID = id },
		linkFields:     []string{"id_script", "id_users"},
		optionalFields: []string{"hostname"},
	})}
}

func (r *computerRepository) AddScript(ctx context.Context, computerID, scriptID bson.ObjectID) (bool, error) {
	return r.update(ctx, computerID, addToSet("id_script", scriptID))
}

func (r *computerRepository) RemoveScript(ctx context.Context, computerID, scriptID bson.ObjectID) (bool, error) {
	return r.update(ctx, computerID, pull("id_script", scriptID))
}

func (r *computerRepository) RemoveScriptEverywhere(ctx context.Context, scriptID bson.ObjectID) error {
	return r.updateAll(ctx, contains("id_script", scriptID), pull("id_script", scriptID))
}

func (r *computerRepository) AddUser(ctx context.Context, computerID, userID bson.ObjectID) (bool, error) {
	return r.update(ctx, computerID, addToSet("id_users", userID))
}

func (r *computerRepository) RemoveUser(ctx context.Context, computerID, userID bson.ObjectID) (bool, error) {
	return r.update(ctx, computerID, pull("id_users", userID))
}

func (r *computerRepository) RemoveUserEverywhere(ctx context.Context, userID bson.ObjectID) error {
	return r.updateAll(ctx, contains("id_users", userID), pull("id_users", userID))
}

type clusterRepository struct {
	*collection[model.Cluster]
}

func NewClusterRepository(db *mongo.Database) repository.ClusterRepository {
	return &clusterRepository{newCollection(db.Collection(clustersCollection), spec[model.Cluster]{
		idOf:       func(c model.Cluster) bson.ObjectID { return c.ID },
		setID:      func(c *model.Cluster, id bson.ObjectID) { c.ID = id },
		linkFields: []string{"id_comp", "id_users"},
	})}
}

func (r *clusterRepository) AddComputer(ctx context.Context, clusterID, computerID bson.ObjectID) (bool, error) {
	return r.update(ctx, clusterID, addToSet("id_comp", computerID))
}

func (r *clusterRepository) RemoveComputer(ctx context.Context, clusterID, computerID bson.ObjectID) (bool, error) {
	return r.update(ctx, clusterID, pull("id_comp", computerID))
}

func (r *clusterRepository) RemoveComputerEverywhere(ctx context.Context, computerID bson.ObjectID) error {
	return r.updateAll(ctx, contains("id_comp", computerID), pull("id_comp", computerID))
}

func (r *clusterRepository) AddUser(ctx context.Context, clusterID, userID bson.ObjectID) (bool, error) {
	return r.update(ctx, clusterID, addToSet("id_users", userID))
}

func (r *clusterRepository) RemoveUser(ctx context.Context, clusterID, userID bson.ObjectID) (bool, error) {
	return r.update(ctx, clusterID, pull("id_users", userID))
}

func (r *clusterRepository) RemoveUserEverywhere(ctx context.Context, userID bson.ObjectID) error {
	return r.updateAll(ctx, contains("id_users", userID), pull("id_users", userID))
}

func (r *clusterRepository) FindByComputer(ctx context.Context, computerID bson.ObjectID) (model.Cluster, error) {
	return r.findOne(ctx, contains("id_comp", computerID))
}

type userRepository struct {
	*collection[model.User]
}

func NewUserRepository(db *mongo.Database) repository.UserRepository {
	return &userRepository{newCollection(db.Collection(usersCollection), spec[model.User]{
		idOf:       func(u model.User) bson.ObjectID { return u.ID },
		setID:      func(u *model.User, id bson.ObjectID) { u.ID = id },
		linkFields: []string{"id_comp_clust"},
	})}
}

func (r *userRepository) FindByGRPCID(ctx context.Context, uid string) (model.User, error) {
	return r.findOne(ctx, bson.D{{Key: "id_user_grpc", Value: uid}})
}

func (r *userRepository) AddManaged(ctx context.Context, userID bson.ObjectID, ref model.ManagedRef) (bool, error) {
	return r.update(ctx, userID, addToSet("id_comp_clust", ref))
}

func (r *userRepository) RemoveManaged(ctx context.Context, userID bson.ObjectID, ref model.ManagedRef) (bool, error) {
	return r.update(ctx, userID, pull("id_comp_clust", ref))
}

func (r *userRepository) RemoveManagedEverywhere(ctx context.Context, ref model.ManagedRef) error {
	filter := bson.D{{Key: "id_comp_clust", Value: ref}}
	return r.updateAll(ctx, filter, pull("id_comp_clust", ref))
}

// ---------- task-uri ----------

type taskRepository struct {
	*collection[model.Task]
}

func NewTaskRepository(db *mongo.Database) repository.TaskRepository {
	return &taskRepository{newCollection(db.Collection(tasksCollection), spec[model.Task]{
		idOf:  func(t model.Task) bson.ObjectID { return t.ID },
		setID: func(t *model.Task, id bson.ObjectID) { t.ID = id },
	})}
}

// ListByUser foloseste indexul id_user din 02-schema.js.
func (r *taskRepository) ListByUser(ctx context.Context, userID bson.ObjectID, offset, limit int) ([]model.Task, int, error) {
	return r.listWhere(ctx, bson.D{{Key: "id_user", Value: userID}}, offset, limit)
}
