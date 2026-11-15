package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/auth"
	"job-management-service/internal/dto"
	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
	"job-management-service/internal/repository"
	"job-management-service/internal/service"
)

const usersPath = basePath + "/users"

// userHandler este controller-ul pentru /users (SP 1.2 si 1.3). Toate rutele sunt doar pentru admin.
type userHandler struct {
	repo     repository.UserRepository
	accounts auth.AccountDirectory
	svc      *service.Service
}

// toModel verifica in A&A Services ca uid-ul exista si copiaza numele de utilizator
// (SP, Tabelul 2: id_user_grpc si userName trebuie sa existe in baza SQL).
// Daca nu reuseste, raspunde cu eroarea potrivita si intoarce false.
func (h *userHandler) toModel(c *gin.Context, req *dto.UserRequest) (model.User, bool) {
	account, err := h.accounts.FindAccount(c.Request.Context(), auth.TokenFromContext(c), req.UID())
	switch {
	case err == nil:
		return req.ToModel(account.Username), true
	case errors.Is(err, auth.ErrAccountNotFound):
		// Referinta catre o resursa inexistenta: valoare nepermisa (L3 1.2: 422).
		httpx.AbortWithFields(c, http.StatusUnprocessableEntity, "Unele campuri au valori care nu sunt admise.", []httpx.FieldError{{
			Field:   "id_user_grpc",
			Message: "nu exista niciun cont cu acest uid in A&A Services",
		}})
	case errors.Is(err, auth.ErrUnavailable):
		log.Printf("ListUsers: %v", err)
		httpx.AbortWithProblem(c, http.StatusServiceUnavailable, "Serviciul de autentificare nu este disponibil momentan.")
	default:
		log.Printf("ListUsers: %v", err)
		httpx.AbortWithProblem(c, http.StatusInternalServerError, "Eroare interna a serverului.")
	}
	return model.User{}, false
}

func (h *userHandler) list(c *gin.Context) {
	page, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	users, total, err := h.repo.List(c.Request.Context(), page.Offset(), page.Size)
	if err != nil {
		abortError(c, err)
		return
	}

	identity := auth.FromContext(c)
	items := make([]dto.User, 0, len(users))
	for _, user := range users {
		items = append(items, dto.NewUser(user, itemLinks(usersPath, user.ID, identity)))
	}
	body := httpx.NewCollection(items, page, total, c.Request.URL)
	addCollectionLinks(body.Links, usersPath, identity)
	c.JSON(http.StatusOK, body)
}

func (h *userHandler) get(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	user, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUser(user, itemLinks(usersPath, user.ID, auth.FromContext(c))))
}

// create: POST /users -> 201 + Location; 422 daca uid-ul nu exista in A&A; 409 daca e deja adaugat.
func (h *userHandler) create(c *gin.Context) {
	var req dto.UserRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	user, ok := h.toModel(c, &req)
	if !ok {
		return
	}
	user, err := h.repo.Create(c.Request.Context(), user)
	if err != nil {
		abortError(c, err)
		return
	}
	links := itemLinks(usersPath, user.ID, auth.FromContext(c))
	c.Header("Location", links["self"].Href)
	c.JSON(http.StatusCreated, dto.NewUser(user, links))
}

// replace: PUT /users/:id -> 204 sau 201. Reface userName din A&A Services; asignarile raman.
func (h *userHandler) replace(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UserRequest
	if !httpx.DecodeJSON(c, &req) {
		return
	}
	user, ok := h.toModel(c, &req)
	if !ok {
		return
	}
	user.ID = id
	created, err := h.repo.Replace(c.Request.Context(), user)
	if err != nil {
		abortError(c, err)
		return
	}
	if !created {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusCreated, dto.NewUser(user, itemLinks(usersPath, user.ID, auth.FromContext(c))))
}

// delete: DELETE /users/:id sterge documentul din MongoDB si asignarile lui de pe calculatoare
// si clustere; contul din A&A Services ramane (se dezactiveaza acolo, cu DeactivateAccount).
func (h *userHandler) delete(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	user, err := h.svc.DeleteUser(c.Request.Context(), id)
	if err != nil {
		abortError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUser(user, deletedLinks(usersPath)))
}
