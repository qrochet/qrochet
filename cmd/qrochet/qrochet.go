package main

import "flag"
import "os"
import "log/slog"
import "context"

// import "os/signal"

import "github.com/qrochet/qrochet/pkg/app"

func main() {
	var nats, addr string
	flag.StringVar(&nats, "n", os.Getenv("QROCHET_NATS"), "nats server to connect to, or nats+builtin:///path for a built in NATS server.")
	flag.StringVar(&addr, "a", os.Getenv("QROCHET_ADDR"), "address to listen on")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q, err := app.New(ctx, addr, nats)
	if err != nil {
		slog.Error("app.New", "err", err)
		os.Exit(2)
	}

	defer q.Close()

	// ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	// defer stop()

	q.ListenAndServe(ctx)
}
