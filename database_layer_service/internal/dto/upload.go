package dto

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
)

// MaxScriptBytes: un fisier de script poate avea cel mult 1 MiB (peste: 413). Un script text
// are de obicei cativa KB, deci limita e generoasa, dar impiedica umplerea discului.
const MaxScriptBytes = 1 << 20

// Campurile acceptate in formularul de upload; orice alt camp da 415, ca la JSON.
var uploadFields = []string{"name", "store_comp", "is_active", "file"}

// scriptFileName: litere, cifre, ".", "_" si "-", maxim 100 de caractere, fara "." la inceput.
var scriptFileName = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._-]{0,99}$`)

// ScriptUpload este formularul multipart/form-data pentru POST /scripts (upload, L3 2.2).
// Clientul trimite datele scriptului si fisierul; store_path il alege serverul (unde salveaza).
// Valorile raman text, cum vin din formular; Validate le verifica, ToModel le converteste.
type ScriptUpload struct {
	Name      *string
	StoreComp *string
	IsActive  *string // "true" sau "false"
	FileName  string
	Content   []byte
	HasFile   bool
	Unknown   []string // campuri care nu exista in aceasta resursa
}

func (u *ScriptUpload) Validate(v *httpx.Validation) {
	for _, field := range u.Unknown {
		v.NotConform(field, "campul nu exista in aceasta resursa")
	}
	requireText(v, "name", u.Name)
	requireAbsolutePath(v, "store_comp", u.StoreComp)
	if u.IsActive != nil && *u.IsActive != "true" && *u.IsActive != "false" {
		v.NotConform("is_active", "trebuie sa fie true sau false")
	}

	switch {
	case !u.HasFile:
		v.Missing("file")
	case !isText(u.Content):
		// L3 2.2: "incarcarea fisierelor text". Un fisier binar nu are formatul cerut: 415.
		v.NotConform("file", "trebuie sa fie un fisier text (UTF-8)")
	case len(u.Content) == 0:
		v.Invalid("file", "fisierul este gol")
	case !scriptFileName.MatchString(u.FileName):
		v.Invalid("file", "numele fisierului poate contine doar litere, cifre, '.', '_' si '-'")
	}
}

// isText: UTF-8 valid si fara octetul 0, care apare in fisierele binare (executabile, imagini).
func isText(content []byte) bool {
	return utf8.Valid(content) && !bytes.Contains(content, []byte{0})
}

// IsUploadField spune daca un camp din formular face parte din resursa.
func IsUploadField(name string) bool {
	return slices.Contains(uploadFields, name)
}

// ToModel construieste scriptul; storePath e calea la care serverul a salvat fisierul.
func (u *ScriptUpload) ToModel(storePath string) model.Script {
	return model.Script{
		Name:      strings.TrimSpace(*u.Name),
		StorePath: storePath,
		StoreComp: *u.StoreComp,
		IsActive:  u.IsActive != nil && *u.IsActive == "true",
	}
}
