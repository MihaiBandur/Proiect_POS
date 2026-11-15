package dto

import (
	"strings"

	"job-management-service/internal/httpx"
	"job-management-service/internal/model"
)

type ComputerRequest struct {
	IP       *string `json:"IP"`
	Hostname *string `json:"hostname"`
}

func (r *ComputerRequest) Validate(v *httpx.Validation) {
	if r.IP == nil {
		v.Missing("IP")
	} else if _, ok := canonicalIP(*r.IP); !ok {
		v.Invalid("IP", "trebuie sa fie o adresa IPv4 sau IPv6, ex. 192.168.1.10")
	}
	if r.Hostname != nil && !isHostname(*r.Hostname) {
		v.Invalid("hostname", "trebuie sa fie un nume de host valid (RFC 1123), ex. node-01.cluster.local")
	}
}

func (r *ComputerRequest) ToModel() model.Computer {
	ip, _ := canonicalIP(*r.IP)
	computer := model.Computer{IP: ip}
	if r.Hostname != nil {
		computer.Hostname = strings.ToLower(*r.Hostname)
	}
	return computer
}

// Computer este reprezentarea unui calculator trimisa clientului.
// id_script si id_users se pot doar citi aici; se modifica prin rutele de legatura.
type Computer struct {
	ID       string      `json:"id"`
	IP       string      `json:"IP"`
	Hostname string      `json:"hostname,omitempty"`
	IDScript []string    `json:"id_script"`
	IDUsers  []string    `json:"id_users"`
	Links    httpx.Links `json:"_links"`
}

// NewComputer construieste reprezentarea; link-urile le alege controller-ul, dupa rol.
func NewComputer(computer model.Computer, links httpx.Links) Computer {
	return Computer{
		ID:       computer.ID.Hex(),
		IP:       computer.IP,
		Hostname: computer.Hostname,
		IDScript: hexIDs(computer.IDScript),
		IDUsers:  hexIDs(computer.IDUsers),
		Links:    links,
	}
}
