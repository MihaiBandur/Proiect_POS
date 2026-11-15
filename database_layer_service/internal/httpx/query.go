package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// QueryBool citeste un filtru optional true/false din URL (ex. ?isActive=true, L3 2.1).
// nil inseamna ca parametrul lipseste; o alta valoare decat true/false da 400 si intoarce false.
func QueryBool(c *gin.Context, name string) (*bool, bool) {
	raw, present := c.GetQuery(name)
	if !present {
		return nil, true
	}
	switch raw {
	case "true":
		value := true
		return &value, true
	case "false":
		value := false
		return &value, true
	}
	AbortWithFields(c, http.StatusBadRequest, "Parametrii din adresa sunt invalizi.", []FieldError{{
		Field:   name,
		Message: "trebuie sa fie true sau false",
	}})
	return nil, false
}

// QueryFlag citeste un comutator din URL (ex. ?verbose, L3 2.1): prezent fara valoare sau cu
// "true" inseamna activ, lipsa sau "false" inseamna inactiv; altceva da 400 si intoarce false.
func QueryFlag(c *gin.Context, name string) (bool, bool) {
	raw, present := c.GetQuery(name)
	switch {
	case !present || raw == "false":
		return false, true
	case raw == "" || raw == "true":
		return true, true
	}
	AbortWithFields(c, http.StatusBadRequest, "Parametrii din adresa sunt invalizi.", []FieldError{{
		Field:   name,
		Message: "se scrie fara valoare (?" + name + ") sau cu true/false",
	}})
	return false, false
}
