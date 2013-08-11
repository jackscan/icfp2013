package main

import (
	"bytes"
	"fmt"
)

type EvalContext struct {
	vars []uint64
}

type EvalFunc func(ctx *EvalContext) uint64

func compile2(b []byte) (EvalFunc, EvalFunc, int) {
	e1, i := compile1(b)
	e2, j := compile1(b[i:])
	return e1, e2, i + j
}

func compile3(b []byte) (EvalFunc, EvalFunc, EvalFunc, int) {
	e1, i := compile1(b)
	e2, e3, j := compile2(b[i:])
	return e1, e2, e3, i + j
}

func Compile(b []byte) EvalFunc {
	fn, size := compile1(b)
	if size != len(b) {
		panic(fmt.Sprintf("|%s| == %d != %d", string(b), len(b), size))
	}
	return fn
}

func compile1(b []byte) (EvalFunc, int) {

	switch b[0] {
	case 'X':
		return func(c *EvalContext) uint64 { return c.vars[0] }, 1
	case 'Y':
		return func(c *EvalContext) uint64 { return c.vars[1] }, 1
	case 'Z':
		return func(c *EvalContext) uint64 { return c.vars[2] }, 1
	case '0':
		return func(c *EvalContext) uint64 { return 0 }, 1
	case '1':
		return func(c *EvalContext) uint64 { return 1 }, 1
	case 'i':
		e1, e2, e3, i := compile3(b[1:])
		return func(c *EvalContext) uint64 {
			if e1(c) == 0 {
				return e2(c)
			} else {
				return e3(c)
			}
		}, 1 + i
	case 'f':
		e1, e2, e3, i := compile3(b[1:])
		return func(c *EvalContext) uint64 {
			y := e1(c)
			z := e2(c)
			if len(c.vars) > 1 {
				panic(len(c.vars))
			}
			c.vars = append(c.vars, y, z)
			for i := uint64(0); i < uint64(8); i++ {
				c.vars[1] = (y >> (i * uint64(8))) & uint64(0xFF)
				c.vars[2] = e3(c)
			}
			z = c.vars[2]
			c.vars = c.vars[:1]
			return z
		}, 1 + i

	case 'n':
		e, i := compile1(b[1:])
		return func(c *EvalContext) uint64 {
			return ^e(c)
		}, 1 + i
	case 'l':
		e, i := compile1(b[1:])
		return func(c *EvalContext) uint64 {
			return e(c) << 1
		}, 1 + i
	case 'r':
		e, i := compile1(b[1:])
		return func(c *EvalContext) uint64 {
			return e(c) >> 1
		}, 1 + i
	case 'q':
		e, i := compile1(b[1:])
		return func(c *EvalContext) uint64 { return e(c) >> 4 }, 1 + i
	case 'h':
		e, i := compile1(b[1:])
		return func(c *EvalContext) uint64 { return e(c) >> 16 }, 1 + i
	case 'a':
		e1, e2, i := compile2(b[1:])
		return func(c *EvalContext) uint64 { return e1(c) & e2(c) }, 1 + i
	case 'o':
		e1, e2, i := compile2(b[1:])
		return func(c *EvalContext) uint64 { return e1(c) | e2(c) }, 1 + i
	case 'x':
		e1, e2, i := compile2(b[1:])
		return func(c *EvalContext) uint64 { return e1(c) ^ e2(c) }, 1 + i
	case 'p':
		e1, e2, i := compile2(b[1:])
		return func(c *EvalContext) uint64 { return e1(c) + e2(c) }, 1 + i
	default:
		panic(b[0])
	}
}

func HasOp(b []byte, op byte) bool {
	return bytes.IndexByte(b, op) >= 0
}

func HasFold(b []byte) bool {
	return HasOp(b, 'f')
}

func HasIf0(b []byte) bool {
	return HasOp(b, 'i')
}

func InFold(b []byte) (result bool) {
	result = false
	i := bytes.IndexByte(b, 'f')
	if i >= 0 {
		// what a hack!
		defer func() { recover() }()
		_, _, j := compile2(b[1+i:])
		result = true
		compile1(b[1+i+j:])
	}
	return false
}

var OpMap = map[string]struct{ sym, arity byte }{
	"if0":   {'i', 3},
	"fold":  {'f', 3},
	"tfold": {'t', 4},
	"not":   {'n', 1},
	"shl1":  {'l', 1},
	"shr1":  {'r', 1},
	"shr4":  {'q', 1},
	"shr16": {'h', 1},
	"and":   {'a', 2},
	"or":    {'o', 2},
	"xor":   {'x', 2},
	"plus":  {'p', 2},
}

func ProgramToString(b []byte) string {
	str, size := programToStringInternal(b)
	if size != len(b) {
		panic(fmt.Sprintf("|%s| == %d != %d", string(b), len(b), size))
	}
	return fmt.Sprintf("(lambda (x) %s)", str)
}

func programToStringInternal(b []byte) (string, int) {
	switch b[0] {
	case 'X':
		return "x", 1
	case 'Y':
		return "y", 1
	case 'Z':
		return "z", 1
	case '0':
		return "0", 1
	case '1':
		return "1", 1
	case 'i':
		s1, i := programToStringInternal(b[1:])
		s2, j := programToStringInternal(b[1+i:])
		s3, k := programToStringInternal(b[1+i+j:])
		return fmt.Sprintf("(if0 %s %s %s)", s1, s2, s3), 1 + i + j + k
	case 'f':
		s1, i := programToStringInternal(b[1:])
		s2, j := programToStringInternal(b[1+i:])
		s3, k := programToStringInternal(b[1+i+j:])
		return fmt.Sprintf("(fold %s %s (lambda (y z) %s))", s1, s2, s3), 1 + i + j + k
	case 'n':
		s, i := programToStringInternal(b[1:])
		return fmt.Sprintf("(not %s)", s), 1 + i
	case 'l':
		s, i := programToStringInternal(b[1:])
		return fmt.Sprintf("(shl1 %s)", s), 1 + i
	case 'r':
		s, i := programToStringInternal(b[1:])
		return fmt.Sprintf("(shr1 %s)", s), 1 + i
	case 'q':
		s, i := programToStringInternal(b[1:])
		return fmt.Sprintf("(shr4 %s)", s), 1 + i
	case 'h':
		s, i := programToStringInternal(b[1:])
		return fmt.Sprintf("(shr16 %s)", s), 1 + i
	case 'a':
		s1, i := programToStringInternal(b[1:])
		s2, j := programToStringInternal(b[1+i:])
		return fmt.Sprintf("(and %s %s)", s1, s2), 1 + i + j
	case 'o':
		s1, i := programToStringInternal(b[1:])
		s2, j := programToStringInternal(b[1+i:])
		return fmt.Sprintf("(or %s %s)", s1, s2), 1 + i + j
	case 'x':
		s1, i := programToStringInternal(b[1:])
		s2, j := programToStringInternal(b[1+i:])
		return fmt.Sprintf("(xor %s %s)", s1, s2), 1 + i + j
	case 'p':
		s1, i := programToStringInternal(b[1:])
		s2, j := programToStringInternal(b[1+i:])
		return fmt.Sprintf("(plus %s %s)", s1, s2), 1 + i + j
	default:
		panic(string(b))
	}
}
