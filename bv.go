package main

import (
	"fmt"
)

type EvalContext struct {
	vars []uint64
}

type EvalFunc func(ctx *EvalContext) uint64

func compile2(b []byte) (EvalFunc, EvalFunc, int) {
	e1, i := Compile(b)
	e2, j := Compile(b[i:])
	return e1, e2, i + j
}

func compile3(b []byte) (EvalFunc, EvalFunc, EvalFunc, int) {
	e1, i := Compile(b)
	e2, e3, j := compile2(b[i:])
	return e1, e2, e3, i + j
}

func Compile(b []byte) (EvalFunc, int) {

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
		e, i := Compile(b[1:])
		return func(c *EvalContext) uint64 {
			return ^e(c)
		}, 1 + i
	case 'l':
		e, i := Compile(b[1:])
		return func(c *EvalContext) uint64 {
			return e(c) << 1
		}, 1 + i
	case 'r':
		e, i := Compile(b[1:])
		return func(c *EvalContext) uint64 {
			return e(c) >> 1
		}, 1 + i
	case 'q':
		e, i := Compile(b[1:])
		return func(c *EvalContext) uint64 { return e(c) >> 4 }, 1 + i
	case 'h':
		e, i := Compile(b[1:])
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

func ProgramToString(b []byte) (string, int) {
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
		s1, i := ProgramToString(b[1:])
		s2, j := ProgramToString(b[1+i:])
		s3, k := ProgramToString(b[1+i+j:])
		return fmt.Sprintf("(if0 %s %s %s)", s1, s2, s3), 1 + i + j + k
	case 'f':
		s1, i := ProgramToString(b[1:])
		s2, j := ProgramToString(b[1+i:])
		s3, k := ProgramToString(b[1+i+j:])
		return fmt.Sprintf("(fold %s %s (lambda (y z) )%s)", s1, s2, s3), 1 + i + j + k
	case 'n':
		s, i := ProgramToString(b[1:])
		return fmt.Sprintf("(not %s)", s), 1 + i
	case 'l':
		s, i := ProgramToString(b[1:])
		return fmt.Sprintf("(shl1 %s)", s), 1 + i
	case 'r':
		s, i := ProgramToString(b[1:])
		return fmt.Sprintf("(shr1 %s)", s), 1 + i
	case 'q':
		s, i := ProgramToString(b[1:])
		return fmt.Sprintf("(shr4 %s)", s), 1 + i
	case 'h':
		s, i := ProgramToString(b[1:])
		return fmt.Sprintf("(shr16 %s)", s), 1 + i
	case 'a':
		s1, i := ProgramToString(b[1:])
		s2, j := ProgramToString(b[1+i:])
		return fmt.Sprintf("(and %s %s)", s1, s2), 1 + i + j
	case 'o':
		s1, i := ProgramToString(b[1:])
		s2, j := ProgramToString(b[1+i:])
		return fmt.Sprintf("(or %s %s)", s1, s2), 1 + i + j
	case 'x':
		s1, i := ProgramToString(b[1:])
		s2, j := ProgramToString(b[1+i:])
		return fmt.Sprintf("(xor %s %s)", s1, s2), 1 + i + j
	case 'p':
		s1, i := ProgramToString(b[1:])
		s2, j := ProgramToString(b[1+i:])
		return fmt.Sprintf("(plus %s %s)", s1, s2), 1 + i + j
	default:
		panic(b[0])
	}
}
