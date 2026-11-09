package auth

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"job-management-service/internal/httpx"
)

const identityKey = "auth.identity"

func Middleware(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			c.Header("WWW-Authenticate", "Bearer")
			httpx.AbortWithProblem(c, http.StatusUnauthorized, "Lipseste header-ul Authorization: Bearer <token>.")
			return
		}

		identity, err := validator.Validate(c.Request.Context(), token)
		switch {
		case err == nil:
			c.Set(identityKey, identity)
			c.Next()
		case errors.Is(err, ErrInvalidToken):
			c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			httpx.AbortWithProblem(c, http.StatusUnauthorized, "Token-ul este invalid, expirat sau revocat.")
		case errors.Is(err, ErrUnavailable):
			log.Printf("autentificare: %v", err)
			httpx.AbortWithProblem(c, http.StatusServiceUnavailable, "Serviciul de autentificare nu este disponibil momentan.")
		default:
			log.Printf("autentificare: %v", err)
			httpx.AbortWithProblem(c, http.StatusInternalServerError, "Eroare interna a serverului.")
		}
	}
}

// RequireAdmin permite operatia doar administratorului (L3 2.1); altfel 403 (RFC 9110, 15.5.4).
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !FromContext(c).IsAdmin() {
			httpx.AbortWithProblem(c, http.StatusForbidden, "Operatia este permisa doar administratorului.")
			return
		}
		c.Next()
	}
}

// FromContext intoarce utilizatorul pus in context de Middleware.
func FromContext(c *gin.Context) Identity {
	value, _ := c.Get(identityKey)
	identity, _ := value.(Identity)
	return identity
}

// bearerToken extrage token-ul din "Bearer <token>".
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	// Numele schemei nu tine cont de majuscule (RFC 9110, sec. 11.1).
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
