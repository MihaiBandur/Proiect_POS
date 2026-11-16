package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const shellScript = "#!/bin/sh\necho \"backup\"\n"

// form descrie un formular de upload: campurile text si (optional) fisierul.
type form struct {
	fields   map[string]string
	fileName string // "" = fara fisier
	content  []byte
}

// sendForm trimite formularul ca multipart/form-data, cum face un browser sau Bruno.
func sendForm(router *gin.Engine, token string, f form) *httptest.ResponseRecorder {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range f.fields {
		writer.WriteField(name, value)
	}
	if f.fileName != "" {
		part, _ := writer.CreateFormFile("file", f.fileName)
		part.Write(f.content)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, scriptsPath, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType()) // contine si delimitatorul (boundary)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func validForm() form {
	return form{
		fields:   map[string]string{"name": "backup", "store_comp": "/opt/scripts/backup.sh", "is_active": "true"},
		fileName: "backup.sh",
		content:  []byte(shellScript),
	}
}

func TestUploadScript(t *testing.T) {
	router := newTestRouter()

	// Upload reusit: 201, fisierul e pe disc, iar store_path il arata.
	rec := sendForm(router, adminToken, validForm())
	expectStatus(t, rec, http.StatusCreated)
	script := decode[struct {
		ID        string `json:"id"`
		StorePath string `json:"store_path"`
		IsActive  bool   `json:"is_active"`
	}](t, rec)
	if rec.Header().Get("Location") != scriptsPath+"/"+script.ID || !script.IsActive {
		t.Errorf("raspuns gresit: %+v, Location %q", script, rec.Header().Get("Location"))
	}
	if !strings.HasSuffix(script.StorePath, "/"+script.ID+"/backup.sh") {
		t.Errorf("store_path = %q, vrem .../<id>/backup.sh", script.StorePath)
	}
	if content, err := os.ReadFile(script.StorePath); err != nil || string(content) != shellScript {
		t.Errorf("fisierul de pe disc: %q, %v", content, err)
	}

	// Stergerea scriptului sterge si fisierul.
	expectStatus(t, send(router, http.MethodDelete, scriptsPath+"/"+script.ID, adminToken, ""), http.StatusOK)
	if _, err := os.Stat(script.StorePath); !os.IsNotExist(err) {
		t.Errorf("fisierul a ramas dupa DELETE: %v", err)
	}
}

func TestUploadErrors(t *testing.T) {
	router := newTestRouter()
	change := func(edit func(*form)) form {
		f := validForm()
		edit(&f)
		return f
	}

	tests := []struct {
		name       string
		token      string
		form       form
		wantStatus int
		wantField  string
	}{
		{"maintainerul nu poate incarca", maintainerToken, validForm(), http.StatusForbidden, ""},
		{"fara fisier", adminToken, change(func(f *form) { f.fileName = "" }), http.StatusUnsupportedMediaType, "file"},
		{"fara nume", adminToken, change(func(f *form) { delete(f.fields, "name") }), http.StatusUnsupportedMediaType, "name"},
		{"camp necunoscut", adminToken, change(func(f *form) { f.fields["store_path"] = "/etc/passwd" }), http.StatusUnsupportedMediaType, "store_path"},
		{"is_active invalid", adminToken, change(func(f *form) { f.fields["is_active"] = "da" }), http.StatusUnsupportedMediaType, "is_active"},
		{"fisier binar", adminToken, change(func(f *form) { f.content = []byte{0x7f, 'E', 'L', 'F', 0, 1} }), http.StatusUnsupportedMediaType, "file"},
		{"fisier gol", adminToken, change(func(f *form) { f.content = nil }), http.StatusUnprocessableEntity, "file"},
		{"nume de fisier cu spatiu", adminToken, change(func(f *form) { f.fileName = "copie backup.sh" }), http.StatusUnprocessableEntity, "file"},
		{"store_comp relativ", adminToken, change(func(f *form) { f.fields["store_comp"] = "opt/b.sh" }), http.StatusUnprocessableEntity, "store_comp"},
		{"fisier prea mare", adminToken, change(func(f *form) { f.content = bytes.Repeat([]byte("a"), 1<<20+1) }), http.StatusRequestEntityTooLarge, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendForm(router, tt.token, tt.form)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantField == "" {
				return
			}
			body := decode[struct {
				Errors []struct {
					Field string `json:"field"`
				} `json:"errors"`
			}](t, rec)
			if len(body.Errors) == 0 || body.Errors[0].Field != tt.wantField {
				t.Errorf("errors = %+v, vrem primul camp %q", body.Errors, tt.wantField)
			}
		})
	}

	// Alt Content-Type decat JSON sau multipart: 415, iar Accept spune ce e acceptat.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, scriptsPath, strings.NewReader("echo"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(rec, req)
	expectStatus(t, rec, http.StatusUnsupportedMediaType)
	if got := rec.Header().Get("Accept"); got != "application/json, multipart/form-data" {
		t.Errorf("Accept = %q", got)
	}

	// Niciun upload respins nu a lasat fisiere pe disc.
	entries, _ := os.ReadDir(testScriptFilesDir())
	if len(entries) != 0 {
		t.Errorf("au ramas %d foldere de scripturi dupa uploadurile respinse", len(entries))
	}
}
