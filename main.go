package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"runtime"
	"time"
)

func calcComplexity(p Problem) int {
	c := 3
	maxa := 1
	l := len(p.Operators)
	for _, op := range p.Operators {

		a := int(OpMap[op].arity) - 1

		if op == "fold" {
			c = 5
		} else if op == "tfold" {
			a--
			c = 4
		} else if op == "bonus" {
			l--
		}

		if maxa < a {
			maxa = a
		}
	}
	return (int(p.Size) * l) * maxa * c
}

func chooseMinProblem(ps []Problem, maxsize int, maxop int, fold bool, tfold bool, bonus bool) *Problem {
	var ret *Problem
	size := math.MaxInt32
ploop:
	for i, p := range ps {
		numOps := len(p.Operators)
		if !p.Solved && (p.TimeLeft == nil || *p.TimeLeft > 0) && p.Size <= int32(maxsize) && numOps <= maxop {

			if !bonus {
				for _, op := range p.Operators {
					if op == "bonus" {
						continue ploop
					}
				}
			}

			if !fold {
				for _, op := range p.Operators {
					if op == "fold" || op == "tfold" {
						continue ploop
					}
				}
			} else if tfold {
				for _, op := range p.Operators {
					if op == "tfold" {
						goto comp
					}
				}
				continue ploop
			}
		comp:
			comp := calcComplexity(p)

			fmt.Printf("%d, %v\n", comp, p)

			if size > comp {
				ret = &ps[i]
				size = comp
			}
			//return &ps[i]
		}
	}
	return ret
}

func chooseMaxProblem(ps []Problem, maxsize int, maxop int, fold bool) *Problem {
	var ret *Problem
	size := 0

	fmt.Printf("max\n")
ploop:
	for i, p := range ps {
		numOps := len(p.Operators)
		if !p.Solved && p.Size <= int32(maxsize) && numOps <= maxop {
			if !fold {
				for _, op := range p.Operators {
					if op == "fold" || op == "tfold" {
						continue ploop
					}
				}
			}

			comp := calcComplexity(p)

			fmt.Printf("%d, %v\n", comp, p)

			if size < comp {
				ret = &ps[i]
				size = comp
			}
		}
	}
	return ret
}

func getProblem(ps []Problem, id string) *Problem {
	for i, p := range ps {
		if p.Id == id {
			return &ps[i]
		}
	}
	return nil
}

func nextFunction(ctx *ProblemContext, st, fn *[]byte) bool {

	scount := 0

	// defer func(start time.Time) {
	// 	end := time.Now()
	// 	fmt.Println("n: ", end.Sub(start))

	// }(time.Now())

	if *st == nil {
		*st = CreateStructRev(ctx.length, ctx.tfold)
		if !ctx.CheckStruct(*st) {
			goto nextst
		}
	}

nextfn:
	if *fn == nil {
		*fn = CreateFunc(ctx, *st)
		if *fn == nil {
			goto nextst
		}
	} else {
		if !PermutateFunc(ctx, *fn, *st) {
			goto nextst
		}
	}

	for !ctx.CheckFun(*fn) {
		scount++
		// fmt.Printf("skipping fn %s\n", string(*fn))
		if !PermutateFunc(ctx, *fn, *st) {
			goto nextst
		}
	}

	// if scount > 0 {
	// 	fmt.Printf("skipped: %d, %s\n", scount, string(*fn))
	// }

	return true

nextst:
	if !PermutateStructRev(*st) {
		return false
	}

	skipped := false

	for !ctx.CheckStruct(*st) || !CheckOp2StructRev(*st) {
		skipped = true
		fmt.Print(".")
		//fmt.Println(*st)
		if !PermutateStructRev(*st) {
			return false
		}
	}
	if skipped {
		fmt.Print("\n")
	}
	*fn = nil
	fmt.Println(*st)

	goto nextfn
}

func structGenerator(ctx *ProblemContext, sres chan *[]byte, done chan int, quick chan int) {
	go func() {

		defer func() {
			err := recover()
			if err != nil {
				fmt.Println("error in struct generator: ", err)
			}
		}()

		fmt.Println("struct generator started")

		st := CreateStructRev(ctx.length, ctx.tfold)

		for {

			if ctx.CheckStruct(st) {
				clone := make([]byte, len(st))
				copy(clone, st)
				sres <- &clone
			}

			select {
			case <-quick:
				PermutateStructFront(st)
				fmt.Println("quick ", st)
			default:
				if !PermutateStructRev(st) {
					break
				}
			}
		}
		fmt.Println("struct generator finished")
		done <- 1
	}()
}

