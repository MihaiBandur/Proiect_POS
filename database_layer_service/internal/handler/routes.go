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

const basePath = "/api/manager"

// Dependencies sunt tot ce are nevoie router-ul. main.go le creeaza pe cele reale,
// iar testele pun in locul lor variante false sau in memorie.
type Dependencies struct {
	Validator auth.TokenValidator
	Accounts  auth.AccountDirectory
	Tx        repository.Transactor
	Scripts   repository.ScriptRepository
	Users     repository.UserRepository
	Computers repository.ComputerRepository
	Clusters  repository.ClusterRepository
}

// NewRouter construieste router-ul complet: tratarea erorilor HTTP si rutele serviciului.
func NewRouter(deps Dependencies) *gin.Engine {
	svc := service.New(deps.Tx, deps.Scripts, deps.Computers, deps.Clusters, deps.Users, deps.Accounts)

	router := httpx.NewEngine()
	api := router.Group(basePath, auth.Middleware(deps.Validator))
	// Rutele din acest grup cer in plus rolul de administrator (L3 2.1).
	admin := api.Group("", auth.RequireAdmin())

	api.GET("", getRoot)

	// Scripturi: oricine le poate citi; doar adminul le gestioneaza (L3 2.1).
	scripts := &scriptHandler{repo: deps.Scripts, svc: svc}
	api.GET("/scripts", scripts.list)
	api.GET("/scripts/:id", scripts.get)
	admin.POST("/scripts", scripts.create)
	admin.PUT("/scripts/:id", scripts.replace)
	admin.DELETE("/scripts/:id", scripts.delete)

	// Calculatoare: maintainerul vede doar ce i-a fost repartizat (L3 2.1); service-ul filtreaza.
	computers := &computerHandler{repo: deps.Computers, svc: svc}
	api.GET("/computers", computers.list)
	api.GET("/computers/:id", computers.get)
	admin.POST("/computers", computers.create)
	admin.PUT("/computers/:id", computers.replace)
	admin.DELETE("/computers/:id", computers.delete)
	// Scripturile instalate (L3 2.1: /computers/{id}/scripts, cu ?isActive si ?verbose).
	api.GET("/computers/:id/scripts", computers.listScripts)
	api.GET("/computers/:id/scripts/:scriptId", computers.getScript)
	admin.PUT("/computers/:id/scripts/:scriptId", computers.installScript)
	admin.DELETE("/computers/:id/scripts/:scriptId", computers.uninstallScript)
	// Maintainerii asignati (SP 1.3).
	computerUsers := computerAssignments(svc)
	admin.GET("/computers/:id/users", computerUsers.listUsers)
	admin.GET("/computers/:id/users/:userId", computerUsers.getUser)
	admin.PUT("/computers/:id/users/:userId", computerUsers.assignUser)
	admin.DELETE("/computers/:id/users/:userId", computerUsers.unassignUser)

	// Clustere: aceleasi reguli ca la calculatoare.
	clusters := &clusterHandler{repo: deps.Clusters, svc: svc}
	api.GET("/clusters", clusters.list)
	api.GET("/clusters/:id", clusters.get)
	admin.POST("/clusters", clusters.create)
	admin.PUT("/clusters/:id", clusters.replace)
	admin.DELETE("/clusters/:id", clusters.delete)
	// Calculatoarele din cluster (SP 1.3).
	api.GET("/clusters/:id/computers", clusters.listComputers)
	api.GET("/clusters/:id/computers/:computerId", clusters.getComputer)
	admin.PUT("/clusters/:id/computers/:computerId", clusters.addComputer)
	admin.DELETE("/clusters/:id/computers/:computerId", clusters.removeComputer)
	// Maintainerii asignati (SP 1.3).
	clusterUsers := clusterAssignments(svc)
	admin.GET("/clusters/:id/users", clusterUsers.listUsers)
	admin.GET("/clusters/:id/users/:userId", clusterUsers.getUser)
	admin.PUT("/clusters/:id/users/:userId", clusterUsers.assignUser)
	admin.DELETE("/clusters/:id/users/:userId", clusterUsers.unassignUser)

	// Gestiunea utilizatorilor e doar a adminului (SP 1.3).
	users := &userHandler{repo: deps.Users, accounts: deps.Accounts, svc: svc}
	admin.GET("/users", users.list)
	admin.GET("/users/:id", users.get)
	admin.POST("/users", users.create)
	admin.PUT("/users/:id", users.replace)
	admin.DELETE("/users/:id", users.delete)

	return router
}

func getRoot(c *gin.Context) {
	c.JSON(http.StatusOK, dto.Root{
		Links: httpx.Links{
			"self":      {Href: basePath},
			"scripts":   {Href: basePath + "/scripts"},
			"computers": {Href: basePath + "/computers"},
			"clusters":  {Href: basePath + "/clusters"},
			"users":     {Href: basePath + "/users"},
		},
	})
}
