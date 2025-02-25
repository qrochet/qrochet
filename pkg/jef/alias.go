package jef

import "github.com/tidwall/expr"

var (
	Undefined = expr.Undefined
	Null      = expr.Null
)

// ErrStop is used to stop the EvalForEach and ForEachValue
var ErrStop = expr.ErrStop

// CharPosOfErr returns the character position of where the error occured in
// the Eval function, or -1 if unknown
var CharPosOfErr func(err error) int = expr.CharPosOfErr

type CallInfo = expr.CallInfo

type Context = expr.Context

type Extender = expr.Extender

// NewExtender is a convenience function for creating a simple extender using
// the provided eval and op functions.

var NewExtender func(
	ref func(info RefInfo, ctx *Context) (Value, error),
	call func(info CallInfo, ctx *Context) (Value, error),
	op func(info OpInfo, ctx *Context) (Value, error),
) Extender = expr.NewExtender

// Op is an operator for Custom values used for the Options.Op function.
type Op = expr.Op

const (
	OpAdd    = expr.OpAdd
	OpSub    = expr.OpSub
	OpMul    = expr.OpMul
	OpDiv    = expr.OpDiv
	OpMod    = expr.OpMod
	OpLt     = expr.OpLt
	OpStEq   = expr.OpStEq
	OpAnd    = expr.OpAnd
	OpOr     = expr.OpOr
	OpBitOr  = expr.OpBitOr
	OpBitXor = expr.OpBitXor
	OpBitAnd = expr.OpBitAnd
	OpCoal   = expr.OpCoal
)

type OpInfo = expr.OpInfo
type RefInfo = expr.RefInfo

// Value represents is the return value of Eval.
type Value = expr.Value

// Array return an array value.
var Array func(values []Value) Value = expr.Array

// Bool returns a bool value.
var Bool func(t bool) Value = expr.Bool

// Eval evaluates an expression and returns the Result.
var Eval func(expr string, ctx *Context) (Value, error) = expr.Eval

// EvalForEach iterates over a series of comma delimited expressions. The last
// value in the series is returned. Returning ErrStop will stop the iteration
// early and return the last known value and nil as an error. Returning any
// other error from iter will stop the iteration and return the same error.
var EvalForEach func(expr string, iter func(value Value) error, ctx *Context,
) (Value, error) = expr.EvalForEach

// Float64 returns an int64 value.
var Float64 func(x float64) Value = expr.Float64

// Function returnsz a function value.
var Function func(name string) Value = expr.Function

// Int64 returns an int64 value.
var Int64 func(x int64) Value = expr.Int64

// Number returns a float64 value.
var Number func(x float64) Value = expr.Number

// Object returns a custom user-defined object.
var Object func(o interface{}) Value = expr.Object

// String returns a string value.
var String func(s string) Value = expr.String

// Uint64 returns a uint64 value.
var Uint64 func(x uint64) Value = expr.Uint64
