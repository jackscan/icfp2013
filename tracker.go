package main

import (
	"bytes"
	"fmt"
	"math/rand"
	// "sort"
)

type ProblemContext struct {
	size      int
	length    int
	ops1      []byte
	ops2      []byte
	ops3      []byte
	tfold     bool
	arguments []uint64
	outputs   []uint64
}

type NextSolution func(s []byte) []byte

func processOperators(ops []string) []byte {
	// case 'X':
	// 	return "x", 1
	// case 'Y':
	// 	return "y", 1
	// case 'Z':
	// 	return "z", 1
	// case '0':
	// 	return "0", 1
	// case '1':
	// 	return "1", 1
	// case 'i':
	// 	s1, i := ProgramToString(b[1:])
	// 	s2, j := ProgramToString(b[1+i:])
	// 	s3, k := ProgramToString(b[1+i+j:])
	// 	return fmt.Sprintf("(if0 %s %s %s)", s1, s2, s3), 1 + i + j + k
	// case 'f':
	// 	s1, i := ProgramToString(b[1:])
	// 	s2, j := ProgramToString(b[1+i:])
	// 	s3, k := ProgramToString(b[1+i+j:])
	// 	return fmt.Sprintf("(fold %s %s (lambda (y z) )%s)", s1, s2, s3), 1 + i + j + k
	// case 'n':
	// 	s, i := ProgramToString(b[1:])
	// 	return fmt.Sprintf("(not %s)", s), 1 + i
	// case 'l':
	// 	s, i := ProgramToString(b[1:])
	// 	return fmt.Sprintf("(shl1 %s)", s), 1 + i
	// case 'r':
	// 	s, i := ProgramToString(b[1:])
	// 	return fmt.Sprintf("(shr1 %s)", s), 1 + i
	// case 'q':
	// 	s, i := ProgramToString(b[1:])
	// 	return fmt.Sprintf("(shr4 %s)", s), 1 + i
	// case 'h':
	// 	s, i := ProgramToString(b[1:])
	// 	return fmt.Sprintf("(shr16 %s)", s), 1 + i
	// case 'a':
	// 	s1, i := ProgramToString(b[1:])
	// 	s2, j := ProgramToString(b[1+i:])
	// 	return fmt.Sprintf("(and %s %s)", s1, s2), 1 + i + j
	// case 'o':
	// 	s1, i := ProgramToString(b[1:])
	// 	s2, j := ProgramToString(b[1+i:])
	// 	return fmt.Sprintf("(or %s %s)", s1, s2), 1 + i + j
	// case 'x':
	// 	s1, i := ProgramToString(b[1:])
	// 	s2, j := ProgramToString(b[1+i:])
	// 	return fmt.Sprintf("(xor %s %s)", s1, s2), 1 + i + j
	// case 'p':
	return nil
}

func (c *ProblemContext) Less(i, j int) bool {
	return c.arguments[i] < c.arguments[j]
}

func (c *ProblemContext) Len() int {
	return len(c.arguments)
}

func (c *ProblemContext) Swap(i, j int) {
	c.arguments[i], c.arguments[j] = c.arguments[j], c.arguments[i]
	c.outputs[i], c.outputs[j] = c.outputs[j], c.outputs[i]
}

func RandomArgs(size int) []uint64 {

	args := make([]uint64, size)

	for i, _ := range args {
		// next:
		a := uint64(rand.Uint32()) | (uint64(rand.Uint32()) << 32)
		// j := sort.Search(c.Len(), func(i int) bool { return c.arguments[i] <= a })
		// if j < c.Len() && c.arguments[j] == a {
		// 	goto next
		// }
		args[i] = a
	}

	return args
}

func CreateStruct(size int) []byte {
	s := make([]byte, size)
	for i := 0; i < size-1; i++ {
		s[i] = 1
	}
	return s
}

func PermutateStruct(s []byte) {
	l := len(s) - 2
	for i := l; i >= 0; i-- {
		a := int(s[i])
		if a > 0 {
			a--
			s[i] = 0
			s[i-1]++
			for j := i - 1; s[j] > 3; j-- {
				a += int(s[j] - 1)
				s[j] = 0
				s[j-1]++
			}

			for j := l - a + 1; j <= l; j++ {
				s[j] = 1
			}
			break
		}
	}
}

func CreateStructRev(size int) []byte {
	s := make([]byte, size)

	if size > 127 {
		panic(fmt.Errorf("size %d too large", size))
	}

	s[0] = byte(size - 1)
	for i := 0; s[i] > 3; i++ {
		s[i+1] = s[i] - 3
		s[i] = 3
	}
	return s
}

func PermutateStructRev(s []byte) bool {
	var i int
	l := len(s)
	for i = l - 1; i > 0; i-- {
		if s[i-1] > 0 {
			s[i-1]--
			s[i]++
			break
		}
	}

	a := s[i]

	// fmt.Printf("a: %d, i: %d, l: %d\n", a, i, l)

	if a >= byte(l-i) {
		s[i] = 0
		for j := i; j > 0; j-- {
			if s[j-1] > 0 {
				a++
				s[j-1]--
				// fmt.Printf("a: %d, j: %d\n", a, j)
				if a < byte(l-j) {
					for a > 3 {
						s[j] = 3
						a -= 3
						j++
					}
					s[j] = a
					break
				}
			}
		}
	}
	return s[0] > 0
}

