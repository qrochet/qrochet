package app

import "net/http"
import "path"
import "fmt"
import "time"
import "log/slog"
import "aidanwoods.dev/go-paseto"

import "github.com/qrochet/qrochet/pkg/model"

// view is the view of state the current (autheticated) user
type view struct {
	app      *Qrochet
	Register register

	Messages []string // Messages to the user.
	Errors   []string // Error messages to the user.

	Session *model.Session
}

func (v *view) Message(form string, args ...any) {
	s := fmt.Sprintf(form, args)
	v.Messages = append(v.Messages, s)
}

func (v *view) Error(form string, args ...any) {
	s := fmt.Sprintf(form, args)
	v.Errors = append(v.Errors, s)
}

func (v *view) check(wr http.ResponseWriter, req *http.Request) error {
	cookie, err := req.Cookie("QROCHET_SESSION")
	if err != nil {
		slog.Error("Error parsing cookie", "err", err)
		// Cookie is not ok but it is the same as not being logged in.
		return nil
	}
	if cookie == nil {
		// Not logged in yet.
		return nil
	}
	if !cookie.Secure || cookie.Expires.Before(time.Now()) {
		slog.Error("Cookie expired or not secure", "err", err)
		// Cookie not ok
		return nil
	}
	tok, err := paseto.NewParser().ParseV4Local(v.app.Key, cookie.Value, []byte{})
	if err != nil {
		slog.Error("PASETO token not secure", "err", err)
		return err
	}

	sub, err := tok.GetSubject()
	if err != nil {
		slog.Error("PASETO token subject", "err", err)
		return err
	}
	session, err := v.app.Repository.Session.Get(req.Context(), sub)
	if err != nil {
		slog.Error("Session expired", "err", err)
		return err
	}

	if session.End.Before(time.Now()) {
		err = v.app.Repository.Session.Delete(req.Context(), sub)
		if err != nil {
			slog.Error("Could not delete expired session", "err", err)
		}
		slog.Error("session expired")
		return nil
	}

	v.Session = &session
	return nil
}

const sessionTimeout = time.Second * 60 * 60 * 24

func (v *view) newSession(wr http.ResponseWriter, req *http.Request, user model.User) error {
	var err error
	session := model.Session{
		UserID: user.ID,
		Start:  time.Now(),
		End:    time.Now().Add(sessionTimeout),
	}

	// Delete any existing sessions and do not care about errors.
	v.app.Repository.Session.Delete(req.Context(), v.Session.UserID)

	session, err = v.app.Repository.Session.Put(req.Context(), v.Session.UserID, *v.Session)
	if err != nil {
		v.Session = nil
		slog.Error("Cannot save session", "err", err)
		return err
	}
	v.Session = &session

	tok := paseto.NewToken()
	tok.SetNotBefore(time.Now())
	tok.SetExpiration(time.Now().Add(sessionTimeout))
	tok.SetSubject(v.Session.UserID)

	encrypted := tok.V4Encrypt(v.app.Key, []byte{})
	cookie := http.Cookie{}
	cookie.Secure = true
	cookie.HttpOnly = true
	cookie.Expires = session.End
	cookie.Value = encrypted
	cookie.Name = "QROCHET_SESSION"
	http.SetCookie(wr, &cookie)

	return nil
}

func (v *view) IsLoggedIn(wr http.ResponseWriter, req *http.Request) bool {
	err := v.check(wr, req)
	if err != nil {
		return false
	}
	return v.Session != nil
}

// Displays the template for the path or the request with this view.
func (v *view) Display(wr http.ResponseWriter, req *http.Request) {
	name := path.Base(req.URL.Path) + ".tmpl.html"
	err := v.app.Template.ExecuteTemplate(wr, name, v)
	if err != nil {
		slog.Error("template", "name", name, "err", err)
	}
}

func (v *view) DisplayError(wr http.ResponseWriter, req *http.Request, form string, args ...any) {
	v.Error(form, args...)
	v.Display(wr, req)
}
