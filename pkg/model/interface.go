// package model is the data model and business logic of Qrochet.
package model

import "context"

// Context is an alias for context.Context.
type Context = context.Context

// A repository allows storing and working with the model's structs.
type Repository interface {
	// User returns the user mapper for this repository.
	User() UserMapper
	// Session returns the session mapper for this repository.
	Session() SessionMapper
	// Craft returns the craft mapper for this repository.
	Craft() CraftMapper
	// Image returns the image mapper for this repository.
	Image() UploadMapper
	// Close closes the repository.
	Close()
}

// BasicMapper is a basic data mapper for one type T.
type BasicMapper[T any] interface {
	Get(ctx Context, key string) (T, error)
	Put(ctx Context, key string, obj T) (T, error)
	Purge(ctx Context, key string) error
	Keys(ctx Context, keys ...string) (chan (string), error)
	Watch(ctx Context, keys ...string) (chan (T), error)
	All(ctx Context, keys ...string) (chan (T), error)
	Delete(ctx Context, key string) error
	GetFirstMatch(ctx Context, matcher func(t *T) bool) (*T, error)
}

// SessionMapper is a data mapper for sessions.
type SessionMapper interface {
	BasicMapper[Session]
}

// UploadMapper is a data mapper for uploads.
type UploadMapper interface {
	Get(ctx Context, key string) (*Upload, error)
	Put(ctx Context, up *Upload) (*Upload, error)
	Delete(ctx Context, key string) error
	List(ctx Context, userId string) (chan (string), error)
	Watch(ctx Context) (chan (*Upload), error)
}

// CraftMapper is a mapper for cafts.
type CraftMapper interface {
	// Inherit from BasicMapper
	BasicMapper[Craft]
	GetForUserID(ctx Context, key string, UserID string) (Craft, error)
	AllForUserID(ctx Context, UserID string) (chan Craft, error)
}

// UserMapper is a mapper for users.
type UserMapper interface {
	// Inherit from BasicMapper
	BasicMapper[User]
	GetByEmail(ctx Context, email string) (*User, error)
}
