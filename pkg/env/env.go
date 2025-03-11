// package env is package to get environment variabels in at type safe way.
package env

import "os"
import "strconv"

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
