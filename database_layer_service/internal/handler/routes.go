package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/httpx"
)

const basePath = "/api/manager"

func NewRouter(validator auth.TokenValidator) *gin.Engine {
	router := httpx.NewEngine()
	api := router.Group(basePath, auth.Middleware(validator))
	api.GET("", getRoot)
	return router
}

func RegisterRoutes(router *gin.Engine, validator auth.TokenValidator) {
	api := router.Group(basePath, auth.Middleware(validator))
	api.GET("", getRoot)
}

func getRoot(c *gin.Context) {
	c.JSON(http.StatusOK, dto.Root{
		Links: dto.Links{
			"self":      {Href: basePath},
			"scripts":   {Href: basePath + "/scripts"},
			"computers": {Href: basePath + "/computers"},
			"clusters":  {Href: basePath + "/clusters"},
			"users":     {Href: basePath + "/users"},
		},
	})
}
