package main

import "flag"
import "fmt"
import "os"
import "io"
import "log/slog"
import "log/syslog"
import "context"

// import "os/signal"

import "aidanwoods.dev/go-paseto"

import "github.com/qrochet/qrochet/pkg/app"
import "github.com/qrochet/qrochet/pkg/env"

func setupSlog(level slog.Level, format, output, tag string) {
	// Determine the log format
	var handler slog.Handler
	var out io.Writer = os.Stdout

	// Determine the log output
	if output == "syslog" {
		syslogLogger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, tag)
		if err != nil {
			panic("Failed to connect to syslog:" + err.Error())
		}
		out = syslogLogger
	} else if output != "" {
		file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic("Failed to open log file:" + err.Error())
		}
		out = file
	}

	switch format {
	case "json", "JSON":
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level})
	case "text", "TEXT":
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				var zero slog.Attr
				if a.Key == "time" {
					return zero
				}
				return a
			},
		})
	}
	// Create the logger
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func key() {
	key := paseto.NewV4SymmetricKey()
	fmt.Printf("Generated PASETO key:\nexport QROCHET_PASETO=%s\n", key.ExportHex())
	os.Exit(0)
}

func main() {
	envErr := env.Read()

	var level = slog.LevelInfo
	var format = env.String("SLOG_FORMAT", "text")
	var output = env.String("SLOG_OUTPUT", "")

	var set app.Settings
	flag.StringVar(&set.NATS, "n", env.String("QROCHET_NATS"), "QROCHET_NATS\tnats server to connect to, or nats+builtin:///path for a built in NATS server.")
	flag.StringVar(&set.Addr, "a", env.String("QROCHET_ADDR"), "QROCHET_ADDR\taddress to listen on")
	flag.StringVar(&set.Key, "k", env.String("QROCHET_PASETO"), "QROCHET_PASETO\tPASETO private key")
	flag.BoolVar(&set.Dev, "D", env.Bool("QROCHET_DEV"), "QROCHET_DEV\tset to true to enable dev mode and use local resources.")
	flag.TextVar(&level, "L", slog.LevelInfo, "log level to use")
	flag.Parse()

	if len(flag.Args()) > 0 && flag.Args()[0] == "key" {
		key()
	}

	setupSlog(level, format, output, "qrochet")
	slog.Info("slog set up", "level", level, "format", format, "output", output)
	if envErr != nil {
		slog.Warn("could not read .env file", "err", envErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q, err := app.New(ctx, set)
	if err != nil {
		slog.Error("app.New", "err", err)
		os.Exit(2)
	}

	defer q.Close()

	// ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	// defer stop()

	q.ListenAndServe(ctx)
}
