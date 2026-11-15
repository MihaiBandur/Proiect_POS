package model

import "go.mongodb.org/mongo-driver/v2/bson"

// Cluster este documentul din colectia "clusters" (SP, Tabelul 4).
type Cluster struct {
	ID   bson.ObjectID `bson:"_id"`
	Nume string        `bson:"nume"`
	// Legaturile se modifica doar prin rutele de legatura (Faza 4), nu prin PUT pe cluster.
	IDComp  []bson.ObjectID `bson:"id_comp"`  // calculatoarele din cluster
	IDUsers []bson.ObjectID `bson:"id_users"` // maintainerii asignati
}
