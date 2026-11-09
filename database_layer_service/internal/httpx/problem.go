package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const problemContentType = "application/problem+json"

type Problem struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func AbortWithProblem(c *gin.Context, status int, detail string) {
	c.Header("Content-Type", problemContentType)
	c.AbortWithStatusJSON(status, Problem{
		Type:     "about:blank",
		Title:    http.StatusText(status),
		Status:   status,
		Detail:   detail,
		Instance: c.Request.URL.Path,
	})
}
