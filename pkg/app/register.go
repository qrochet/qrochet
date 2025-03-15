package app

import "net/http"
import "net/mail"
import "strconv"
import "log/slog"
import "math/rand/v2"

import "github.com/qrochet/qrochet/pkg/model"

var stupidCAPTCHA = []struct {
	q string
	a int
}{
	{q: "Two plus three is?", a: 5},
	{q: "Nine minus seven is?", a: 2},
	{q: "Three times three?", a: 9},
}

type register struct {
	Name     string
	Email    string
	Pass     string
	CAPTCHAQ string
	CAPTCHA  int
	CAPTCHAI int
	Submit   bool
	OK       bool
}

const mpfMaxMemory = 0

func (q *Qrochet) register(wr http.ResponseWriter, req *http.Request) {
	var err error

	v := q.view()
	if v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Already logged in.")
		return
	}

	slog.Info("register")
	err = req.ParseMultipartForm(mpfMaxMemory)
	if err != nil {
		slog.Error("register req.ParseForm", "err", err)
		v.DisplayError(wr, req, "Form error.")
		return
	}
	v.Register.Name = req.FormValue("name")
	v.Register.Email = req.FormValue("email")
	v.Register.Pass = req.FormValue("pass")
	v.Register.Submit, _ = strconv.ParseBool(req.FormValue("submit"))
	v.Register.CAPTCHA, _ = strconv.Atoi(req.FormValue("capcha"))
	v.Register.CAPTCHAI, _ = strconv.Atoi(req.FormValue("capchai"))

	if v.Register.Submit {
		_, err = mail.ParseAddress(v.Register.Email)
		if err != nil {
			slog.Error("mail.ParseAddress", "err", err, "v.Register.Email", v.Register.Email)
			v.DisplayError(wr, req, "Email is not valid.")
			return
		}

		user := model.User{}
		user.ID = model.Key(v.Register.Email)
		user.Email = v.Register.Email
		user.Name = v.Register.Name
		user.SetPassword(v.Register.Pass)

		existing, err := v.app.Repository.User.Get(req.Context(), user.ID)
		if err == nil && existing.ID == user.ID {
			slog.Error("User.Put", "err", err)
			v.DisplayError(wr, req, "This email address is already registered")
			return
		}

		created, err := v.app.Repository.User.Put(req.Context(), user.ID, user)
		if err != nil {
			slog.Error("User.Put", "err", err)
			v.DisplayError(wr, req, "Registration failed")
			return
		}
		err = v.newSession(wr, req, created)
		if err != nil {
			slog.Error("User.Put", "err", err)
			v.DisplayError(wr, req, "Session creation failed")
			return
		}
		v.Message("Registration OK")
		v.Register.OK = true
		v.Display(wr, req)
		return
	} else {
		slog.Error("Not submitted?", "form", req.Form, "post", req.PostForm)
		v.Register.CAPTCHA = 0
		v.Register.CAPTCHAI = rand.IntN(len(stupidCAPTCHA))
		v.Register.CAPTCHAQ = stupidCAPTCHA[v.Register.CAPTCHAI].q
		v.Display(wr, req)
		return
	}
}
