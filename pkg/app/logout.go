package app

import "net/http"
import "time"
import "log/slog"
import "strconv"

type logout struct {
	Submit bool
	OK     bool
}

func (q *Qrochet) logout(wr http.ResponseWriter, req *http.Request) {
	var err error

	v := q.view()
	if !v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Already logged out.")
		return
	}

	slog.Info("logout")
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
	v.Logout.Submit, _ = strconv.ParseBool(req.FormValue("submit"))

	if v.Logout.Submit {
		err := v.app.Repository.Session.Delete(req.Context(), v.Session.UserID)
		if err != nil {
			slog.Error("could not delete session", "err", err)
		}

		cookie := http.Cookie{}
		cookie.Secure = true
		cookie.HttpOnly = true
		cookie.Expires = time.Now()
		cookie.MaxAge = -1
		cookie.Value = ""
		cookie.Name = cookieName
		http.SetCookie(wr, &cookie)
		v.Session = nil
		v.Logout.OK = true
		v.Message("Log out OK")
		v.Display(wr, req)
		return
	} else {
		v.Display(wr, req)
		return
	}
}
