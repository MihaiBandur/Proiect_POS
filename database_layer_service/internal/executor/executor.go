// Package executor vorbeste cu executorul SSH din student-kit, care ruleaza scripturile pe
// calculatoare. Comunicarea e IPC (L3 2.2), printr-un socket Unix, dupa contractul din
// student-kit/README.md: o conexiune = o cerere si un raspuns, fiecare o linie JSON.
package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"
)

// Executorul accepta o linie in cel mult 2 s dupa conectare; ne incadram cu totul in 3 s.
const ipcTimeout = 3 * time.Second

var (
	// ErrUnavailable: socketul nu raspunde (executorul nu ruleaza inca sau a cazut).
	ErrUnavailable = errors.New("executorul nu este disponibil")
	// ErrRejected: executorul a refuzat cererea (ex. job inexistent sau care nu mai e pending).
	ErrRejected = errors.New("executorul a refuzat cererea")
)

// Dispatcher anunta executorul ca un job (task) asteapta sa fie rulat.
type Dispatcher interface {
	Run(ctx context.Context, jobID string) error
	// Health intreaba executorul daca e gata ({"type": "health"}).
	Health(ctx context.Context) error
	// Mock spune daca serviciul ruleaza fara executor; atunci task-urile sunt simulate.
	Mock() bool
}

// New verifica la pornire configuratia ceruta de SP 1.3: canalul IPC catre executor
// (IPC_SOCKET_PATH; SP vorbeste de un PIPE FIFO, kit-ul profesorului foloseste un socket Unix)
// si link-ul catre Job Status Service (JOB_STATUS_URL). Daca lipseste ceva, intoarce
// dispecerul mock si motivul ("in absenta lor, modulul curent va furniza raspunsuri mock").
func New(socketPath, jobStatusURL string) (Dispatcher, string) {
	if socketPath == "" {
		return mock{}, "IPC_SOCKET_PATH nu e setat"
	}
	if u, err := url.Parse(jobStatusURL); jobStatusURL == "" || err != nil || !u.IsAbs() {
		return mock{}, "JOB_STATUS_URL lipseste sau nu e o adresa absoluta"
	}
	return &socketClient{path: socketPath}, ""
}

// mock nu trimite nimic; serviciul raspunde cu task-uri simulate.
type mock struct{}

func (mock) Run(context.Context, string) error { return nil }
func (mock) Health(context.Context) error      { return nil }
func (mock) Mock() bool                        { return true }

// socketClient trimite cererile pe socketul Unix al executorului.
type socketClient struct {
	path string
}

func (s *socketClient) Mock() bool { return false }

// Run: {"type": "run", "job_id": "..."} -> {"status": "accepted", ...}
func (s *socketClient) Run(ctx context.Context, jobID string) error {
	reply, err := s.request(ctx, map[string]string{"type": "run", "job_id": jobID})
	if err != nil {
		return err
	}
	if reply.Status != "accepted" {
		return fmt.Errorf("%w: %s", ErrRejected, reply.Error)
	}
	return nil
}

// Health: {"type": "health"} -> {"status": "ready", ...}
func (s *socketClient) Health(ctx context.Context) error {
	reply, err := s.request(ctx, map[string]string{"type": "health"})
	if err != nil {
		return err
	}
	if reply.Status != "ready" {
		return fmt.Errorf("%w: %s", ErrRejected, reply.Error)
	}
	return nil
}

type reply struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// request deschide o conexiune, trimite o linie JSON si citeste linia de raspuns.
func (s *socketClient) request(ctx context.Context, message map[string]string) (reply, error) {
	ctx, cancel := context.WithTimeout(ctx, ipcTimeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", s.path)
	if err != nil {
		return reply{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	conn.SetDeadline(deadline)

	line, err := json.Marshal(message)
	if err != nil {
		return reply{}, err
	}
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return reply{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	answer, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return reply{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	var r reply
	if err := json.Unmarshal(answer, &r); err != nil {
		return reply{}, fmt.Errorf("%w: raspuns invalid %q", ErrUnavailable, answer)
	}
	return r, nil
}
