package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/httpx"
	"job-management-service/internal/repository"
	"job-management-service/internal/service"
)

func abortError(c *gin.Context, err error) {
	var (
		notFound  *service.NotFoundError
		conflict  *service.ConflictError
		duplicate *repository.DuplicateError
	)
	switch {
	case errors.As(err, &notFound):
		httpx.AbortWithProblem(c, http.StatusNotFound, notFound.Detail)
	case errors.Is(err, repository.ErrNotFound):
		httpx.AbortWithProblem(c, http.StatusNotFound, "Resursa ceruta nu exista.")
	case errors.Is(err, service.ErrNotAssigned):
		// L3 2.1: maintainerul vede doar resursele repartizate lui (RFC 9110, 15.5.4).
		httpx.AbortWithProblem(c, http.StatusForbidden, "Resursa nu ti-a fost asignata.")
	case errors.As(err, &duplicate):
		// Constrangere de unicitate (L3 1.2: 409; RFC 9110, 15.5.10).
		httpx.AbortWithFields(c, http.StatusConflict, "Exista deja o resursa cu aceeasi valoare.", []httpx.FieldError{{
			Field:   duplicate.Field,
			Message: "valoarea trebuie sa fie unica",
		}})
	case errors.As(err, &conflict):
		httpx.AbortWithProblem(c, http.StatusConflict, conflict.Detail)
	case errors.Is(err, repository.ErrUnavailable):
		log.Printf("baza de date: %v", err)
		httpx.AbortWithProblem(c, http.StatusServiceUnavailable, "Baza de date nu este disponibila momentan.")
	case errors.Is(err, auth.ErrUnavailable):
		log.Printf("A&A Services: %v", err)
		httpx.AbortWithProblem(c, http.StatusServiceUnavailable, "Serviciul de autentificare nu este disponibil momentan.")
	default:
		log.Printf("eroare interna: %v", err)
		httpx.AbortWithProblem(c, http.StatusInternalServerError, "Eroare interna a serverului.")
	}
}
