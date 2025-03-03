package model

import "fmt"

// Role is the role of a user. It also determines privileges.
type Role int

const (
	RoleNone Role = iota * 100
	RoleGuest
	RoleStart
	RoleHobby
	RolePro
	RoleStaff
)

// Theme is the UI theme the user is using.
type Theme string

// User	is a user of the Qrochet application without the password hash.
type User struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Role     Role     `json:"role`
	Theme    Theme    `json:"theme`
	CraftIDs []string `json:"craft_ids"`
}

// UserWithHash	is a user of the Qrochet application with their hashed password.
type UserWithHash struct {
	User
	Hash string `json:"hash"`
}

// Upload is a reference to an uploaded file.
type Upload string

// Craft is a craft that a user has made and is presenting on Qrochet.
type Craft struct {
	ID      string   `json:"id"`
	UserID  string   `json:"user_id"`
	Title   string   `json:"title"`
	Detail  string   `json:"detail"`
	Image   Upload   `json:"image"`
	Pattern Upload   `json:"pattern"`
	Tags    []string `json:"tags"`
}

// Login in a log in request
type Login struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

// Error is returned on errors.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

// Accept is a response to accept a login.
type Accept struct {
	Self  User   `json:"self"`
	Token string `json:"token"`
}

// RangeQuery is a generic range query request for a resource.
type RangeQuery[T any] struct {
	First  string `json:"first"`
	Amount int    `json:"amount"`
	Item   T
}

// RangeResult is a generic range query result for a resource.
type RangeResult[T any] struct {
	First  string `json:"first"`
	Last   string `json:"last"`
	Amount int    `json:"amount"`
	Items  []T
}

// GetQuery is a generic get query request for a resource.
type GetResult[T any] struct {
	ID   string `json:"id"`
	Item T
}
