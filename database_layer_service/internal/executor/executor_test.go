package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// fakeExecutor asculta pe un socket Unix si raspunde ca executorul din student-kit.
// Pe macOS, calea unui socket Unix are cel mult ~104 caractere, deci folosim un folder scurt.
func fakeExecutor(t *testing.T) (path string, received chan map[string]string) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "ipc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path = filepath.Join(dir, "executor.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	received = make(chan map[string]string, 10)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			line, _ := bufio.NewReader(conn).ReadBytes('\n')
			var message map[string]string
			json.Unmarshal(line, &message)
			received <- message
			answer := map[string]any{"status": "accepted", "job_id": message["job_id"]}
			switch {
			case message["type"] == "health":
				answer = map[string]any{"status": "ready", "workers": 3, "queued": 0}
			case message["job_id"] == "inexistent":
				answer = map[string]any{"status": "rejected", "error": "job not found"}
			}
			out, _ := json.Marshal(answer)
			conn.Write(append(out, '\n'))
			conn.Close()
		}
	}()
	return path, received
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		socket   string
		url      string
		wantMock bool
	}{
		{"configuratie completa", "/run/lab-manager/ipc/executor.sock", "http://job-status:8081", false},
		{"fara IPC_SOCKET_PATH", "", "http://job-status:8081", true},
		{"fara JOB_STATUS_URL", "/run/lab-manager/ipc/executor.sock", "", true},
		{"JOB_STATUS_URL relativ", "/run/lab-manager/ipc/executor.sock", "job-status", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher, reason := New(tt.socket, tt.url)
			if dispatcher.Mock() != tt.wantMock || (tt.wantMock && reason == "") {
				t.Errorf("Mock = %v (%q), vrem %v", dispatcher.Mock(), reason, tt.wantMock)
			}
		})
	}
}

func TestSocketClient(t *testing.T) {
	path, received := fakeExecutor(t)
	dispatcher, _ := New(path, "http://job-status:8081")
	ctx := context.Background()

	if err := dispatcher.Health(ctx); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if got := <-received; got["type"] != "health" {
		t.Errorf("cererea health = %v", got)
	}

	// run acceptat: pe socket pleaca exact {"type": "run", "job_id": "..."}.
	if err := dispatcher.Run(ctx, "6650f1c2a1b2c3d4e5f60718"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := <-received; got["type"] != "run" || got["job_id"] != "6650f1c2a1b2c3d4e5f60718" {
		t.Errorf("cererea run = %v", got)
	}

	// run refuzat de executor.
	if err := dispatcher.Run(ctx, "inexistent"); !errors.Is(err, ErrRejected) {
		t.Errorf("Run refuzat: %v, vrem ErrRejected", err)
	}

	// Socket inexistent: executorul nu ruleaza.
	missing, _ := New(filepath.Join(filepath.Dir(path), "lipsa.sock"), "http://job-status:8081")
	if err := missing.Run(ctx, "6650f1c2a1b2c3d4e5f60718"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Run fara executor: %v, vrem ErrUnavailable", err)
	}
}
