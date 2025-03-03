// app implements the qrochet application
package app

import "net/http"
import "net/utl"
import "os"
import "log/slog"


import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Qrochet struct {
	app.Compo
	http.Client

	name string
}

/*
func (q *Qrochet) Render() app.UI {
	return app.Div().Body(
		app.H1().Body(
			app.Text("Hello, "),
			app.If(q.name != "", func() app.UI {
				return app.Text(q.name)
			}).Else(func() app.UI {
				return app.Text("World!")
			}),
		),
		app.P().Body(
			app.Input().
				Type("text").
				Value(q.name).
				Placeholder("What is your name?").
				AutoFocus(true).
				OnChange(q.ValueTo(&q.name)),
		),
	)
}*/

// The Render method is where the component appearance is defined. Here, a
// "Hello World!" is displayed as a heading.
func (q *Qrochet) Render() app.UI {
	return app.H1().Text("Hello Qrochet!")
}

func ListenAndServe() {
	// Components routing:
	app.Route("/", func() app.Composer { return &Qrochet{} })
	app.Route("/qrochet", func() app.Composer { return &Qrochet{} })
	app.RunWhenOnBrowser()

	// HTTP routing:
	http.Handle("/", &app.Handler{
		Name:        "Qrochet",
		Description: "Qrochet handler",
	})

	port := ":9637"
	slog.Info("Qtarting Qrochet on port", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("ListenAndServe", "err", err)
		os.Exit(1)
	}
}
