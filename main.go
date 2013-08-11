package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
)

func calcComplexity(p Problem) int {
	fold := 1
	for _, op := range p.Operators {
		if op == "fold" || op == "tfold" {
			fold = 2
		}
	}
	return len(p.Operators) * int(p.Size) * fold
}

func chooseMinProblem(ps []Problem, maxsize int, maxop int, fold bool, tfold bool) *Problem {
	var ret *Problem
	size := maxop*maxsize*2 + 1

	fmt.Printf("min: %d\n", size)
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

	if *st == nil {
		*st = CreateStructRev(ctx.length)
		if !ctx.CheckStruct(*st) {
			goto nextst
		}
	}

nextfn:
	if *fn == nil {
		*fn = CreateFunc(ctx, *st)
	} else {
		if !PermutateFunc(ctx, *fn, *st) {
			goto nextst
		}
	}

	for !ctx.CheckFun(*fn) {
		if !PermutateFunc(ctx, *fn, *st) {
			goto nextst
		}
	}
	return true

nextst:
	if !PermutateStructRev(*st) {
		return false
	}

	skipped := false

	for !ctx.CheckStruct(*st) {
		skipped = true
		fmt.Print(".")
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

func solve(problem *Problem, server *Server, argCount int) (result bool) {

	result = false

	fmt.Println(problem)

	ctx := CreateProblemContext(*problem)

	args := RandomArgs(argCount)
	//fmt.Println(args)

	outputs := server.Eval(problem.Id, args)

	if outputs == nil {
		problem.Solved = true
		return true
	}

	ctx.AddResults(args, outputs)

	// for i, a := range args {
	// 	fmt.Println(a, "->", outputs[i])
	// }

	var st, fn []byte

	for nextFunction(ctx, &st, &fn) {

		//fmt.Println(string(fn), st)

		efun := Compile(fn)
		if ctx.CheckFunction(efun) {
			win, values := server.Guess(problem.Id, fn)

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
	var listProblems bool
	var useTFold bool

	flag.StringVar(&url, "url", "http://icfpc2013.cloudapp.net", "game server")
	flag.StringVar(&auth, "auth", "0000abcdefghijklmnopqrstuvwxyz0123456789vpsH1H", "authorization token")
	flag.BoolVar(&queryProblems, "getproblems", false, "query and print myproblems")
	flag.BoolVar(&queryTrain, "train", false, "query and add training problem to problems file")
	flag.StringVar(&problemType, "type", "", "problem type")
	flag.BoolVar(&test, "test", false, "test")
	flag.BoolVar(&useFold, "fold", false, "whether to choose problems containing fold")
	flag.BoolVar(&useTFold, "tfold", false, "whether to filter problems containing tfold")
	flag.BoolVar(&listProblems, "list", false, "show overview of problems")
	flag.StringVar(&problemsFile, "problems", "train.txt", "problems file name")
	flag.StringVar(&evalString, "eval", "", "evaluate function")
	flag.StringVar(&argString, "arg", "", "evaluate argument")
	flag.IntVar(&problemSize, "size", 3, "maximum problem size")
	flag.IntVar(&opCount, "opcount", 4, "maximum operator count")
	flag.IntVar(&argCount, "argcount", 256, "number of arguments to evaluate")
	flag.StringVar(&problemId, "solve", "", "problem id")
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

	} else if test {

		problems := loadProblems(problemsFile)
		problem := getProblem(problems, "9yEs85hEXD97rCdPjpv37BdS")

		ctx := CreateProblemContext(*problem)
		var st, fn []byte
		for nextFunction(ctx, &st, &fn) {
			fmt.Println(st, string(fn))
		}

		// problem := server.Train(6)
		// b, err := json.Marshal(problem)

		// if err != nil {
		// 	panic(err)
		// }

		/*problems := make([]Problem, 0)

		data, err := ioutil.ReadFile(problemsFile)

		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(data, &problems)
		if err != nil {
			panic(err)
		}

		// fmt.Println(string(data))

		for {

			problem := chooseProblem(problems, problemSize, false)

			if problem == nil {
				fmt.Println("no problem found")
				return
			}

			if solve(problem, server) {
				writeProblems(problemsFile, problems)
			} else {
				return
			}
		}*/

		//fmt.Println(string(fn), ":", ProgramToString(fn))

		// fmt.Println(st)
		// fmt.Println(fn)

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
		chooseMinProblem(problems, problemSize, opCount, useFold, useTFold)
	} else {

		// minMaxToggle := false
		problems := loadProblems(problemsFile)

		for {
			var problem *Problem

			// if minMaxToggle {
			// 	problem = chooseMaxProblem(problems, problemSize, opCount, useFold)
			// } else {
			// 	problem = chooseMinProblem(problems, problemSize, opCount, useFold)
			// }

			problem = chooseMinProblem(problems, problemSize, opCount, useFold, useTFold)

			// minMaxToggle = !minMaxToggle

			if problem == nil {
				fmt.Println("no problem found")
				return
			}

			fmt.Printf("problem: %v\n", *problem)
			fmt.Printf("complexity: %d\n", calcComplexity(*problem))

			if solve(problem, server, argCount) {
				writeProblems(problemsFile, problems)
			} else {
				fmt.Println("failed to solve problem")
				return
			}
		}
	}
	// else {
	// 	fmt.Println(string(server.Post("train", nil)))
	// }

}
