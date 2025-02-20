// the jef tool allows to filter JSON files using gjson and
// tidwall/expr expressions.
package main

import "os"
import "fmt"

import "github.com/tidwall/expr"

func ref(info expr.RefInfo, ctx *expr.Context) (expr.Value, error) {
	val, ok := os.LookupEnv(info.Ident)
	if !ok {
		return expr.Undefined, nil
	}
	return expr.String(val), nil
}

func call(info expr.CallInfo, ctx *expr.Context) (expr.Value, error) {
	return expr.Undefined, nil
}

func op(info expr.OpInfo, ctx *expr.Context) (expr.Value, error) {
	return expr.Undefined, nil
}

func main() {
	ctx := &expr.Context{}
	ctx.Extender = expr.NewExtender(ref, call, op)
	if len(os.Args) > 1 {
		res, err := expr.Eval(os.Args[1], ctx)
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
