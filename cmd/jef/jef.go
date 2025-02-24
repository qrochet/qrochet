// the jef tool allows to filter JSON files using gjson and
// tidwall/expr expressions.
package main

import "os"
import "fmt"

import "github.com/qrochet/qrochet/pkg/jef"

func main() {
	j := jef.New()
	if len(os.Args) > 2 {
		res, err := j.EvalJSON(os.Args[1], []byte(os.Args[2]))
		if err == nil {
			fmt.Fprintf(os.Stdout, "Result: %v\n", res)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	} else if len(os.Args) > 1 {
		res, err := j.EvalString(os.Args[1])
		if err == nil {
			fmt.Fprintf(os.Stdout, "Result: %v\n", res)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

	} else {
		fmt.Fprintf(os.Stderr, "jef <expression>\n")
		os.Exit(2)
	}
}
