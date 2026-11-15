package model

import "go.mongodb.org/mongo-driver/v2/bson"

type Script struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	StorePath string        `bson:"store_path"`
	StoreComp string        `bson:"store_comp"`
	IsActive  bool          `bson:"is_active"`
}
