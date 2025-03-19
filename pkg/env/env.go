// package env is package to get environment variabels in at type safe way.
package env

import "os"
import "strconv"
import "bufio"
import "strings"
import "path/filepath"

// ReadWithSetter reads a .env file and cals the setter with the keey value pairs.
// An env file may only have # comments, empty lines
// or assignments of the form k=v and sets the assignments into the environment.
func ReadWithSetter(name string, setter func(key, value string) error) (err error) {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		line = strings.TrimLeft(line, " \t")
		if strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		err := setter(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadEnv is a shorthand for ReadWithSetter(name, os.Setenv).
func ReadEnv(name string) (err error) {
	return ReadWithSetter(name, os.Setenv)
}

// Read is a shorthand for ReadWithSetter(filepath.Join(os.Getwd, ".env"), os.Setenv).
func Read() (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	name := filepath.Join(cwd, ".env")
	return ReadWithSetter(name, os.Setenv)
}

// Env gets an environment variable of type T.
// If otherwise given otherwise[0] is returned if the env variable is not found.
// IF otherwise is not given and the env variable is not found this function
// returns the zero value of T
func Env[T any](convert func(s string) (T, error), name string, otherwise ...T) T {
	var zero T

	res, err := convert(os.Getenv(name))
	if err != nil {
		if len(otherwise) > 0 {
			return otherwise[0]
		}
		return zero
	}
	return res
}

func Bool(name string, otherwise ...bool) bool {
	return Env(strconv.ParseBool, name, otherwise...)
}

func Int(name string, otherwise ...int) int {
	return Env(strconv.Atoi, name, otherwise...)
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 0, 64)
}

func Int64(name string, otherwise ...int64) int64 {
	return Env(parseInt64, name, otherwise...)
}

func parseString(s string) (string, error) {
	return s, nil
}

func String(name string, otherwise ...string) string {
	return Env(parseString, name, otherwise...)
}
