package dto

import (
	"strings"

	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
)

type UserRequest struct {
	IDUserGRPC *string `json:"id_user_grpc"`
}

func (r *UserRequest) Validate(v *httpx.Validation) {
	if r.IDUserGRPC == nil {
		v.Missing("id_user_grpc")
	} else if !uuidPattern.MatchString(*r.IDUserGRPC) {
		v.Invalid("id_user_grpc", "trebuie sa fie un UUID, ex. 0190f2a8-7c1e-7a3b-9d4e-2f6a8b1c3d5e")
	}
}

func (r *UserRequest) UID() string {
	return strings.ToLower(*r.IDUserGRPC)
}

func (r *UserRequest) ToModel(userName string) model.User {
	return model.User{IDUserGRPC: r.UID(), UserName: userName}
}

// User este reprezentarea unui utilizator trimisa clientului.
// id_comp_clust se poate doar citi aici; se modifica prin rutele de asignare.
type User struct {
	ID          string       `json:"id"`
	IDUserGRPC  string       `json:"id_user_grpc"`
	UserName    string       `json:"userName"`
	IDCompClust []ManagedRef `json:"id_comp_clust"`
	Links       httpx.Links  `json:"_links"`
}

// ManagedRef este un element din id_comp_clust, in forma din SP: {id, tip}.
type ManagedRef struct {
	ID  string `json:"id"`
	Tip string `json:"tip"`
}

// NewUser construieste reprezentarea; link-urile le alege controller-ul.
func NewUser(user model.User, links httpx.Links) User {
	refs := make([]ManagedRef, 0, len(user.IDCompClust))
	for _, ref := range user.IDCompClust {
		refs = append(refs, ManagedRef{ID: ref.ID.Hex(), Tip: ref.Tip})
	}
	return User{
		ID:          user.ID.Hex(),
		IDUserGRPC:  user.IDUserGRPC,
		UserName:    user.UserName,
		IDCompClust: refs,
		Links:       links,
	}
}
