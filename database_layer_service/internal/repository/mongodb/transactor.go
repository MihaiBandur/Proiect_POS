package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"job-management-service/internal/repository"
)

type transactor struct {
	client *mongo.Client
}

func NewTransactor(client *mongo.Client) repository.Transactor {
	return &transactor{client: client}
}

func (t *transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := t.client.StartSession()
	if err != nil {
		return convertError(err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	})
	return convertError(err)
}
