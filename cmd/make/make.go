package main

import "os/exec"
import "os"
import "log/slog"
import "strings"

func runEnv(env []string, command string, args ...string) {
	cmd := exec.Command(command, args...)
	if cmd.Err != nil {
		slog.Error("run", "err", cmd.Err, "command", command)
		os.Exit(1)
	}
	if env != nil {
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, env...)
	}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	slog.Info("runEnv", "cmd", cmd)
	err := cmd.Run()
	if err != nil {
		slog.Error("run", "err", err, "command", command, "stdout", stdout.String(),
			"stderr", stderr.String())
		os.Exit(1)
	}
}

func run(command string, args ...string) {
	runEnv(nil, command, args...)
}

func build() {
	runEnv([]string{"GOARCH=wasm", "GOOS=js"}, "go", "build", "-o", "web/app.wasm", "./cmd/qrochet")
	run("go", "build", "./cmd/qrochet")
}

func qrochet() {
	run("./qrochet")
}

func target() string {
	if len(os.Args) < 2 {
		return ""
	}
	return os.Args[1]
}

func main() {
	switch target() {
	case "build":
		build()
	default:
		build()
		qrochet()
	}
}
