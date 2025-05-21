package model

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/mail"
	"time"
)

import (
	"github.com/oklog/ulid/v2"
	"golang.org/x/image/draw"
)

import (
	"github.com/qrochet/qrochet/pkg/censor"
)

const (
	maxImageSize = 4000 * 1000 * 1000
)

var (
	// ErrorEmailNotValid indicates an email adddress was not valid.
	ErrorEmailNotValid = errors.New("email not valid")

	// ErrorEmailNotRegistered indicates the email address is not registered yet or the password is not correct.
	// These two errors are not distinct to discourage brute force attacks somewhat.
	ErrorEmailNotRegistered = errors.New("this email address is not registered yet or the password is not correct")

	// ErrorSessionNotCreated indicates that the session creation failed.
	ErrorSessionNotCreated = errors.New("session creation failed")

	// ErrorAlreadyLoggedOut indicates that the session is already logged out or nil.
	ErrorAlreadyLoggedOut = errors.New("already logged out")

	// ErrorDeleteSession indicates that the session could not be deleted.
	ErrorDeleteSession = errors.New("could not delete session")

	// ErrorGetEmail indicates that there was an error getting user by email address
	ErrorGetEmail = errors.New("error getting user by email address")

	// ErrorEmailRegistered indicates that the email address was already registered.
	ErrorEmailRegistered = errors.New("This email address is already registered")

	// ErrorRegistrationFailed indicates registration failed for storage reasons.
	ErrorRegistrationFailed = errors.New("registration failed")

	// ErrorPleaseLogIn indicates that the session is already logged out or nil and that the user needs to log in.
	ErrorPleaseLogIn = errors.New("please log in")

	// ErrorImageTooLarge means the uploaded image is too large.
	ErrorImageTooLarge = errors.New("image too large, maximum 4 MiB.")

	// ErrorImageResize means scaling down an image file failed.
	ErrorImageResize = errors.New("image resize failed")

	// ErrorImageUpload means uploading an image file to the repository failed.
	ErrorImageUpload = errors.New("image upload failed")

	// ErrorImageGet means getting an image file from the repository failed.
	ErrorImageGet = errors.New("image get failed")

	// ErrorCraftCreate means creating a craft failed.
	ErrorCraftCreate = errors.New("craft create failed")
)

// Logic implements the model core business logic using abstracted interfaces.
// It is a concrete type itself fo ease of use.
type Logic struct {
	Repository
	Sender
}

// NewLogic returns a new instance of a logic processor that uses the given
// repository and sender.
func NewLogic(r Repository, s Sender) *Logic {
	return &Logic{Repository: r, Sender: s}
}

// NewSession creates a new session for the given user, deleting
// any old session first.
func (l *Logic) NewSession(ctx Context, user User, sessionTimeout time.Duration) (*Session, error) {
	var err error

	session := Session{
		UserID: user.ID,
		Start:  time.Now(),
		End:    time.Now().Add(sessionTimeout),
	}

	// Delete old session ignoring any errors in case it didn't exist.
	_ = l.Session().Delete(ctx, user.ID)

	session, err = l.Session().Put(ctx, user.ID, session)
	if err != nil {
		slog.Error("Cannot save session", "err", err, "user", user.ID, "email", user.Email)
		return nil, ErrorSessionNotCreated
	}
	return &session, nil
}

// Login logs in a user by email and password and returns the user and session.
func (l *Logic) Login(ctx Context, email, password string, sessionTimeout time.Duration) (*User, *Session, error) {
	_, err := mail.ParseAddress(email)
	if err != nil {
		slog.Error("mail.ParseAddress", "err", err, "email", email)
		return nil, nil, ErrorEmailNotValid
	}

	existing, err := l.User().GetByEmail(ctx, email)
	if err != nil || existing == nil || existing.Email != email {
		slog.Error("User.GetForEmail", "err", err, "email", email)
		return nil, nil, ErrorEmailNotRegistered
	}

	err = existing.CheckPassword(password)
	if err != nil {
		slog.Error("User.CheckPassword", "err", err, "email", email)
		return nil, nil, ErrorEmailNotRegistered
	}

	session, err := l.NewSession(ctx, *existing, sessionTimeout)
	if err != nil {
		slog.Error("Logic.NewSession", "err", err, "email", email)
		return nil, nil, ErrorSessionNotCreated
	}
	return existing, session, nil
}

// Logout logs out a user with the given session.
// The session will be deleted and cannot be used anymore after this.
// The UserID of the session will be set to the empty string.
func (l *Logic) Logout(ctx Context, session *Session) error {
	if session == nil || session.UserID == "" {
		return ErrorAlreadyLoggedOut
	}

	slog.Info("logout")

	err := l.Session().Delete(ctx, session.UserID)
	if err != nil {
		slog.Error("could not delete session", "err", err)
		return ErrorDeleteSession
	}
	session.UserID = ""
	return nil
}

