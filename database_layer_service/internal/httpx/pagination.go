package httpx

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage         = 1
	defaultItemsPerPage = 10
	maxItemsPerPage     = 100
	maxPage             = math.MaxInt32
)

type Page struct {
	Number int
	Size   int
}

func (p Page) Offset() int {
	return (p.Number - 1) * p.Size
}

func ParsePage(c *gin.Context) (Page, bool) {
	var fields []FieldError
	number := queryInt(c, "page", defaultPage, maxPage, &fields)
	size := queryInt(c, "items_per_page", defaultItemsPerPage, maxItemsPerPage, &fields)
	if len(fields) > 0 {
		AbortWithFields(c, http.StatusBadRequest, "Parametrii de paginare sunt invalizi.", fields)
		return Page{}, false
	}
	return Page{Number: number, Size: size}, true
}

func queryInt(c *gin.Context, name string, fallback, highest int, fields *[]FieldError) int {
	raw, present := c.GetQuery(name)
	if !present {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	switch {
	case err != nil:
		*fields = append(*fields, FieldError{Field: name, Message: "trebuie sa fie un numar intreg"})
	case value < 1:
		*fields = append(*fields, FieldError{Field: name, Message: "trebuie sa fie cel putin 1"})
	case value > highest:
		*fields = append(*fields, FieldError{Field: name, Message: fmt.Sprintf("trebuie sa fie cel mult %d", highest)})
	default:
		return value
	}
	return fallback
}

func NewCollection[T any](items []T, page Page, totalItems int, requestURL *url.URL) Collection[T] {
	if items == nil {
		items = []T{}
	}
	return Collection[T]{
		Items:        items,
		Page:         page.Number,
		ItemsPerPage: page.Size,
		TotalItems:   totalItems,
		Links:        pageLinks(requestURL, page, totalItems),
	}
}

func pageLinks(requestURL *url.URL, page Page, totalItems int) Links {
	lastPage := max(1, (totalItems+page.Size-1)/page.Size) // o colectie goala are o pagina, goala
	links := Links{
		"self":  {Href: pageHref(requestURL, page.Number, page.Size)},
		"first": {Href: pageHref(requestURL, 1, page.Size)},
		"last":  {Href: pageHref(requestURL, lastPage, page.Size)},
	}
	if page.Number > 1 {

		links["prev"] = Link{Href: pageHref(requestURL, min(page.Number-1, lastPage), page.Size)}
	}
	if page.Number < lastPage {
		links["next"] = Link{Href: pageHref(requestURL, page.Number+1, page.Size)}
	}
	return links
}

func pageHref(requestURL *url.URL, number, size int) string {
	query := requestURL.Query()
	query.Set("page", strconv.Itoa(number))
	query.Set("items_per_page", strconv.Itoa(size))
	return requestURL.Path + "?" + query.Encode()
}
