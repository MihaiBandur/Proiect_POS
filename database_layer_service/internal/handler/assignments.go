package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
	"job-management-service/internal/service"
)

// Asignarea maintainerilor pe calculatoare si clustere (SP 1.3), doar pentru admin:
//
//	GET/PUT/DELETE /computers/:id/users[/:userId]
//	GET/PUT/DELETE /clusters/:id/users[/:userId]
//
// Rutele sunt aceleasi pentru calculatoare si clustere; assignmentHandler primeste
// operatiile din service potrivite pentru una sau alta.
type assignmentHandler struct {
	ownerPath string // computersPath sau clustersPath
	list      func(ctx context.Context, ownerID bson.ObjectID, offset, limit int) ([]model.User, int, error)
	get       func(ctx context.Context, ownerID, userID bson.ObjectID) (model.User, error)
	assign    func(ctx context.Context, token string, ownerID, userID bson.ObjectID) (bool, error)
	unassign  func(ctx context.Context, ownerID, userID bson.ObjectID) error
}

func computerAssignments(svc *service.Service) *assignmentHandler {
	return &assignmentHandler{
		ownerPath: computersPath,
		list:      svc.ComputerUsers,
		get:       svc.ComputerUser,
		assign:    svc.AssignToComputer,
		unassign:  svc.UnassignFromComputer,
	}
}

func clusterAssignments(svc *service.Service) *assignmentHandler {
	return &assignmentHandler{
		ownerPath: clustersPath,
		list:      svc.ClusterUsers,
		get:       svc.ClusterUser,
		assign:    svc.AssignToCluster,
		unassign:  svc.UnassignFromCluster,
	}
}

func (h *assignmentHandler) subPath(c *gin.Context) string {
	return h.ownerPath + "/" + c.Param("id") + "/users"
}

func (h *assignmentHandler) listUsers(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	users, total, err := h.list(c.Request.Context(), id, page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	identity := auth.FromContext(c)
	subPath := h.subPath(c)
	items := make([]dto.User, 0, len(users))
	for _, user := range users {
		items = append(items, dto.NewUser(user, relationLinks(subPath, user.ID, "user", usersPath, identity)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addSubCollectionLinks(body.Links, h.ownerPath+"/"+id.Hex(), subPath, "userId", identity)
	c.JSON(http.StatusOK, body)
}

func (h *assignmentHandler) getUser(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	userID, ok := httpx.ParseID(c, "userId")
	if !ok {
		return
	}
	user, err := h.get(c.Request.Context(), id, userID)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUser(user, relationLinks(h.subPath(c), userID, "user", usersPath, auth.FromContext(c))))
}

// assignUser: PUT .../users/:userId -> 201 sau 204; 404 daca lipseste ceva;
// 409 daca utilizatorul nu are rolul de maintainer in A&A Services; 503 daca A&A e oprit.
func (h *assignmentHandler) assignUser(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	userID, ok := httpx.ParseID(c, "userId")
	if !ok {
		return
	}
	created, err := h.assign(c.Request.Context(), auth.TokenFromContext(c), id, userID)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	user, err := h.get(c.Request.Context(), id, userID)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewUser(user, relationLinks(h.subPath(c), userID, "user", usersPath, auth.FromContext(c))))
}

// unassignUser: DELETE .../users/:userId -> 204 (SP 1.3: "revoca utilizatori din rolul de maintainer").
func (h *assignmentHandler) unassignUser(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	userID, ok := httpx.ParseID(c, "userId")
	if !ok {
		return
	}
	if err := h.unassign(c.Request.Context(), id, userID); err != nil {
		abortError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
