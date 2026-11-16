// Package files salveaza fisierele scripturilor pe disc. In Docker, folderul e volumul partajat
// cu modulul care ruleaza task-urile (SP 1.3: fisierele "sunt stocate pe un volum Docker partajat").
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ScriptStore pastreaza fisierele in dir/<id-script>/<nume-fisier>. Un folder pentru fiecare
// script evita conflictele cand doua scripturi au fisiere cu acelasi nume.
type ScriptStore struct {
	dir string // cale absoluta
}

// NewScriptStore pregateste folderul (il creeaza daca lipseste). Calea devine absoluta, pentru
// ca store_path trebuie sa fie o cale absoluta (SP, Tabelul 5).
func NewScriptStore(dir string) (*ScriptStore, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("folderul scripturilor %s: %w", abs, err)
	}
	return &ScriptStore{dir: abs}, nil
}

// Save scrie fisierul scriptului si intoarce calea lui absoluta (store_path).
// name trebuie validat inainte (dto.ScriptUpload); aici folosim oricum doar ultima lui parte.
func (s *ScriptStore) Save(id bson.ObjectID, name string, content []byte) (string, error) {
	folder := filepath.Join(s.dir, id.Hex())
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(folder, filepath.Base(name))
	// O_EXCL: nu suprascriem niciodata un fisier existent.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	return path, file.Close()
}

// Remove sterge fisierul si folderul lui, dar doar daca e in folderul gestionat de noi:
// un script adaugat prin JSON poate avea store_path oriunde, iar acela nu e al nostru.
func (s *ScriptStore) Remove(storePath string) error {
	if !s.Owns(storePath) {
		return nil
	}
	return os.RemoveAll(filepath.Dir(storePath))
}

// Owns spune daca fisierul e in folderul gestionat: dir/<id>/<fisier>.
func (s *ScriptStore) Owns(storePath string) bool {
	rel, err := filepath.Rel(s.dir, storePath)
	return err == nil && !strings.HasPrefix(rel, "..") && strings.Count(rel, string(filepath.Separator)) == 1
}

// Dir intoarce folderul scripturilor (cale absoluta).
func (s *ScriptStore) Dir() string {
	return s.dir
}
