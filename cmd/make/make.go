package main

import "os/exec"
import "os"

import "path/filepath"
import "log/slog"
import "strings"

func outdated(target string, sources ...string) bool {
	stat, err := os.Stat(target)
	if err != nil {
		return true
	}
	for _, source := range sources {
		matches, err := filepath.Glob(source)
		if err != nil {
			slog.Error("outdated", err, err)
			return true
		}
		for _, match := range matches {
			matchStat, err := os.Stat(match)
			if err != nil {
				slog.Error("outdated", err, err)
				return true
			}
			if stat.ModTime().Before(matchStat.ModTime()) {
				return true
			}
		}
	}
	return false
}

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
	if outdated("web/app.wasm", "pkg/*/*.go", "cmd/*/*.go") {
		runEnv([]string{"GOARCH=wasm", "GOOS=js"}, "go", "build", "-o", "web/app.wasm", "./cmd/qrochet")
		run("go", "build", "./cmd/qrochet")
	} else {
		slog.Info("build skipped, all targets up to date")
	}
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