func (l *Logic) sendRegistrationMail(user User) error {
	if l.Sender == nil {
		slog.Warn("Mailer not available will not send registration email.")
		return nil
	}

	msg := Mail{}
	msg.To = user.Name + "<" + user.Email + ">"

	msg.Printf("Dear %s, welcome to Qrochet\n\n", user.Name)
	msg.Println("Thank you for registering with Qrochet, the website for chroochet and hand crafts.")
	msg.Println("This is a message to confirm your registration. You do not have to reply to it.")
	msg.Println("Kind regards, Qrochet.")

	err := l.Sender.Send(msg)
	if err != nil {
		slog.Error("While sending mail", "err", err)
		return err
	}
	return nil
}

// Register registers a new user of Qrochet, sending an email if possible.
// It does not log in the newly registered user.
func (l *Logic) Register(ctx Context, name, email, password string) (*User, error) {
	var err error

	slog.Info("register")

	_, err = mail.ParseAddress(email)
	if err != nil {
		slog.Error("mail.ParseAddress", "err", err, "email", email)
		return nil, ErrorEmailNotValid
	}

	user := User{}
	user.ID = ulid.Make().String()
	user.Email = email
	user.Name = name
	user.SetPassword(password)

	existing, err := l.User().GetByEmail(ctx, user.Email)
	if err != nil {
		slog.Error("User.GetByEmail", "err", err)
		return nil, ErrorGetEmail
	}

	if existing != nil && existing.Email == user.Email {
		slog.Error("User.GetByEmail", "err", err)
		return nil, ErrorEmailRegistered
	}

	created, err := l.User().Put(ctx, user.ID, user)
	if err != nil {
		slog.Error("User.Put", "err", err)
		return nil, ErrorRegistrationFailed
	}

	// Send mail but don't care if it worked or not.
	// Maybe later if we have private messages then send one there.
	go l.sendRegistrationMail(created)
	return &created, nil
}

// RegisterAndLogin registers a new user of Qrochet,
// sending an email if possible, and then logs in that user.
func (l *Logic) RegisterAndLogin(ctx Context, name, email, password string, sessionTimeout time.Duration) (*User, *Session, error) {
	user, err := l.Register(ctx, name, email, password)
	if err != nil {
		return user, nil, err
	}
	return l.Login(ctx, user.Email, password, sessionTimeout)
}

func resizeImageJPEG(input io.Reader, width, height, quality int) (*bytes.Buffer, error) {
	output := &bytes.Buffer{}

	src, _, err := image.Decode(input)

	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill image with transparent white first
	tw := color.RGBA{255, 255, 255, 0}
	draw.Draw(dst, dst.Bounds(), &image.Uniform{tw}, image.ZP, draw.Src)
	// Then draw image
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	err = jpeg.Encode(output, dst, &jpeg.Options{Quality: quality})
	return output, err
}

// NewCraftForSession creates a new craft for the user that this session belongs to.
func (l *Logic) NewCraftForSession(ctx Context, name, description string, file multipart.File, header *multipart.FileHeader, session *Session) (*Craft, error) {
	var err error

	if session == nil || session.UserID == "" {
		return nil, ErrorPleaseLogIn
	}

	slog.Info("NewCraftForSession")

	name = censor.Replace(name)
	description = censor.Replace(description)

	if header.Size > maxImageSize {
		slog.Error("createCraft Size > maxImageSize", "err", err)
		return nil, ErrorImageTooLarge
	}

	resized, err := resizeImageJPEG(file, 640, 640, 90)
	if err != nil {
		slog.Error("Image resizing failed", "err", err)
		return nil, ErrorImageResize
	}

	upload := &Upload{
		ID:         Reference(ulid.Make().String() + ".jpeg"),
		UserID:     session.UserID,
		ReadCloser: io.NopCloser(resized),
	}

	upload, err = l.Image().Put(ctx, upload)
	if err != nil {
		slog.Error("Image upload failed", "err", err)
		return nil, ErrorImageUpload
	}

	craft := &Craft{}
	craft.ID = ulid.Make().String()
	craft.Title = name
	craft.Detail = description
	craft.Image = upload.ID
	craft.UserID = session.UserID
	created, err := l.Craft().Put(ctx, craft.ID, *craft)
	if err != nil {
		slog.Error("Craft.Put", "err", err)
		// XXX should probably delete the uploaded image if the craft
		// cannot be created to prevent it from "dangling".
		return nil, ErrorCraftCreate
	}

	return &created, nil
}

// CraftsForSession returns the crafts for the user that this session belongs to.
func (l *Logic) CraftsForSession(ctx Context, session *Session) (chan Craft, error) {
	if session == nil || session.UserID == "" {
		return nil, ErrorPleaseLogIn
	}
	return l.Craft().AllForUserID(ctx, session.UserID)
}

// GetImage returns an uploaded image for the session and ID
func (l *Logic) GetImage(ctx Context, session *Session, id string) (*Upload, error) {
	if session == nil || session.UserID == "" {
		return nil, ErrorPleaseLogIn
	}

	file, err := l.Image().Get(ctx, id)
	if err != nil {
		slog.Error("GetImage", "err", err)
		return nil, ErrorImageGet
	}

	return file, nil
}
