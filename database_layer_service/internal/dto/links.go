package dto

type Link struct {
	Href string `json:"href"`
	Type string `json:"type,omitempty"`
}

type Links map[string]Link

type Root struct {
	Links Links `json:"_links"`
}
