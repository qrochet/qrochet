package app

import "net/http"
import "net/mail"
import "strconv"
import "log/slog"

type login struct {
	Email  string
	Pass   string
	Submit bool
	OK     bool
}

func (q *Qrochet) login(wr http.ResponseWriter, req *http.Request) {
	var err error

	v := q.view()
	if v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Already logged in.")
		return
	}

	slog.Info("Login")
	if req.Method == "POST" {
		err = req.ParseMultipartForm(mpfMaxMemory)
		if err != nil {
			slog.Error("Login req.ParseForm", "err", err)
			v.DisplayError(wr, req, "Form error.")
			return
		}
	}
	v.Login.Email = req.FormValue("email")
	v.Login.Pass = req.FormValue("pass")
	v.Login.Submit, _ = strconv.ParseBool(req.FormValue("submit"))

	if v.Login.Submit {
		_, err = mail.ParseAddress(v.Login.Email)
		if err != nil {
			slog.Error("mail.ParseAddress", "err", err, "v.Login.Email", v.Login.Email)
			v.DisplayError(wr, req, "Email is not valid.")
			return
		}

		existing, err := v.app.Repository.User.GetByEmail(req.Context(), v.Login.Email)
		if err != nil || existing == nil || existing.Email != v.Login.Email {
			slog.Error("User.GetForEmail", "err", err)
			v.DisplayError(wr, req, "This email address is not registered yet or the password is not correct")
			return
		}

		err = existing.CheckPassword(v.Login.Pass)
		if err != nil {
			slog.Error("User.CheckPassword", "err", err)
			v.DisplayError(wr, req, "This email address is not registered yet or the password is not correct")
			return
		}

		err = v.newSession(wr, req, *existing)
		if err != nil {
			slog.Error("User.Put", "err", err)
			v.DisplayError(wr, req, "Session creation failed")
			return
		}
		v.Message("Log in OK")
		v.Login.OK = true
		v.Display(wr, req)
		return
	} else {
		slog.Error("Not submitted?", "form", req.Form, "post", req.PostForm)
		v.Display(wr, req)
		return
	}
}
