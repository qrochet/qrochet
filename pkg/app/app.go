// app implements the qrochet application
package app

import "net/http"
import "net"
import "os"
import "io/fs"
import "io"
import "log/slog"
import "context"
import "embed"
import "html/template"
import "aidanwoods.dev/go-paseto"

import (
	"github.com/qrochet/qrochet/pkg/doc"
	"github.com/qrochet/qrochet/pkg/mail"
	"github.com/qrochet/qrochet/pkg/model"
	"github.com/qrochet/qrochet/pkg/repo"
)

var templateFuncs = template.FuncMap{
	"doc": doc.HTML,
}

//go:embed web
var resources embed.FS

//go:embed tmpl
var templates embed.FS

type Template interface {
	Execute(wr io.Writer, data any) error
	ExecuteTemplate(wr io.Writer, name string, data any) error
}

type Settings struct {
	NATS string
	Addr string
	Key  string
	Dev  bool
}

type Qrochet struct {
	http.Server
	*RemoteAddrRateLimiter
	*http.ServeMux
	model.Repository
	*template.Template
	sub  fs.FS
	Key  paseto.V4SymmetricKey
	msrv mail.Sender
}

func New(ctx context.Context, s Settings) (*Qrochet, error) {
	var err error
	q := &Qrochet{}

	if s.Key == "" {
		q.Key = paseto.NewV4SymmetricKey()
	} else {
		q.Key, err = paseto.V4SymmetricKeyFromHex(s.Key)
		if err != nil {
			return nil, err
		}
	}

	q.Template = template.New("").Funcs(templateFuncs)

	if s.Dev {
		q.Template, err = q.Template.ParseGlob("pkg/app/tmpl/*.tmpl.html")
		if err != nil {
			return nil, err
		}
	} else {
		q.Template, err = q.Template.ParseFS(templates, "tmpl/*.tmpl.html")
		if err != nil {
			return nil, err
		}
	}

	q.RemoteAddrRateLimiter = NewRemoteAddrRateLimiter(1, 4)
	q.Server.Addr = s.Addr
	q.ServeMux = http.NewServeMux()
	q.Server.Handler = q.RemoteAddrRateLimiter.Middleware(q.ServeMux)
	if s.Dev {
		q.sub = os.DirFS("pkg/app/web")
	} else {
		q.sub, err = fs.Sub(resources, "web")
		if err != nil {
			return nil, err
		}
	}

	fs.WalkDir(q.sub, "", func(path string, d fs.DirEntry, err error) error {
		slog.Info("Available Resource: ", "path", path, "entry", d)
		return nil
	})

	q.Repository, err = repo.Open(s.NATS)
	if err != nil {
		return nil, err
	}
	slog.Info("NATS connected", "URL", s.NATS)
	return q, nil
}

func (q *Qrochet) Close() {
	slog.Info("qrochet shutting down")
	q.Repository.Close()
	q.Server.Close()
}

// SetMailSender sets the mail sender to use.
// If not set or nil, no mails will be sent.
func (q *Qrochet) SetMailSender(msrv mail.Sender) {
	q.msrv = msrv
	if q.msrv == nil {
		slog.Warn("sending of mails disabled")
	} else {
		slog.Info("sending of mails enabled")
	}
}

func (q *Qrochet) view() *view {
	return &view{app: q}
}

func (q *Qrochet) index(wr http.ResponseWriter, req *http.Request) {
	view := q.view()
	view.check(wr, req)
	slog.Info("index")
	err := q.Template.ExecuteTemplate(wr, "index.tmpl.html", view)
	if err != nil {
		slog.Error("index", err)
	}
}

func (q *Qrochet) ListenAndServe(ctx context.Context) {
	// Routing
	q.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	q.ServeMux.HandleFunc("/", q.index)
	q.ServeMux.HandleFunc("/register", q.register)
	q.ServeMux.HandleFunc("/login", q.login)
	q.ServeMux.HandleFunc("/logout", q.logout)
	q.ServeMux.HandleFunc("GET /my/craft", q.getMyCraft)
	q.ServeMux.HandleFunc("GET /my/crafts", q.getMyCrafts)
	q.ServeMux.HandleFunc("POST /my/craft", q.postMyCraft)
	q.ServeMux.HandleFunc("GET /upload/{id}", q.getUpload)
	q.ServeMux.Handle("/web/",
		http.StripPrefix("/web/", http.FileServer(http.FS(q.sub))),
	)

	defer func() {
		<-ctx.Done()
		slog.Info("qrochet interrupted")
		q.Server.Shutdown(ctx)
	}()

	slog.Info("Starting Qrochet", "addr", q.Server.Addr)
	if err := q.Server.ListenAndServe(); err != nil {
		slog.Error("ListenAndServe", "err", err)
		os.Exit(1)
	}
}
