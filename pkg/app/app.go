// app implements the qrochet application
package app

import "net/http"
import "net"
import "os"
import "io/fs"
import "log/slog"
import "context"
import "embed"
import "html/template"
import "strconv"

import (
	"github.com/qrochet/qrochet/pkg/repo"
)

//go:embed web
var resources embed.FS

//go:embed tmpl
var templates embed.FS

type Settings struct {
	NATS string
	Addr string
	Dev  bool
}

type Qrochet struct {
	http.Server
	*http.ServeMux
	*repo.Repository
	*template.Template
	sub fs.FS
}

func New(ctx context.Context, s Settings) (*Qrochet, error) {
	var err error
	q := &Qrochet{}

	q.Template = template.New("")
	q.Template, err = q.Template.ParseFS(templates, "tmpl/*.tmpl.html")
	if err != nil {
		return nil, err
	}

	q.Server.Addr = s.Addr
	q.ServeMux = http.NewServeMux()
	q.Server.Handler = q.ServeMux
	q.sub, err = fs.Sub(resources, "web")
	if err != nil {
		return nil, err
	}

	entries, err := resources.ReadDir("web")
	for _, entry := range entries {
		slog.Info("Available Resource: ", "entry", entry)
	}
	if err != nil {
		slog.Error("readdir", "err", err)
	}

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

type view struct {
	Register struct {
		Name   string
		Email  string
		Submit bool
	}
}

func (q *Qrochet) view() *view {
	return &view{}
}

func (q *Qrochet) index(wr http.ResponseWriter, req *http.Request) {
	view := q.view()
	slog.Info("index")
	err := q.Template.ExecuteTemplate(wr, "index.tmpl.html", view)
	if err != nil {
		slog.Error("index", err)
	}
}

func (q *Qrochet) register(wr http.ResponseWriter, req *http.Request) {
	view := q.view()
	slog.Info("register")
	err := req.ParseForm()
	if err != nil {
		slog.Error("register req.ParseForm", err)
	}
	view.Register.Name = req.FormValue("name")
	view.Register.Email = req.FormValue("email")
	view.Register.Submit, _ = strconv.ParseBool(req.FormValue("submit"))
	err = q.Template.ExecuteTemplate(wr, "register.tmpl.html", view)
	if err != nil {
		slog.Error("register", err)
	}
}

func (q *Qrochet) ListenAndServe(ctx context.Context) {
	// Routing
	q.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	q.ServeMux.HandleFunc("/", q.index)
	q.ServeMux.HandleFunc("/register", q.register)
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
