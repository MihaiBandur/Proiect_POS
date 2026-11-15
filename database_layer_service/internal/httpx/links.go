package httpx

type Link struct {
	Href string `json:"href"`
	Type string `json:"type,omitempty"`
	// Templated: href e un sablon URI (RFC 6570), ex. /computers/{id}/scripts/{scriptId};
	// clientul inlocuieste {scriptId} cu id-ul ales. Forma e cea din HAL.
	Templated bool `json:"templated,omitempty"`
}

type Links map[string]Link
type Collection[T any] struct {
	Items        []T   `json:"items"`
	Page         int   `json:"page"`
	ItemsPerPage int   `json:"items_per_page"`
	TotalItems   int   `json:"total_items"`
	Links        Links `json:"_links"`
}
