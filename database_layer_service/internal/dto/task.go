package dto

import (
	"time"

	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
)

// Task este reprezentarea unei rulari de script (SP, Tabelul 6) trimisa clientului.
type Task struct {
	ID           string       `json:"id"`
	IDUser       string       `json:"id_user"`
	IDScript     string       `json:"id_script"`
	Target       ManagedRef   `json:"target"`
	Date         time.Time    `json:"date"`
	State        string       `json:"state"`
	Status       string       `json:"status,omitempty"`
	Message      string       `json:"message,omitempty"`
	Results      []TaskResult `json:"results,omitempty"`
	Notification bool         `json:"notification"`
	Links        httpx.Links  `json:"_links"`
}

// TaskResult este rezultatul pe un calculator (scris de executor).
type TaskResult struct {
	IDComp          string     `json:"id_comp"`
	IP              string     `json:"IP"`
	Hostname        string     `json:"hostname,omitempty"`
	Status          string     `json:"status"`
	ExitCode        *int       `json:"exit_code"`
	Stdout          string     `json:"stdout"`
	Stderr          string     `json:"stderr"`
	OutputTruncated bool       `json:"output_truncated"`
	Started         *time.Time `json:"started,omitempty"`
	Finished        *time.Time `json:"finished,omitempty"`
}

// NewTask construieste reprezentarea; link-urile le alege controller-ul.
func NewTask(task model.Task, links httpx.Links) Task {
	representation := Task{
		ID:           task.ID.Hex(),
		IDUser:       task.IDUser.Hex(),
		IDScript:     task.IDScript.Hex(),
		Target:       ManagedRef{ID: task.Target.ID.Hex(), Tip: task.Target.Tip},
		Date:         task.Date,
		State:        task.State,
		Status:       task.Status,
		Message:      task.Message,
		Notification: task.Notification,
		Links:        links,
	}
	if representation.State == "" {
		representation.State = model.TaskPending // SP, Tabelul 6: starea lipsa inseamna pending
	}
	for _, r := range task.Results {
		representation.Results = append(representation.Results, TaskResult{
			IDComp: r.IDComp.Hex(), IP: r.IP, Hostname: r.Hostname, Status: r.Status,
			ExitCode: r.ExitCode, Stdout: r.Stdout, Stderr: r.Stderr,
			OutputTruncated: r.OutputTruncated, Started: r.Started, Finished: r.Finished,
		})
	}
	return representation
}
