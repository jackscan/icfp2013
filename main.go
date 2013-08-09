package main

import (
	"encoding/json"
	// "bytes"
	"flag"
	"fmt"
	"io/ioutil"
)

func main() {
	var url, auth string
	var queryProblems bool
	var queryTrain bool
	var test bool
	var problemsFile string

	flag.StringVar(&url, "url", "http://icfpc2013.cloudapp.net", "game server")
	flag.StringVar(&auth, "auth", "0000abcdefghijklmnopqrstuvwxyz0123456789vpsH1H", "authorization token")
	flag.BoolVar(&queryProblems, "getproblems", false, "query and print myproblems")
	flag.BoolVar(&queryTrain, "addtrain", false, "query and add training problem to problems file")
	flag.BoolVar(&test, "test", false, "test")
	flag.StringVar(&problemsFile, "problems", "train.txt", "problems file name")
	flag.Parse()

	server := CreateServer(url, auth)

	if test {
		problem := server.Train(3)
		b, err := json.Marshal(problem)

		if err != nil {
			panic(err)
		}

		fmt.Println(string(b))

		result := server.Eval(problem.Id, []uint64{0xA5A5A5A5A5A5A5A5, 0x1, 0xFFFFFFFFFFFFFFFF})

		fmt.Println("result:", result)

	} else if queryProblems {

		data := server.Post("myproblems", "")

		// err := json.Unmarshal(data, &problems)
		// if err != nil {
		// 	panic(err)
		// }

		// b, err := json.Marshal(problems)

		// if err != nil {
		// 	panic(err)
		// }

		// fmt.Println(string(b))

		fmt.Println(string(data))
	} else {
		problems := make([]Problem, 0)

		data, err := ioutil.ReadFile(problemsFile)

		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(data, &problems)
		if err != nil {
			panic(err)
		}

		if queryTrain {

			var problem Problem
			data := server.Post("train", "")

			err = json.Unmarshal(data, &problem)
			if err != nil {
				panic(err)
			}

			problems = append(problems, problem)

			b, err := json.Marshal(problems)

			if err != nil {
				panic(err)
			}

			ioutil.WriteFile(problemsFile, b, 0644)

		} else {
			b, err := json.Marshal(problems)

			if err != nil {
				panic(err)
			}

			fmt.Println(string(b))
		}
	}
	// else {
	// 	fmt.Println(string(server.Post("train", nil)))
	// }

}