func funGenerator(ctx *ProblemContext, sres, fres chan *[]byte, done chan int) {
	go func() {
		fmt.Println("fun generator started")

		defer func() {
			err := recover()
			if err != nil {
				fmt.Println("error in fun generator: ", err)
			}
		}()

		for {
			st, ok := <-sres

			if st == nil || !ok {
				fmt.Println("fun generator finished")
				done <- 1
				return
			}

			fn := CreateFunc(ctx, *st)

			if fn == nil {
				fmt.Println("failed to create function for struct ", *st)
				continue
			}

			fmt.Printf("%v, %s\n", *st, string(fn))

			for {
				// if ctx.CheckFun(fn) {
				//fmt.Printf("%v, %s\n", st, string(fn))
				efun := Compile(fn)
				if ctx.CheckFunction(efun) {
					clone := make([]byte, len(fn))
					copy(clone, fn)
					fres <- &clone
				}
				// }
				if !PermutateFunc(ctx, fn, *st) {
					break
				}
			}
		}
	}()
}

func solveParallel(problem *Problem, server *Server, argCount int, routineCount int, maxTime int, quickTime int) bool {

	fmt.Println("Parallel")

	fmt.Println(problem)

	ctx := CreateProblemContext(*problem)

	args := RandomArgs(argCount)

	outputs, solved := server.Eval(problem.Id, args)

	if outputs == nil {
		if solved {
			problem.Solved = true
		} else {
			problem.TimeLeft = new(float32)
		}
		return true
	}

	ctx.AddResults(args, outputs)

	stchan := make(chan *[]byte, routineCount)
	fnchan := make(chan *[]byte)
	donech := make(chan int)
	quickch := make(chan int)

	defer close(stchan)
	defer close(fnchan)
	defer close(donech)
	defer close(quickch)

	structGenerator(ctx, stchan, donech, quickch)
	for i := 0; i < routineCount; i++ {
		funGenerator(ctx, stchan, fnchan, donech)
	}

loop:
	for done := 0; done < routineCount+1; {
		select {
		case <-time.After(time.Duration(maxTime) * time.Second):
			fmt.Println("timed out")
			break loop
		case <-time.Tick(time.Duration(quickTime) * time.Second):

			fmt.Println("skip structs")
			quickch <- 1

		case <-donech:

			if done == 0 {
				for i := 0; i < routineCount; i++ {
					stchan <- nil
				}
			}
			done++
		case fn := <-fnchan:
			win, values := server.Guess(problem.Id, *fn)

			if win {
				fmt.Printf("WIN %s: %s\n", problem.Id, ProgramToString(*fn))
				problem.Solved = true
				break loop
			}

			if len(values) > 0 {
				if values[1] != values[2] {
					fmt.Printf("value mismatch: f(%x) == %x != %x, %s, %s\n", values[0], values[1], values[2], string(*fn), ProgramToString(*fn))
				}

				ctx.AddResult(values[0], values[1])
			}
		}
	}

	return problem.Solved
}

func solve(problem *Problem, server *Server, argCount int) (result bool) {

	result = false

	fmt.Println(problem)

	ctx := CreateProblemContext(*problem)

	args := RandomArgs(argCount)
	//fmt.Println(args)

	outputs, solved := server.Eval(problem.Id, args)

	if outputs == nil {
		if solved {
			problem.Solved = true
		} else {
			problem.TimeLeft = new(float32)
		}
		return true
	}

	ctx.AddResults(args, outputs)

	// for i, a := range args {
	// 	fmt.Println(a, "->", outputs[i])
	// }

	var st, fn []byte

	for nextFunction(ctx, &st, &fn) {

		// fmt.Println(string(fn), st)

		// start := time.Now()
		efun := Compile(fn)
		if ctx.CheckFunction(efun) {
			win, values := server.Guess(problem.Id, fn)

			// fmt.Printf("check %s\n", string(fn))

			if win {
				fmt.Printf("WIN %s: %s\n", problem.Id, ProgramToString(fn))
				problem.Solved = true
				return true
			}

			if len(values) > 0 {
				if values[1] != values[2] {
					fmt.Printf("value mismatch: f(%x) == %x != %x, %s, %s\n", values[0], values[1], values[2], string(fn), ProgramToString(fn))
				}

				ctx.AddResult(values[0], values[1])
			}
		}
		// end := time.Now()
		// fmt.Println(end.Sub(start))
	}

	return false
}

func loadProblems(problemsFile string) []Problem {
	problems := make([]Problem, 0)
	data, err := ioutil.ReadFile(problemsFile)

	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &problems)
	if err != nil {
		panic(err)
	}
	return problems
}

func writeProblems(file string, problems []Problem) {
	b, err := json.Marshal(problems)

	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	err = json.Indent(&buf, b, "", "   ")

	if err != nil {
		fmt.Printf("error at indent: %v\n", err)
	}

	fmt.Printf("writing problems to %s\n", file)

	ioutil.WriteFile(file, buf.Bytes(), 0644)
}

