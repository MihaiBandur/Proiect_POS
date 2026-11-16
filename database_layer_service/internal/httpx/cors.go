package httpx

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	corsMethods        = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	corsRequestHeaders = []string{"Authorization", "Content-Type"}
	corsExposedHeaders = []string{"Location", "WWW-Authenticate", "Allow"}
)

const corsMaxAge = 600

func CORS(allowedOrigins []string) gin.HandlerFunc {
	methods := strings.Join(corsMethods, ", ")
	requestHeaders := strings.Join(corsRequestHeaders, ", ")
	exposedHeaders := strings.Join(corsExposedHeaders, ", ")
	maxAge := strconv.Itoa(corsMaxAge)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		// Raspunsul depinde de Origin, deci un cache nu trebuie sa-l refoloseasca pentru
		// alta origine (RFC 9110, 12.5.5: Vary).
		c.Writer.Header().Add("Vary", "Origin")
		if !slices.Contains(allowedOrigins, origin) {
			c.Next()
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
			c.Header("Access-Control-Allow-Methods", methods)
			c.Header("Access-Control-Allow-Headers", requestHeaders)
			c.Header("Access-Control-Max-Age", maxAge)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Header("Access-Control-Expose-Headers", exposedHeaders)
		c.Next()
	}
}
