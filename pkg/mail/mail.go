// package mail allows sending maile over SMTP services.
// This can work with a free SMTP service as well.
package mail

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
)

import (
	"github.com/qrochet/qrochet/pkg/model"
)

// Server is a n SMTP server that can send emails.
type Server struct {
	Name      string // Name is a host:port combination.
	User      string // User is the name of the user of the server.
	Pass      string // Password is the password of the user.
	Host      string
	Port      string
	TLSConfig *tls.Config // TLSConfig is the TLS / SSL configuration.
}

// NewServer returns a new mail server.
func NewServer(name, user, pass string) *Server {
	s := &Server{Name: name, User: user, Pass: pass}
	host, port, _ := net.SplitHostPort(s.Name)
	s.Host = host
	s.Port = port
	s.TLSConfig = &tls.Config{
		ServerName: s.Host,
	}
	return s
}

// Mail is an alias to model.Mail.
type Mail = model.Mail

// Send sends a mail using the Server, or returns an error on failure.
func (s Server) Send(m Mail) error {
	if m.From == "" {
		m.From = "Do Not Reply <no-reply@" + s.Host + ">"
	}

	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return err
	}
	subj := m.Subject
	body := m.Body

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server

	auth := smtp.PlainAuth("", s.User, s.Pass, s.Host)

	c, err := smtp.Dial(s.Name)
	if err != nil {
		return err
	}

	defer c.Quit()
	defer c.Close()

	tls, _ := c.Extension("STARTTLS")

	if tls {
		err = c.StartTLS(s.TLSConfig)
		if err != nil {
			return err
		}
	}

	// Auth
	if err = c.Auth(auth); err != nil {
		slog.Error("Mail authentication failed", "err", err, "u", s.User)
		return err
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		return err
	}

	if err = c.Rcpt(to.Address); err != nil {
		return err
	}

	// Data
	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	slog.Info("Sent mail to ", "to", to)

	return nil
}