func getOps(c *ProblemContext, f []byte, arity byte) (ops []byte) {

	switch arity {
	case 0:
		ops = append(make([]byte, 0, 5), '0', '1', 'X')
		if InFold(f) {
			ops = append(ops, 'Y', 'Z')
		}
	case 1:
		ops = c.ops1
	case 2:
		ops = c.ops2
	case 3:
		ops = make([]byte, 0, 2)

		if HasIf0(c.ops3) {
			ops = append(ops, 'i')
		}

		if HasFold(c.ops3) && !HasFold(f) {
			ops = append(ops, 'f')
		}
	}

	return ops
}

func CreateFunc(c *ProblemContext, s []byte) []byte {
	f := make([]byte, len(s))

	for i, arity := range s {
		ops := getOps(c, f[:i], arity)
		if len(ops) == 0 {
			return nil
		}

		f[i] = ops[0]
	}

	return f
}

func PermutateFunc(c *ProblemContext, f, s []byte) bool {
	for i := len(f) - 1; i >= 0; i-- {
		op := f[i]
		arity := s[i]
		ops := getOps(c, f[:i], arity)

		if len(ops) == 0 {
			return false
		}

		for j, validOp := range ops {

			if op == validOp && j+1 < len(ops) {
				f[i] = ops[j+1]
				return true
			}
		}

		f[i] = ops[0]
	}
	return false
}

func CreateProblemContext(p Problem) *ProblemContext {
	var ctx ProblemContext

	for _, op := range p.Operators {
		opInfo := OpMap[op]
		switch opInfo.arity {
		case 1:
			ctx.ops1 = append(ctx.ops1, opInfo.sym)
		case 2:
			ctx.ops2 = append(ctx.ops2, opInfo.sym)
		case 3:
			ctx.ops3 = append(ctx.ops3, opInfo.sym)
		case 4: // tfold
			ctx.tfold = true
			ctx.ops3 = append(ctx.ops3, 'f')
		}
	}

	ctx.size = int(p.Size)
	// size-1 for lambda
	ctx.length = int(p.Size - 1)

	// adjust length by one for embedded lambda
	if HasFold(ctx.ops3) {
		ctx.length--
	}

	return &ctx
}

func (c *ProblemContext) CheckStruct(st []byte) bool {
	count := [3]int{len(c.ops1), len(c.ops2), len(c.ops3)}

	for _, arity := range st {
		if arity > 0 {
			count[arity-1]--
		}
	}

	for _, c := range count {
		if c > 0 {
			return false
		}
	}
	return true
}

func (c *ProblemContext) CheckFun(fn []byte) bool {
	for _, o := range c.ops1 {
		if bytes.IndexByte(fn, o) < 0 {
			// fmt.Println("skip fn: ", string(fn))
			return false
		}
	}
	for _, o := range c.ops1 {
		if bytes.IndexByte(fn, o) < 0 {
			// fmt.Println("skip fn: ", string(fn))
			return false
		}
	}
	for _, o := range c.ops3 {
		if bytes.IndexByte(fn, o) < 0 {
			// fmt.Println("skip fn: ", string(fn))
			return false
		}
	}
	return true
}

func (c *ProblemContext) AddResults(args, outputs []uint64) {
	if len(args) != len(outputs) {
		panic(fmt.Sprintf("%d != %d", len(args), len(outputs)))
	}
	c.arguments = append(c.arguments, args...)
	c.outputs = append(c.outputs, outputs...)
}

func (c *ProblemContext) AddResult(arg, output uint64) {
	c.arguments = append(c.arguments, arg)
	c.outputs = append(c.outputs, output)
}

func (c *ProblemContext) CheckFunction(fn EvalFunc) bool {
	ec := EvalContext{make([]uint64, 1)}
	for i, a := range c.arguments {
		ec.vars[0] = a
		res := fn(&ec)
		if res != c.outputs[i] {
			//fmt.Printf("%03d  f(%X) == %X != %X\n", i, a, res, c.outputs[i])
			return false
		}
	}
	return true
}

// func BruteForce(p Problem) (ProblemContext, NextSolution) {
// 	ctx := ProblemContext{
// 		p.Size,
// 		processOperators(p.Operators),
// 		make([]uint64, 0),
// 		make([]uint64, 0),
// 	}

// 	return ctx, func(s []byte) []byte {

// 		for i := len(s) - 1; i >= 0; i-- {
// 			b := s[i]
// 			for j, op := range ctx.operators {
// 				if b == op && j+1 < len(ctx.operators) {
// 					s[i] = ctx.operators[j+1]
// 					return s
// 				}
// 			}
// 		}
// 	}
// }
