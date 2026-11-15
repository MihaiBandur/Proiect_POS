package dto

import (
	"strings"

	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
)

type ClusterRequest struct {
	Nume *string `json:"nume"`
}

func (r *ClusterRequest) Validate(v *httpx.Validation) {
	requireText(v, "nume", r.Nume)
}

func (r *ClusterRequest) ToModel() model.Cluster {
	return model.Cluster{Nume: strings.TrimSpace(*r.Nume)}
}

// Cluster este reprezentarea unui cluster trimisa clientului.
// id_comp si id_users se pot doar citi aici; se modifica prin rutele de legatura.
type Cluster struct {
	ID      string      `json:"id"`
	Nume    string      `json:"nume"`
	IDComp  []string    `json:"id_comp"`
	IDUsers []string    `json:"id_users"`
	Links   httpx.Links `json:"_links"`
}

// NewCluster construieste reprezentarea; link-urile le alege controller-ul, dupa rol.
func NewCluster(cluster model.Cluster, links httpx.Links) Cluster {
	return Cluster{
		ID:      cluster.ID.Hex(),
		Nume:    cluster.Nume,
		IDComp:  hexIDs(cluster.IDComp),
		IDUsers: hexIDs(cluster.IDUsers),
		Links:   links,
	}
}
