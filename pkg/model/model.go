package model

import "fmt"
import "io"
import "time"
import "encoding"
import "errors"
import "encoding/base32"
import "log/slog"
import "golang.org/x/crypto/bcrypt"

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

func (r Role) String() string {
	switch r {
	case RoleNone:
		return "none"
	case RoleGuest:
		return "guest"
	case RoleStart:
		return "start"
	case RoleHobby:
		return "hobby"
	case RolePro:
		return "pro"
	case RoleStaff:
		return "staff"
	default:
		return "unknown"
	}
}

var ErrorUnknownRole = errors.New("unknown role")

func (r Role) MarshalText() ([]byte, error) {
	res := r.String()
	if res == "unknown" || res == "" {
		return nil, ErrorUnknownRole
	}
	return []byte(res), nil
}

func (r *Role) UnmarshalText(text []byte) error {
	s := string(text)
	switch s {
	case "none":
		*r = RoleNone
		return nil
	case "guest":
		*r = RoleGuest
		return nil
	case "start":
		*r = RoleStart
		return nil
	case "hobby":
		*r = RoleHobby
		return nil
	case "pro":
		*r = RolePro
		return nil
	case "staff":
		*r = RoleStaff
		return nil
	default:
		return ErrorUnknownRole
	}
}

var _ encoding.TextMarshaler = RoleStaff

// Theme is the UI theme the user is using.
type Theme string

// User	is a user of the Qrochet application without the password hash.
type User struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Role     Role     `json:"role"`
	Theme    Theme    `json:"theme"`
	CraftIDs []string `json:"craft_ids"`
	Hash     string   `json:"hash"` // password hash
}

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// Key is the base32 encoded ID, for use with NATS k/V.
func Key(id string) string {
	return b32.EncodeToString([]byte(id))
}

// Key is the base32 encoded ID, for use with NATS k/V.
func (u User) Key() string {
	return Key(u.ID)
}

func (u User) CheckPassword(pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(pass))
}

func (u *User) SetPassword(pass string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Hash = string(hash)
	return nil
}

func (u User) Redact() User {
	u.Hash = "*REDACTED*"
	return u
}

// Implement LogValuer on user for privacy and security.
func (u User) LogValue() slog.Value {
	return slog.AnyValue(u.Redact)
}

// Reference is a reference to an Referenceed file.
type Reference string

// Craft is a craft that a user has made and is presenting on Qrochet.
type Craft struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Title   string    `json:"title"`
	Detail  string    `json:"detail"`
	Image   Reference `json:"image"`
	Pattern Reference `json:"pattern"`
	Tags    []string  `json:"tags"`
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

// Session is a session of an authenticated user
type Session struct {
	UserID string    `json:"user_id"` // Also is the ID of the session of the session.
	Token  string    `json:"token"`   // Token is the security token for the session.
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

// Upload is an uploaded file.
type Upload struct {
	io.ReadCloser `json:"-"`
	ID            Reference `json:"id"`
	Title         string    `json:"title"`
	Detail        string    `json:"detail"`
	UserID        string    `json:"user_id"`
	MIME          string    `json:"mime"`
}
