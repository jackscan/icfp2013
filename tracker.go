package main

import (
	"bytes"
	"fmt"
	"math/rand"
	// "sort"
	"sync"
)

type OpCount struct {
	max   int
	count int
}

type ProblemContext struct {
	size    int
	length  int
	ops     [][]byte
	foldOps [][]byte
	tfold   bool

	lock      sync.RWMutex
	arguments []uint64
	outputs   []uint64

	opcount map[byte]OpCount
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
		a := uint64(rand.Uint32()) | (uint64(rand.Uint32()) << 32)
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

func CreateStructRev(size int, tfold bool) []byte {
	s := make([]byte, size)

	if size > 127 {
		panic(fmt.Errorf("size %d too large", size))
	}

	if tfold {
		s[0] = 3
		s[1] = 0
		s[2] = 0

		s[3] = byte(size - 3 - 1)
		for i := 3; s[i] > 3; i++ {
			s[i+1] = s[i] - 3
			s[i] = 3
		}

	} else {
		s[0] = byte(size - 1)
		for i := 0; s[i] > 3; i++ {
			s[i+1] = s[i] - 3
			s[i] = 3
		}
	}
	return s
}

func PermutateStructFront(s []byte) {
	var j int
	for j = 0; s[j] <= 1 && j < len(s); j++ {
	}

	if j < len(s) {
		for i := j + 1; i < len(s); i++ {
			if s[i] < 3 {
				s[i]++
				break
			}
		}
		s[j]--
	}
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

func InFold(index int, b []byte, st []byte) (result bool) {
	result = false
	i := bytes.IndexByte(b, 'f')
	if i >= 0 {
		i1 := skipStruct(st, i+1, 2)
		i2 := skipStruct(st, i1, 1)

		// fmt.Printf("infold: %d, %d, %d, %d\n", i, i1, i2, index)
		return i1 <= index && index < i2
	}
	return false
}

func skipStruct(s []byte, index int, count int) int {
	a := count
	var i int

	// fmt.Printf("skip:\n%v\n", s[index:])

	for i = index; a > 0; i++ {
		a = a + int(s[i]) - 1
		// fmt.Print(a, ",")
	}
	// fmt.Println(i)
	return i
}

// Check if 2nd arg of all Op2 has smaller arity than 1st arg
func CheckOp2StructRev(s []byte) bool {
	for i, arity := range s {
		if arity == 2 {
			i1 := i + 1
			i2 := skipStruct(s, i1, 1)
			if s[i1] > s[i2] {
				//fmt.Printf("skipop2 %v, %v, %b\n", s, s[i1:i2], s[i2])
				return false
			}
		}
	}
	return true
}

// func updateOpUsage(c *ProblemContext, op byte, unused *[]byte, acount int) bool {

// }

func checkOp(o []byte, fn []byte, st []byte, index int) bool {

	if st[index] == 0 {
		return true
	}

	ops := make([]byte, len(o))
	copy(ops, o)

	for _, b := range fn[:index+1] {
		i := bytes.IndexByte(ops, b)
		if i >= 0 {
			n := len(ops) - 1
			ops[i] = ops[n]
			ops = ops[0:n]
		}
	}

	acount := 0

	for _, a := range st[index+1:] {
		if a == st[index] {
			acount++
		}
	}

	if acount < len(ops) {
		// fmt.Printf("checkOps: %s, %v, %d, %v\n", string(fn), st, index, string(ops))
		return false
	}
	return true
}

func initFunc(c *ProblemContext, s []byte, f []byte, start int) bool {

	for i := start; i < len(s); i++ {
		arity := s[i]

		ops := c.ops[arity]
		if (arity == 3 && HasFold(f)) || InFold(i, f, s) {
			ops = c.foldOps[arity]
		}

		if len(ops) == 0 {
			return false
		}

		for j := 0; j < len(ops); j++ {
			f[i] = ops[j]
			if checkOp(ops, f, s, i) {
				break
			}
		}
	}
	return true
}

func CreateFunc(c *ProblemContext, s []byte) []byte {
	f := make([]byte, len(s))

	if c.tfold {

		// fmt.Println("tfold")

		f[0] = 'f'
		f[1] = 'X'
		f[2] = '0'

		for i := 3; i < len(s); i++ {

			// fmt.Printf("%v, %s, %d, %v\n", s, string(f), i, *c)
			arity := s[i]
			ops := c.foldOps[arity]
			if len(ops) == 0 {
				// fmt.Println("exit", arity)
				return nil
			}

			for j := 0; j < len(ops); j++ {
				f[i] = ops[j]
				if checkOp(ops, f, s, i) {
					break
				}
			}
		}

	} else {

		if !initFunc(c, s, f, 0) {
			return nil
		}
	}

	return f
}

func PermutateFunc(c *ProblemContext, f, s []byte) bool {

	min := 0

	if c.tfold {
		min = 3
	}

	// fmt.Printf("%v, %s\n", s, string(f))

	mask := AnalyzeFunc(f)

	for i := len(f) - 1; i >= min; i-- {

		op := f[i]
		arity := s[i]

		var ops []byte

		if c.tfold || (arity == 3 && HasFold(f)) || InFold(i, f, s) {
			ops = c.foldOps[arity]
		} else {
			ops = c.ops[arity]
		}

		if len(ops) == 0 {
			return false
		}

		if mask[i] != 'i' {

		oploop:
			for j, validOp := range ops {

				if op == validOp {
					for k := j + 1; k < len(ops); k++ {
						f[i] = ops[k]

						if checkOp(ops, f, s, i) {
							if initFunc(c, s, f, i+1) {
								return true
							} else {
								break oploop
							}
						}
					}
				}
			}
		} else {
			// fmt.Printf("skipped: %d\n", i)
		}

		f[i] = ops[0]
	}

	return false
}

func CreateProblemContext(p Problem) *ProblemContext {
	var ctx ProblemContext

	ctx.ops = make([][]byte, 4)
	ctx.foldOps = make([][]byte, 4)

	// for i, _ := range ctx.ops {
	// 	ctx.ops[i] = make([]byte, 0)
	// }

	for _, op := range p.Operators {
		opInfo := OpMap[op]
		sym := opInfo.sym
		arity := opInfo.arity

		if sym == 't' {
			ctx.tfold = true
			sym = 'f'
		}

		ctx.ops[arity] = append(ctx.ops[arity], sym)
		if sym != 'f' {
			ctx.foldOps[arity] = append(ctx.foldOps[arity], sym)
		}
	}

	if ctx.tfold {
		ctx.ops[0] = []byte{'0', '1', 'Y', 'Z'}
		ctx.foldOps[0] = []byte{'0', '1', 'Y', 'Z'}
	} else {
		ctx.ops[0] = []byte{'0', '1', 'X'}
		ctx.foldOps[0] = []byte{'0', '1', 'X', 'Y', 'Z'}
	}

	ctx.size = int(p.Size)
	// size-1 for lambda
	ctx.length = int(p.Size - 1)

	// adjust length by one for embedded lambda
	if HasFold(ctx.ops[3]) {
		ctx.length--
	}

	return &ctx
}

func (c *ProblemContext) CheckStruct(st []byte) bool {
	count := [3]int{len(c.ops[1]), len(c.ops[2]), len(c.ops[3])}

	for _, arity := range st {
		if arity > 0 {
			count[arity-1]--
		}

		if len(c.ops[arity]) == 0 {
			fmt.Printf("no op for arity %d\n", arity)
			return false
		}
	}

	for _, c := range count {
		if c > 0 {
			// fmt.Printf("arity %d: %d, %v\n", i+1, c, st)
			return false
		}
	}
	return true
}

func (c *ProblemContext) CheckFun(fn []byte) bool {

	for _, ops := range c.ops[1:] {
		for _, o := range ops {
			if bytes.IndexByte(fn, o) < 0 {
				fmt.Printf("skip fn: %s %c\n", string(fn), o)
				return false
			}
		}
	}
	return true
}

func (c *ProblemContext) AddResults(args, outputs []uint64) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if len(args) != len(outputs) {
		panic(fmt.Sprintf("%d != %d", len(args), len(outputs)))
	}
	c.arguments = append(c.arguments, args...)
	c.outputs = append(c.outputs, outputs...)
}

func (c *ProblemContext) AddResult(arg, output uint64) {

	c.lock.Lock()
	defer c.lock.Unlock()

	c.arguments = append(c.arguments, arg)
	c.outputs = append(c.outputs, output)
}

func (c *ProblemContext) CheckFunction(fn EvalFunc) bool {

	c.lock.RLock()
	defer c.lock.RUnlock()

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
