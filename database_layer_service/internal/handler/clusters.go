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

const clustersPath = basePath + "/clusters"

// clusterHandler este controller-ul pentru /clusters si sub-colectiile lui (L3 2.1, SP 1.3).
type clusterHandler struct {
	repo repository.ClusterRepository
	svc  *service.Service
}

// list: adminul vede toate clusterele; maintainerul doar pe cele repartizate lui (L3 2.1).
func (h *clusterHandler) list(c *gin.Context) {
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	clusters, total, err := h.svc.ListClusters(c.Request.Context(), identity, page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	items := make([]dto.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		items = append(items, dto.NewCluster(cluster, clusterLinks(cluster.ID, identity)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addCollectionLinks(body.Links, clustersPath, identity)
	c.JSON(http.StatusOK, body)
}

// get: 200; 404 daca nu exista; 403 daca maintainerul nu il are asignat.
func (h *clusterHandler) get(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	identity := auth.FromContext(c)
	cluster, err := h.svc.GetCluster(c.Request.Context(), identity, id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewCluster(cluster, clusterLinks(cluster.ID, identity)))
}

// create: POST /clusters -> 201 + Location; 409 daca numele exista deja.
func (h *clusterHandler) create(c *gin.Context) {
	var req dto.ClusterRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	cluster, err := h.repo.Create(c.Request.Context(), req.ToModel())
	if err != nil {
		abortError(c, err)
		return
	}
	links := clusterLinks(cluster.ID, auth.FromContext(c))
	c.Header("Location", links["self"].Href)
	c.JSON(http.StatusCreated, dto.NewCluster(cluster, links))
}

// replace: PUT /clusters/:id -> 204 sau 201. Inlocuieste numele; calculatoarele si
// maintainerii asignati raman.
func (h *clusterHandler) replace(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.ClusterRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	cluster := req.ToModel()
	cluster.ID = id
	created, err := h.repo.Replace(c.Request.Context(), cluster)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusCreated, dto.NewCluster(cluster, clusterLinks(cluster.ID, auth.FromContext(c))))
}

// delete: 200 cu ultima reprezentare; clusterul dispare si din lista maintainerilor.
// Calculatoarele din el raman, doar ca nu mai fac parte din niciun cluster.
func (h *clusterHandler) delete(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	cluster, err := h.svc.DeleteCluster(c.Request.Context(), id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewCluster(cluster, deletedLinks(clustersPath)))
}
