package app

import "net/http"
import "net/mail"
import "strconv"
import "log/slog"
import "math/rand/v2"

import "github.com/qrochet/qrochet/pkg/model"
import mailer "github.com/qrochet/qrochet/pkg/mail"
import "github.com/oklog/ulid/v2"

var stupidCAPTCHA = []struct {
	q string
	a int
}{
	{q: "Two plus three is?",
		a: 5},
	{q: "Nine minus seven is?",
		a: 2},
	{q: "Three times three?",
		a: 9},
	{q: "Six divided by two?",
		a: 3},
	{q: "One times one?",
		a: 1},
	{q: "Lucky seven?",
		a: 7},
	{q: "Two pairs?",
		a: 4},
	{q: "Next number after five?",
		a: 6},
	{q: "Two times two times two?",
		a: 8},
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

func (r *register) regenerate() {
	r.CAPTCHA = 0
	r.CAPTCHAI = rand.IntN(len(stupidCAPTCHA))
	r.CAPTCHAQ = stupidCAPTCHA[r.CAPTCHAI].q
}

const mpfMaxMemory = 0

func (q *Qrochet) sendRegistrationMail(user model.User) error {
	if q.msrv == nil {
		slog.Warn("Mailer not available will not send registration email.")
		return nil
	}

	msg := mailer.Mail{}
	msg.To = user.Name + "<" + user.Email + ">"

	msg.Printf("Dear %s, welcome to Qrochet\n\n", user.Name)
	msg.Println("Thank you for registering with Qrochet, the website for chroochet and hand crafts.")
	msg.Println("This is a message to confirm your registration. You do not have to reply to it.")
	msg.Println("Kind regards, Qrochet.")

	err := q.msrv.Send(msg)
	if err != nil {
		slog.Error("While sending mail", "err", err)
		return err
	}
	return nil
}

func (q *Qrochet) register(wr http.ResponseWriter, req *http.Request) {
	var err error

	v := q.view()
	if v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Already logged in.")
		return
	}

	slog.Info("register")
	if req.Method == "POST" {
		err = req.ParseMultipartForm(mpfMaxMemory)
		if err != nil {
			slog.Error("register req.ParseForm", "err", err)
			v.DisplayError(wr, req, "Form error.")
			return
		}
	}
	v.Register.Name = req.FormValue("name")
	v.Register.Email = req.FormValue("email")
	v.Register.Pass = req.FormValue("pass")
	v.Register.Submit, _ = strconv.ParseBool(req.FormValue("submit"))
	v.Register.CAPTCHA, _ = strconv.Atoi(req.FormValue("captcha"))
	v.Register.CAPTCHAI, _ = strconv.Atoi(req.FormValue("captchai"))

	if v.Register.Submit {
		_, err = mail.ParseAddress(v.Register.Email)
		if err != nil {
			slog.Error("mail.ParseAddress", "err", err, "v.Register.Email", v.Register.Email)
			v.Register.regenerate()
			v.DisplayError(wr, req, "Email is not valid.")
			return
		}

		if v.Register.CAPTCHAI < 0 || v.Register.CAPTCHAI >= len(stupidCAPTCHA) ||
			stupidCAPTCHA[v.Register.CAPTCHAI].a != v.Register.CAPTCHA {
			slog.Error("Register CAPTCHA not correct", "captchai", v.Register.CAPTCHAI, "captcha", v.Register.CAPTCHA)
			v.Register.regenerate()
			v.DisplayError(wr, req, "Question answer not correct. Please check again.")
			return
		}

		user := model.User{}
		user.ID = ulid.Make().String()
		user.Email = v.Register.Email
		user.Name = v.Register.Name
		user.SetPassword(v.Register.Pass)

		existing, err := v.app.Repository.User().GetByEmail(req.Context(), user.Email)
		if err != nil {
			slog.Error("User.GetByEmail", "err", err)
			v.Register.regenerate()
			v.DisplayError(wr, req, "Error getting user by email address")
			return
		}

		if existing != nil && existing.Email == user.Email {
			slog.Error("User.GetByEmail", "err", err)
			v.Register.regenerate()
			v.DisplayError(wr, req, "This email address is already registered")
			return
		}

		created, err := v.app.Repository.User().Put(req.Context(), user.ID, user)
		if err != nil {
			slog.Error("User.Put", "err", err)
			v.Register.regenerate()
			v.DisplayError(wr, req, "Registration failed")
			return
		}
		err = v.newSession(wr, req, created)
		if err != nil {
			slog.Error("User.Put", "err", err)
			v.Register.regenerate()
			v.DisplayError(wr, req, "Session creation failed")
			return
		}
		go q.sendRegistrationMail(created)

		v.Message("Registration OK")
		v.Register.OK = true
		v.Display(wr, req)
		return
	} else {
		slog.Error("Not submitted?", "form", req.Form, "post", req.PostForm)
		v.Register.regenerate()
		v.Display(wr, req)
		return
	}
}
