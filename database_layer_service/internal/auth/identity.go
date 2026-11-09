package auth

type Role int

const (
	RoleUnknown Role = iota
	RoleAdmin
	RoleMaintainer
)

type Identity struct {
	UID  string
	Role Role
}

func (i Identity) IsAdmin() bool {
	return i.Role == RoleAdmin
}
