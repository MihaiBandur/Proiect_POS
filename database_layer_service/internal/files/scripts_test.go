package files

import (
	"os"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestScriptStore(t *testing.T) {
	store, err := NewScriptStore(t.TempDir()) // folder temporar, sters automat dupa test
	if err != nil {
		t.Fatal(err)
	}
	id := bson.NewObjectID()

	path, err := store.Save(id, "backup.sh", []byte("#!/bin/sh\necho ok\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(path) || filepath.Base(filepath.Dir(path)) != id.Hex() {
		t.Errorf("store_path = %q, vrem <dir>/%s/backup.sh", path, id.Hex())
	}
	if content, _ := os.ReadFile(path); string(content) != "#!/bin/sh\necho ok\n" {
		t.Errorf("continut = %q", content)
	}

	// Acelasi script, acelasi nume: nu suprascriem.
	if _, err := store.Save(id, "backup.sh", []byte("altceva")); err == nil {
		t.Error("al doilea Save a suprascris fisierul")
	}

	// Remove sterge doar fisiere din folderul gestionat.
	outside := filepath.Join(t.TempDir(), "strain.sh")
	os.WriteFile(outside, []byte("x"), 0o644)
	if err := store.Remove(outside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("Remove a sters un fisier din afara folderului scripturilor")
	}
	if err := store.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Error("folderul scriptului nu a fost sters")
	}
}
