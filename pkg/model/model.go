package model

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

// User	is a user of the Qrochet application
type User struct {
	ID       string   `json:"id"`
	Hash     string   `json:"hash"`
	Role     Role     `json:"role`
	Theme    Theme    `json:"theme`
	CraftIDs []string `json:"craft_ids"`
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
