package model

import "go.mongodb.org/mongo-driver/v2/bson"

// Computer este documentul din colectia "computers" (SP, Tabelul 3).
type Computer struct {
	ID bson.ObjectID `bson:"_id"`
	IP string        `bson:"IP"`
	// Hostname poate fi omis (SP). omitempty: un hostname gol nu se scrie deloc in MongoDB,
	// iar indexul unic partial din Faza 5 il va ignora.
	Hostname string `bson:"hostname,omitempty"`
	// Legaturile se modifica doar prin rutele de legatura (Faza 4), nu prin PUT pe calculator.
	IDScript []bson.ObjectID `bson:"id_script"` // scripturile instalate
	IDUsers  []bson.ObjectID `bson:"id_users"`  // maintainerii asignati
}
