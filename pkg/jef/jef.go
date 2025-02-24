// package jef contains a JSON Expression And Filter.
// It allows filtering and evanualtion expressions on streams of JSON files.
// It is based on Josh Baker's gjson, sjson and expr libraries.
package jef

import "os"
import "math"
import "github.com/tidwall/expr"
import "github.com/tidwall/gjson"

// Value is an expr value
type Value = expr.Value

// Result is a result of a gjson parse
type Result = gjson.Result

func result2Value(r Result) Value {
	if r.IsArray() {
		values := results2Values(r.Array())
		return expr.Array(values)
	} else if r.IsObject() {
		return expr.Object(r)
	}

	switch r.Type {
	case gjson.Null:
		return expr.Null
	case gjson.False:
		return expr.Bool(false)
	case gjson.True:
		return expr.Bool(true)
	case gjson.Number:
		if math.Trunc(r.Num) != r.Num {
			return expr.Float64(r.Num)
		} else {
			return expr.Int64(r.Int())
		}
	case gjson.String:
		return expr.String(r.Str)
	case gjson.JSON:
		return expr.Object(r.Raw)
	default:
		return expr.Undefined
	}
}

func results2Values(results []Result) []Value {
	values := make([]Value, len(results))
	for i, r := range results {
		values[i] = result2Value(r)
	}
	return values
}

func (j *Jef) ref(info expr.RefInfo, ctx *expr.Context) (expr.Value, error) {
	r := j.gr.Get(info.Ident)
	if r.Type != gjson.Null {
		return result2Value(r), nil
	}

	val, ok := os.LookupEnv(info.Ident)
	if !ok {
		return expr.Undefined, nil
	}
	return expr.String(val), nil
}

func (j *Jef) call(info expr.CallInfo, ctx *expr.Context) (expr.Value, error) {
	return expr.Undefined, nil
}

func (j *Jef) op(info expr.OpInfo, ctx *expr.Context) (expr.Value, error) {
	return expr.Undefined, nil
}

type Jef struct {
	*expr.Context
	gr gjson.Result
}

func New() *Jef {
	j := &Jef{}
	j.Context = &expr.Context{}
	j.Context.Extender = expr.NewExtender(j.ref, j.call, j.op)
	return j
}

// EvalString evaluates a string in the context of Jef.
func (j *Jef) EvalString(ex string) (expr.Value, error) {
	res, err := expr.Eval(ex, j.Context)
	return res, err
}

// EvalJSON evaluates expr with the given byte arrays, which should be in JSON
// format as the data based on the expression ex.
func (j *Jef) EvalJSON(ex string, js []byte) (Value, error) {
	j.gr = gjson.ParseBytes(js)
	res, err := expr.Eval(ex, j.Context)
	return res, err
}

// Filter filters the channel of byte arrays, which should be in JSON format
// based on the expression ex.
func (j *Jef) Filter(ex string, jsons chan ([]byte)) (chan []byte, error) {
	return nil, nil
}

// Map transforms the channel of byte arrays, which should be in JSON format
// based on the expression ex.
func (j *Jef) Map(ex string, jsons chan ([]byte)) (chan []byte, error) {
	return nil, nil
}
