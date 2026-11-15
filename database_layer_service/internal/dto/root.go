package dto

import "job-management-service/internal/httpx"

type Root struct {
	Links httpx.Links `json:"_links"`
}
