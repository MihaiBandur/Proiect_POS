package dto

import (
	"strings"

	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
)

type ScriptRequest struct {
	Name      *string `json:"name"`
	StorePath *string `json:"store_path"`
	StoreComp *string `json:"store_comp"`
	IsActive  *bool   `json:"is_active"`
}

func (r *ScriptRequest) Validate(v *httpx.Validation) {
	requireText(v, "name", r.Name)
	requireAbsolutePath(v, "store_path", r.StorePath)
	requireAbsolutePath(v, "store_comp", r.StoreComp)
}

func (r *ScriptRequest) ToModel() model.Script {
	return model.Script{
		Name:      strings.TrimSpace(*r.Name),
		StorePath: *r.StorePath,
		StoreComp: *r.StoreComp,
		IsActive:  r.IsActive != nil && *r.IsActive,
	}
}

type Script struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	StorePath string      `json:"store_path"`
	StoreComp string      `json:"store_comp"`
	IsActive  bool        `json:"is_active"`
	Links     httpx.Links `json:"_links"`
}

func NewScript(script model.Script, links httpx.Links) Script {
	return Script{
		ID:        script.ID.Hex(),
		Name:      script.Name,
		StorePath: script.StorePath,
		StoreComp: script.StoreComp,
		IsActive:  script.IsActive,
		Links:     links,
	}
}
