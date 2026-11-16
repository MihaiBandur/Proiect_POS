package mongodb

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"job-management-service/internal/repository"
)

type spec[T any] struct {
	idOf           func(T) bson.ObjectID
	setID          func(*T, bson.ObjectID)
	linkFields     []string
	optionalFields []string
}

type collection[T any] struct {
	coll *mongo.Collection
	spec spec[T]
}

func newCollection[T any](coll *mongo.Collection, s spec[T]) *collection[T] {
	return &collection[T]{coll: coll, spec: s}
}

func byID(id bson.ObjectID) bson.D {
	return bson.D{{
		Key:   "_id",
		Value: id,
	}}
}

func (c *collection[T]) List(ctx context.Context, offset, limit int) ([]T, int, error) {
	return c.listWhere(ctx, bson.D{}, offset, limit)
}

// listWhere: o pagina din documentele care se potrivesc cu filter, sortata dupa _id,
// plus numarul lor total.
func (c *collection[T]) listWhere(ctx context.Context, filter bson.D, offset, limit int) ([]T, int, error) {
	total, err := c.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, convertError(err)
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: 1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))
	docs, err := c.findAll(ctx, filter, opts)
	return docs, int(total), err
}

func (c *collection[T]) Get(ctx context.Context, id bson.ObjectID) (T, error) {
	return c.findOne(ctx, byID(id))
}

func (c *collection[T]) GetMany(ctx context.Context, ids []bson.ObjectID) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}}
	return c.findAll(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
}

func (c *collection[T]) Create(ctx context.Context, doc T) (T, error) {
	c.spec.setID(&doc, bson.NewObjectID())
	if _, err := c.coll.InsertOne(ctx, doc); err != nil {
		var zero T
		return zero, convertError(err)
	}
	return doc, nil
}

func (c *collection[T]) Replace(ctx context.Context, doc T) (bool, error) {
	update, err := c.replaceUpdate(doc)
	if err != nil {
		return false, err
	}
	result, err := c.coll.UpdateOne(ctx, byID(c.spec.idOf(doc)), update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, convertError(err)
	}
	return result.UpsertedCount == 1, nil
}

func (c *collection[T]) replaceUpdate(doc T) (bson.D, error) {
	raw, err := bson.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var fields bson.M
	if err := bson.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	delete(fields, "_id")
	for _, name := range c.spec.linkFields {
		delete(fields, name)
	}
	update := bson.D{{Key: "$set", Value: fields}}

	unset := bson.D{}
	for _, name := range c.spec.optionalFields {
		if _, present := fields[name]; !present {
			unset = append(unset, bson.E{Key: name, Value: ""})
		}
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	return update, nil
}

func (c *collection[T]) Delete(ctx context.Context, id bson.ObjectID) (T, error) {
	var doc T
	err := c.coll.FindOneAndDelete(ctx, byID(id)).Decode(&doc)
	return doc, convertError(err)
}

func (c *collection[T]) update(ctx context.Context, id bson.ObjectID, update bson.D) (bool, error) {
	result, err := c.coll.UpdateOne(ctx, byID(id), update)
	if err != nil {
		return false, convertError(err)
	}
	if result.MatchedCount == 0 {
		return false, repository.ErrNotFound
	}
	return result.ModifiedCount == 1, nil
}

func (c *collection[T]) updateAll(ctx context.Context, filter, update bson.D) error {
	_, err := c.coll.UpdateMany(ctx, filter, update)
	return convertError(err)
}

func (c *collection[T]) findOne(ctx context.Context, filter bson.D) (T, error) {
	var doc T
	err := c.coll.FindOne(ctx, filter).Decode(&doc)
	return doc, convertError(err)
}

func (c *collection[T]) findAll(ctx context.Context, filter bson.D, opts *options.FindOptionsBuilder) ([]T, error) {
	cursor, err := c.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, convertError(err)
	}
	docs := []T{}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, convertError(err)
	}
	return docs, nil
}

var duplicateIndex = regexp.MustCompile(`index: (\S+) dup key`)

func convertError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, mongo.ErrNoDocuments):
		return repository.ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		field := "id"
		if match := duplicateIndex.FindStringSubmatch(err.Error()); match != nil {
			field = match[1]
		}
		return &repository.DuplicateError{Field: field}
	case mongo.IsTimeout(err), mongo.IsNetworkError(err):
		return fmt.Errorf("%w: %v", repository.ErrUnavailable, err)

	}
	return err
}
