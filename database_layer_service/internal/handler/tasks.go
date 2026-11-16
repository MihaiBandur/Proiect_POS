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

const tasksPath = basePath + "/tasks"

// taskHandler este controller-ul pentru rularea scripturilor (L3 2.2, SP 1.3).
type taskHandler struct {
	svc *service.Service
}

// taskLinks: task-ul, colectia, scriptul si tinta (calculatorul sau clusterul).
func taskLinks(task model.Task) httpx.Links {
	links := httpx.Links{
		"self":   {Href: tasksPath + "/" + task.ID.Hex()},
		"parent": {Href: tasksPath},
		"script": {Href: scriptsPath + "/" + task.IDScript.Hex()},
	}
	if task.Target.Tip == model.TipCluster {
		links["cluster"] = httpx.Link{Href: clustersPath + "/" + task.Target.ID.Hex()}
	} else {
		links["computer"] = httpx.Link{Href: computersPath + "/" + task.Target.ID.Hex()}
	}
	return links
}

// launchOnComputer: POST /computers/:id/scripts/:scriptId/tasks.
// Raspunsul e 202 Accepted (RFC 9110, 15.3.3): cererea a fost acceptata, dar rularea abia
// incepe. Location si self duc la task, care e "monitorul de stare" cerut de RFC.
func (h *taskHandler) launchOnComputer(c *gin.Context) {
	h.launch(c, h.svc.LaunchOnComputer)
}

// launchOnCluster: POST /clusters/:id/scripts/:scriptId/tasks (pe toate calculatoarele).
func (h *taskHandler) launchOnCluster(c *gin.Context) {
	h.launch(c, h.svc.LaunchOnCluster)
}

type launchFunc func(ctx context.Context, identity auth.Identity, targetID, scriptID bson.ObjectID) (model.Task, error)

func (h *taskHandler) launch(c *gin.Context, run launchFunc) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	scriptID, ok := httpx.ParseID(c, "scriptId")
	if !ok {
		return
	}
	task, err := run(c.Request.Context(), auth.FromContext(c), id, scriptID)
	if err != nil {
		abortError(c, err)
		return
	}
	links := taskLinks(task)
	c.Header("Location", links["self"].Href)
	c.JSON(http.StatusAccepted, dto.NewTask(task, links))
}

// list: GET /tasks -> adminul vede toate task-urile, maintainerul doar pe ale lui.
func (h *taskHandler) list(c *gin.Context) {
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	tasks, total, err := h.svc.ListTasks(c.Request.Context(), auth.FromContext(c), page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}
	items := make([]dto.Task, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, dto.NewTask(task, taskLinks(task)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	body.Links["parent"] = httpx.Link{Href: basePath}
	c.JSON(http.StatusOK, body)
}

// get: GET /tasks/:id -> 200; 404 daca nu exista; 403 pentru task-ul altui maintainer.
func (h *taskHandler) get(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	task, err := h.svc.GetTask(c.Request.Context(), auth.FromContext(c), id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewTask(task, taskLinks(task)))
}