func main() {
	var url, auth string
	var queryProblems bool
	var queryTrain bool
	var test bool
	var problemType string
	var problemsFile string
	var problemSize int
	var evalString string
	var argString string
	var problemId string
	var opCount int
	var argCount int
	var useFold bool
	var keepGoing bool
	var listProblems bool
	var useTFold bool
	var useBonus bool
	var analyze string
	var routineCount int
	var maxTime int
	var quickTime int

	flag.StringVar(&url, "url", "http://icfpc2013.cloudapp.net", "game server")
	flag.StringVar(&auth, "auth", "0000abcdefghijklmnopqrstuvwxyz0123456789vpsH1H", "authorization token")
	flag.BoolVar(&queryProblems, "getproblems", false, "query and print myproblems")
	flag.BoolVar(&queryTrain, "train", false, "query and add training problem to problems file")
	flag.StringVar(&problemType, "type", "", "problem type")
	flag.BoolVar(&test, "test", false, "test")
	flag.BoolVar(&useFold, "fold", false, "whether to choose problems containing fold")
	flag.BoolVar(&useTFold, "tfold", false, "whether to filter problems containing tfold")
	flag.BoolVar(&useBonus, "bonus", false, "whether to filter problems containing bonus")
	flag.BoolVar(&listProblems, "list", false, "show overview of problems")
	flag.StringVar(&problemsFile, "problems", "train.txt", "problems file name")
	flag.StringVar(&evalString, "eval", "", "evaluate function")
	flag.StringVar(&argString, "arg", "", "evaluate argument")
	flag.IntVar(&problemSize, "size", 3, "maximum problem size")
	flag.IntVar(&opCount, "opcount", 4, "maximum operator count")
	flag.IntVar(&argCount, "argcount", 256, "number of arguments to evaluate")
	flag.IntVar(&routineCount, "rcount", 0, "number of parallel routines")
	flag.StringVar(&problemId, "solve", "", "problem id")
	flag.StringVar(&analyze, "analyze", "", "analyze function")
	flag.IntVar(&maxTime, "timelimit", 300, "limit in seconds to spend on problem")
	flag.IntVar(&quickTime, "quick", 60, "limit in seconds before skipping some structures")
	flag.BoolVar(&keepGoing, "k", false, "continue on failure to solve problem")
	flag.Parse()

	if useTFold {
		useFold = true
	}

	server := CreateServer(url, auth)

	if len(problemId) > 0 {

		problems := loadProblems(problemsFile)

		problem := getProblem(problems, problemId)

		if problem == nil {
			fmt.Printf("problem %s not found\n", problemId)
			return
		}

		if solve(problem, server, argCount) {
			writeProblems(problemsFile, problems)
		} else {
			fmt.Println("failed to solve problem")
			return
		}
	} else if len(evalString) > 0 {
		fn := []byte(evalString)
		fmt.Println(ProgramToString(fn))
		efun := Compile(fn)
		a := decodeHex(argString)
		ec := EvalContext{[]uint64{a}}

		res := efun(&ec)
		fmt.Printf("f(%X) = %X\n", a, res)
	} else if len(analyze) > 0 {
		fn := []byte(analyze)
		mask := AnalyzeFunc(fn)

		fmt.Printf("%s, %s\n%s\n", string(fn), string(mask), ProgramToString(fn))

	} else if test {

	} else if queryProblems {

		problems := make([]Problem, 0)
		data := server.Post("myproblems", "")

		err := json.Unmarshal(data, &problems)
		if err != nil {
			panic(err)
		}

		writeProblems(problemsFile, problems)
	} else if queryTrain {

		problems := loadProblems(problemsFile)
		problem := server.Train(problemSize, problemType)
		problems = append(problems, problem)
		writeProblems(problemsFile, problems)
	} else if listProblems {
		problems := loadProblems(problemsFile)
		p := chooseMinProblem(problems, problemSize, opCount, useFold, useTFold, useBonus)
		fmt.Println("next: ", p)
	} else {

		problems := loadProblems(problemsFile)

		if routineCount > 0 {
			runtime.GOMAXPROCS(4)
		}

		for {
			var problem *Problem

			problem = chooseMinProblem(problems, problemSize, opCount, useFold, useTFold, useBonus)

			if problem == nil {
				fmt.Println("no problem found")
				return
			}

			fmt.Printf("problem: %v\n", *problem)
			fmt.Printf("complexity: %d\n", calcComplexity(*problem))

			var success bool

			if routineCount > 0 {
				success = solveParallel(problem, server, argCount, routineCount, maxTime, quickTime)
			} else {
				success = solve(problem, server, argCount)
			}

			if success {
				writeProblems(problemsFile, problems)
			} else {
				fmt.Println("failed to solve problem")
				if !keepGoing {
					return
				}
			}
		}
	}
}
