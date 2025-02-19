package model

type Role int

const (
	RoleNone Role = iota
    RoleStarter
	RoleHobby
	RolePro
	RoleStaff
)

type User struct {
ID string `json:"id"`
Hash string `json:"hash"`

}

