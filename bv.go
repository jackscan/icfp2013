package main

import (
	"bytes"
	"fmt"
	"math"
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
		panic(fmt.Sprintf("compile: |%s| == %d != %d", string(b), len(b), size))
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

type AnalyzeContext struct {
	b             []byte
	ignoreMask    []byte
	yZeros, yOnes uint64
}

func AnalyzeFunc(fn []byte) []byte {
	var ac AnalyzeContext
	ac.b = fn
	ac.ignoreMask = make([]byte, len(fn))
	for i, _ := range ac.b {
		ac.ignoreMask[i] = '.'
	}

	// fmt.Printf("fun: %s\n", string(fn))
	_, _, s := analyze(&ac, 0, 0)
	//fmt.Printf("0: %v, 1: %v, %s, %s\n", z, n, string(fn), string(ac.ignoreMask))

	if s != len(fn) {
		panic(fmt.Errorf("analyze: |%s| == %d != %d", string(fn), len(fn), s))
	}

	return ac.ignoreMask
}

func analyze(c *AnalyzeContext, index int, ignore uint64) (zeros, ones uint64, size int) {

	next := index + 1
	size = 0

	if ignore == math.MaxUint64 {
		c.ignoreMask[index] = 'i'
	}

	switch c.b[index] {
	case 'X':
		zeros, ones = 0, 0
	case 'Y':
		zeros, ones = c.yZeros, c.yOnes
	case 'Z':
		zeros, ones = 0, 0
	case '0':
		zeros, ones = math.MaxUint64, 0
	case '1':
		zeros, ones = ^uint64(1), 1
	case 'i':
		i1 := ignore
		i2 := ignore
		z0, n0, s0 := analyze(c, next, 0)
		if n0 != 0 {
			i1 = math.MaxUint64
		} else if z0 == math.MaxUint64 {
			i2 = math.MaxUint64
		}
		z1, n1, s1 := analyze(c, next+s0, i1)
		z2, n2, s2 := analyze(c, next+s0+s1, i2)

		if z0 == math.MaxUint64 {
			zeros = z1
			ones = n1
		} else if n0 != 0 {
			zeros = z2
			ones = n2
		} else {
			zeros = z1 & z2
			ones = n1 & n2
		}

		size = s0 + s1 + s2

	case 'f':
		z0, n0, s0 := analyze(c, next, 0)
		_, _, s1 := analyze(c, next+s0, 0)

		c.yZeros, c.yOnes = 0, 0
		for i := 0; i < 8; i++ {
			c.yZeros &= (z0 >> uint(i*8)) & uint64(0xFF)
			c.yOnes &= (n0 >> uint(i*8)) & uint64(0xFF)
		}
		c.yZeros |= ^uint64(0xFF)

		_, _, s2 := analyze(c, next+s0+s1, 0)
		zeros, ones = 0, 0
		size = s0 + s1 + s2
	case 'n':
		ones, zeros, size = analyze(c, next, ignore)
	case 'l':
		zeros, ones, size = analyze(c, next, (uint64(1)<<63)|(ignore>>1))
		zeros = (zeros << 1) | 1
		ones = (ones << 1)
	case 'r':
		zeros, ones, size = analyze(c, next, uint64(1)|(ignore<<1))
		zeros = (zeros >> 1) | (uint64(1) << 63)
		ones = (ones >> 1)
	case 'q':
		zeros, ones, size = analyze(c, next, uint64(0xF)|(ignore<<4))
		zeros = (zeros >> 4) | (uint64(0xF) << 60)
		ones = (ones >> 4)
	case 'h':
		zeros, ones, size = analyze(c, next, uint64(0xFFFF)|(ignore<<16))
		zeros = (zeros >> 16) | (uint64(0xFFFF) << 48)
		ones = (ones >> 16)
	case 'a':
		z0, n0, s0 := analyze(c, next, ignore)
		z1, n1, s1 := analyze(c, next+s0, z0|ignore)
		zeros = z0 | z1
		ones = n0 & n1
		size = s0 + s1
	case 'o':
		z0, n0, s0 := analyze(c, next, ignore)
		z1, n1, s1 := analyze(c, next+s0, n0|ignore)
		zeros = z0 & z1
		ones = n0 | n1
		size = s0 + s1
	case 'x':
		z0, n0, s0 := analyze(c, next, ignore)
		z1, n1, s1 := analyze(c, next+s0, ignore)
		zeros = (z0 & z1) | (n0 & n1)
		ones = (z0 & n1) | (n0 & z1)
		size = s0 + s1
	case 'p':
		_, _, s0 := analyze(c, next, ignore)
		_, _, s1 := analyze(c, next+s0, ignore)
		// TODO:
		zeros, ones = 0, 0
		size = s0 + s1
	default:
		panic(string(c.b[index:]))
	}

	if (zeros & ones) != 0 {
		panic(fmt.Errorf("overlap: %X, %X\n"))
	}

	size++

	//fmt.Printf("%s: z: %X, n: %X\n", c.b[index:index+size], zeros, ones)

	return
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

var OpMap = map[string]struct{ sym, arity byte }{
	"if0":   {'i', 3},
	"fold":  {'f', 3},
	"tfold": {'t', 3},
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
