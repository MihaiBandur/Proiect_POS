package httpx

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewEngine() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), recovery())

	engine.HandleMethodNotAllowed = true
	engine.NoRoute(notFound)
	engine.NoMethod(methodNotAllowed)
	return engine
}

func notFound(c *gin.Context) {
	AbortWithProblem(c, http.StatusNotFound, "Nu exista nicio resursa la aceasta adresa.")

}

func methodNotAllowed(c *gin.Context) {
	AbortWithProblem(c, http.StatusMethodNotAllowed,
		fmt.Sprintf("Metoda %s nu este permisa pe aceasta resursa.", c.Request.Method))
}

func recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, _ any) {
		AbortWithProblem(c, http.StatusInternalServerError, "Eroare interna a serverului.")
	})
}
