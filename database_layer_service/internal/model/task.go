package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Task este documentul din colectia "tasks" (SP, Tabelul 6): o rulare a unui script.
// Campurile noi (SP permite: "puteti propune scheme noi") sunt cele cerute de executorul
// din student-kit: target (pe ce ruleaza), message si results (scrise de executor).
type Task struct {
	ID       bson.ObjectID `bson:"_id"`
	IDUser   bson.ObjectID `bson:"id_user"`   // documentul din /users al maintainerului
	IDScript bson.ObjectID `bson:"id_script"` // scriptul rulat
	// Target: calculatorul sau clusterul pe care ruleaza, in forma {tip, id} din id_comp_clust
	// (SP, Tabelul 2). Executorul citeste acest camp ca sa stie unde sa ruleze scriptul.
	Target ManagedRef `bson:"target"`
	Date   time.Time  `bson:"date"`  // ultima schimbare a starii
	State  string     `bson:"state"` // pending | active | finished
	// Status, Message si Results le scrie executorul la final: status e succeeded, failed,
	// partial sau error; results are cate o intrare pentru fiecare calculator.
	Status  string       `bson:"status,omitempty"`
	Message string       `bson:"message,omitempty"`
	Results []TaskResult `bson:"results,omitempty"`
	// Notification: daca ultima schimbare de stare a ajuns la utilizator (Job Status Service).
	Notification bool `bson:"notification"`
}

// TaskResult este rezultatul rularii pe un calculator, asa cum il scrie executorul din kit.
type TaskResult struct {
	IDComp          bson.ObjectID `bson:"id_comp"`
	IP              string        `bson:"IP"`
	Hostname        string        `bson:"hostname"`
	Status          string        `bson:"status"`    // succeeded, failed, timed_out, output_limit_exceeded, error
	ExitCode        *int          `bson:"exit_code"` // nil daca scriptul a fost oprit (timeout)
	Stdout          string        `bson:"stdout"`
	Stderr          string        `bson:"stderr"`
	OutputTruncated bool          `bson:"output_truncated"`
	Started         *time.Time    `bson:"started"`
	Finished        *time.Time    `bson:"finished"`
}

// Starile unui task (SP, Tabelul 6).
const (
	TaskPending  = "pending"
	TaskActive   = "active"
	TaskFinished = "finished"
)
