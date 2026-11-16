package handler

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/httpx"
)

// Spatiu in plus fata de fisier pentru restul formularului (campuri, delimitatori, header-e).
const multipartOverhead = 64 << 10

// upload: POST /scripts cu multipart/form-data (L3 2.2: "incarcarea fisierelor text").
// Fisierul se salveaza in folderul scripturilor, iar store_path e calea lui pe server.
// 201 + Location; 413 daca fisierul depaseste 1 MiB; 415/422 dupa regulile din 4b.
func (h *scriptHandler) upload(c *gin.Context) {
	upload, ok := readScriptUpload(c)
	if !ok {
		return
	}

	// Id-ul il alegem inainte, pentru ca face parte din calea fisierului.
	id := bson.NewObjectID()
	storePath, err := h.files.Save(id, upload.FileName, upload.Content)
	if err != nil {
		log.Printf("salvarea scriptului: %v", err)
		httpx.AbortWithProblem(c, http.StatusInternalServerError, "Eroare interna a serverului.")
		return
	}
	script := upload.ToModel(storePath)
	script.ID = id
	if _, err := h.repo.Replace(c.Request.Context(), script); err != nil {
		h.files.Remove(storePath) // fara document, fisierul nu mai are rost
		abortError(c, err)
		return
	}

	links := itemLinks(scriptsPath, id, auth.FromContext(c))
	c.Header("Location", links["self"].Href)
	c.JSON(http.StatusCreated, dto.NewScript(script, links))
}

// readScriptUpload citeste formularul in dto.ScriptUpload si il valideaza.
// Daca cererea nu e acceptabila, raspunde cu eroarea potrivita si intoarce false.
func readScriptUpload(c *gin.Context) (dto.ScriptUpload, bool) {
	var upload dto.ScriptUpload
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, dto.MaxScriptBytes+multipartOverhead)
	if err := c.Request.ParseMultipartForm(dto.MaxScriptBytes + multipartOverhead); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.AbortWithProblem(c, http.StatusRequestEntityTooLarge, "Fisierul depaseste limita de 1 MiB.")
		} else {
			httpx.AbortWithProblem(c, http.StatusBadRequest, "Formularul multipart nu poate fi citit.")
		}
		return upload, false
	}
	form := c.Request.MultipartForm
	defer form.RemoveAll() // sterge eventualele fisiere temporare

	for name, values := range form.Value {
		if !dto.IsUploadField(name) {
			upload.Unknown = append(upload.Unknown, name)
			continue
		}
		value := values[0]
		switch name {
		case "name":
			upload.Name = &value
		case "store_comp":
			upload.StoreComp = &value
		case "is_active":
			upload.IsActive = &value
		}
	}
	for name := range form.File {
		if name != "file" {
			upload.Unknown = append(upload.Unknown, name)
		}
	}

	if headers := form.File["file"]; len(headers) > 0 {
		header := headers[0]
		if header.Size > dto.MaxScriptBytes {
			httpx.AbortWithProblem(c, http.StatusRequestEntityTooLarge, "Fisierul depaseste limita de 1 MiB.")
			return upload, false
		}
		file, err := header.Open()
		if err != nil {
			httpx.AbortWithProblem(c, http.StatusBadRequest, "Fisierul nu poate fi citit.")
			return upload, false
		}
		defer file.Close()
		content := make([]byte, header.Size)
		if _, err := io.ReadFull(file, content); err != nil {
			httpx.AbortWithProblem(c, http.StatusBadRequest, "Fisierul nu poate fi citit.")
			return upload, false
		}
		upload.FileName, upload.Content, upload.HasFile = header.Filename, content, true
	}

	return upload, httpx.CheckValid(c, &upload)
}
