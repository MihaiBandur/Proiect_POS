package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes = 1 << 20

const notConformDetail = "Reprezentarea nu este conforma cu structura resursei."

const unknownFieldPrefix = "json: unknown field "

type Validator interface {
	Validate(v *Validation)
}

type Validation struct {
	missing []FieldError
	invalid []FieldError
}

func (v *Validation) Missing(field string) {
	v.NotConform(field, "campul este obligatoriu")
}

// NotConform noteaza o parte a cererii care nu respecta structura resursei, cu un mesaj propriu
// (ex. un fisier binar acolo unde se cere text): tot 415, ca un camp lipsa (L3 1.2).
func (v *Validation) NotConform(field, message string) {
	v.missing = append(v.missing, FieldError{Field: field, Message: message})
}

func (v *Validation) Invalid(field, message string) {
	v.invalid = append(v.invalid, FieldError{Field: field, Message: message})
}

func DecodeJSON(c *gin.Context, dst Validator) bool {
	if !isJSON(c.GetHeader("Content-Type")) {
		c.Header("Accept", "application/json")
		AbortWithProblem(c, http.StatusUnsupportedMediaType, "Corpul cererii trebuie trimis cu Content-Type: application/json.")
		return false
	}

	body := http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		abortDecodeError(c, err)
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		AbortWithProblem(c, http.StatusBadRequest, "Corpul cererii trebuie sa contina un singur obiect JSON.")
		return false
	}

	return CheckValid(c, dst)
}

// CheckValid ruleaza regulile DTO-ului si raspunde 415 sau 422 daca nu sunt respectate.
// O folosesc atat corpurile JSON (DecodeJSON), cat si formularele multipart (upload).
func CheckValid(c *gin.Context, dst Validator) bool {
	var v Validation
	dst.Validate(&v)
	switch {
	case len(v.missing) > 0:
		AbortWithFields(c, http.StatusUnsupportedMediaType, notConformDetail, v.missing)
		return false
	case len(v.invalid) > 0:
		AbortWithFields(c, http.StatusUnprocessableEntity, "Unele campuri au valori care nu sunt admise.", v.invalid)
		return false
	}
	return true
}

// MediaType intoarce tipul corpului cererii, fara parametri si cu litere mici
// (ex. "multipart/form-data" din "multipart/form-data; boundary=..."), sau "" daca lipseste.
func MediaType(c *gin.Context) string {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil {
		return ""
	}
	return mediaType
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func abortDecodeError(c *gin.Context, err error) {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var tooLargeErr *http.MaxBytesError
	var invalidErr *json.InvalidUnmarshalError

	switch {
	case errors.Is(err, io.EOF):
		AbortWithProblem(c, http.StatusBadRequest, "Corpul cererii lipseste.")
	case errors.Is(err, io.ErrUnexpectedEOF):
		AbortWithProblem(c, http.StatusBadRequest, "JSON invalid: corpul se termina inainte de finalul obiectului.")
	case errors.As(err, &syntaxErr):
		AbortWithProblem(c, http.StatusBadRequest, fmt.Sprintf("JSON invalid: eroare de sintaxa la octetul %d.", syntaxErr.Offset))

	case errors.As(err, &tooLargeErr):
		AbortWithProblem(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("Corpul cererii depaseste limita de %d octeti.", tooLargeErr.Limit))

	case errors.As(err, &typeErr) && typeErr.Field == "":
		AbortWithProblem(c, http.StatusUnsupportedMediaType, "Corpul cererii trebuie sa fie un obiect JSON.")
	case errors.As(err, &typeErr):
		AbortWithFields(c, http.StatusUnsupportedMediaType, notConformDetail, []FieldError{{
			Field:   typeErr.Field,
			Message: fmt.Sprintf("tip gresit: s-a primit %s, se astepta %s", typeErr.Value, typeErr.Type),
		}})
	case isUnknownField(err):
		AbortWithFields(c, http.StatusUnsupportedMediaType, notConformDetail, []FieldError{{
			Field:   unknownFieldName(err),
			Message: "campul nu exista in aceasta resursa",
		}})
	case errors.As(err, &invalidErr):
		panic(err)
	default:
		AbortWithProblem(c, http.StatusBadRequest, "Corpul cererii nu a putut fi citit.")
	}
}

func isUnknownField(err error) bool {
	return strings.HasPrefix(err.Error(), unknownFieldPrefix)
}

func unknownFieldName(err error) string {
	quoted := strings.TrimPrefix(err.Error(), unknownFieldPrefix)
	if name, unquoteErr := strconv.Unquote(quoted); unquoteErr == nil {
		return name
	}
	return quoted
}
