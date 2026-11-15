package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ParseID(c *gin.Context, param string) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param(param))
	if err != nil || id.IsZero() {
		AbortWithFields(c, http.StatusBadRequest, "Identificatorul din adresa nu este valid.", []FieldError{{
			Field:   param,
			Message: "trebuie sa fie un ObjectID: 24 de caractere hexazecimale",
		}})
		return bson.ObjectID{}, false
	}
	return id, true
}
