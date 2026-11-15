package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/httpx"
	"job-management-service/internal/repository"
	"job-management-service/internal/service"
)

const computersPath = basePath + "/computers"

// computerHandler este controller-ul pentru /computers si sub-colectiile lui (L3 2.1, SP 1.3).
type computerHandler struct {
	repo repository.ComputerRepository
	svc  *service.Service
}

// list: adminul vede toate calculatoarele; maintainerul doar pe cele repartizate lui (L3 2.1).
func (h *computerHandler) list(c *gin.Context) {
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	computers, total, err := h.svc.ListComputers(c.Request.Context(), identity, page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	items := make([]dto.Computer, 0, len(computers))
	for _, computer := range computers {
		items = append(items, dto.NewComputer(computer, computerLinks(computer.ID, identity)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addCollectionLinks(body.Links, computersPath, identity)
	c.JSON(http.StatusOK, body)
}

// get: 200; 404 daca nu exista; 403 daca maintainerul nu il are asignat.
func (h *computerHandler) get(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	computer, err := h.svc.GetComputer(c.Request.Context(), identity, id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewComputer(computer, computerLinks(computer.ID, identity)))
}

// create: POST /computers -> 201 + Location; 409 daca IP-ul sau hostname-ul exista deja.
func (h *computerHandler) create(c *gin.Context) {
	var req dto.ComputerRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	computer, err := h.repo.Create(c.Request.Context(), req.ToModel())
	if err != nil {
		abortError(c, err)
		return
	}
	links := computerLinks(computer.ID, auth.FromContext(c))
	c.Header("Location", links["self"].Href)
	c.JSON(http.StatusCreated, dto.NewComputer(computer, links))
}

// replace: PUT /computers/:id -> 204 sau 201. Inlocuieste IP-ul si hostname-ul (un hostname
// omis se sterge); scripturile si maintainerii asignati raman.
func (h *computerHandler) replace(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.ComputerRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	computer := req.ToModel()
	computer.ID = id
	created, err := h.repo.Replace(c.Request.Context(), computer)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusCreated, dto.NewComputer(computer, computerLinks(computer.ID, auth.FromContext(c))))
}

// delete: 200 cu ultima reprezentare; calculatorul dispare si din clustere si de la maintaineri.
func (h *computerHandler) delete(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	computer, err := h.svc.DeleteComputer(c.Request.Context(), id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewComputer(computer, deletedLinks(computersPath)))
}
