package model

import "go.mongodb.org/mongo-driver/v2/bson"

// User este documentul din colectia "users" (SP, Tabelul 2).
type User struct {
	ID         bson.ObjectID `bson:"_id"`
	IDUserGRPC string        `bson:"id_user_grpc"` // uid-ul contului din A&A Services
	UserName   string        `bson:"userName"`     // copiat din A&A Services
	// IDCompClust: calculatoarele si clusterele pe care le gestioneaza utilizatorul.
	// Se modifica doar prin rutele de asignare (Faza 4).
	IDCompClust []ManagedRef `bson:"id_comp_clust"`
}

// ManagedRef este un element din id_comp_clust: SP cere forma {id: "...", tip: "calculator|cluster"}.
type ManagedRef struct {
	ID  bson.ObjectID `bson:"id"`
	Tip string        `bson:"tip"`
}

// Valorile permise pentru ManagedRef.Tip (SP, Tabelul 2).
const (
	TipCalculator = "calculator"
	TipCluster    = "cluster"
)
