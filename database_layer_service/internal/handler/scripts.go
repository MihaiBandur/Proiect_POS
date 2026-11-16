package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/files"
	"job-management-service/internal/httpx"
	"job-management-service/internal/repository"
	"job-management-service/internal/service"
)

const scriptsPath = basePath + "/scripts"

// scriptHandler este controller-ul pentru /scripts (L3 2.1).
type scriptHandler struct {
	repo  repository.ScriptRepository
	svc   *service.Service
	files *files.ScriptStore // fisierele incarcate prin upload (6.2)
}

// list: GET /scripts?page=&items_per_page= -> 200 (RFC 9110, 9.3.1).
func (h *scriptHandler) list(c *gin.Context) {
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	scripts, total, err := h.repo.List(c.Request.Context(), page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	identity := auth.FromContext(c)
	items := make([]dto.Script, 0, len(scripts))
	for _, script := range scripts {
		items = append(items, dto.NewScript(script, itemLinks(scriptsPath, script.ID, identity)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addCollectionLinks(body.Links, scriptsPath, identity)
	c.JSON(http.StatusOK, body)
}

// get: GET /scripts/:id -> 200 sau 404.
func (h *scriptHandler) get(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	script, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewScript(script, itemLinks(scriptsPath, script.ID, auth.FromContext(c))))
}

// create: POST /scripts -> 201 + Location + reprezentarea (RFC 9110, 9.3.3 si 15.3.2).
// Serverul alege id-ul, de aceea folosim POST.
func (h *scriptHandler) create(c *gin.Context) {
	// Acelasi POST primeste doua formate, dupa Content-Type (RFC 9110, 8.3): multipart/form-data
	// cu fisierul (upload, 6.2) sau JSON doar cu datele, pentru un fisier aflat deja pe server.
	switch httpx.MediaType(c) {
	case "multipart/form-data":
		h.upload(c)
		return
	case "application/json":
	default:
		c.Header("Accept", "application/json, multipart/form-data")
		httpx.AbortWithProblem(c, http.StatusUnsupportedMediaType,
			"Corpul cererii trebuie trimis ca application/json sau multipart/form-data.")
		return
	}

	var req dto.ScriptRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	script, err := h.repo.Create(c.Request.Context(), req.ToModel())
	if err != nil {
		abortError(c, err)
		return
	}
	links := itemLinks(scriptsPath, script.ID, auth.FromContext(c))
	c.Header("Location", links["self"].Href)
	c.JSON(http.StatusCreated, dto.NewScript(script, links))
}

// replace: PUT /scripts/:id -> inlocuire completa (RFC 9110, 9.3.4).
// 204 daca scriptul exista (L3 1.2), 201 + reprezentarea daca PUT l-a creat.
func (h *scriptHandler) replace(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.ScriptRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	script := req.ToModel()
	script.ID = id // la PUT, id-ul vine din adresa, nu din corp
	created, err := h.repo.Replace(c.Request.Context(), script)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusCreated, dto.NewScript(script, itemLinks(scriptsPath, script.ID, auth.FromContext(c))))
}

// delete: DELETE /scripts/:id -> 200 cu ultima reprezentare (L3 1.2, RFC 9110 9.3.5).
// Scriptul e scos si de pe calculatoarele pe care era instalat (stergere in cascada).
func (h *scriptHandler) delete(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	script, err := h.svc.DeleteScript(c.Request.Context(), id)
	if err != nil {
		abortError(c, err)
		return
	}
	// Fisierul incarcat prin upload dispare odata cu scriptul. Un fisier din afara folderului
	// nostru (script adaugat prin JSON) ramane neatins.
	if err := h.files.Remove(script.StorePath); err != nil {
		log.Printf("stergerea fisierului %s: %v", script.StorePath, err)
	}
	c.JSON(http.StatusOK, dto.NewScript(script, deletedLinks(scriptsPath)))
}
