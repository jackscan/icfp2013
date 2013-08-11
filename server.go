package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	url  string
	auth string
}

func CreateServer(url, auth string) (s *Server) {
	s = new(Server)
	s.url = url
	s.auth = auth

	return s
}

func (s *Server) Post(req string, payload string) []byte {

	reqStr := fmt.Sprintf("%s/%s?auth=%s", s.url, req, s.auth)

	// fmt.Println(reqStr)
	// fmt.Println(payload)
retry:
	resp, err := http.Post(reqStr, "application/json", strings.NewReader(payload))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("got status %d: %s\n", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode == 429 {
		fmt.Println("retrying in 4 seconds")
		time.Sleep(time.Second * 4)
		goto retry
	}

	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	// fmt.Println(string(body))

	return body
}

type Problem struct {
	Id        string   `json:"id"`
	Size      int32    `json:"size"`
	Operators []string `json:"operators"`
	Solved    bool     `json:"solved"`
	TimeLeft  *float32 `json:"timeLeft",omitempty`
	Challenge *string  `json:"challenge",omitempty`
}

// func (s *Server) Train(size int) Problem {
// 	var problem Problem
// 	data := s.Post("train", fmt.Sprintf(
// 		`{ "size": %d }`, size))

// 	err := json.Unmarshal(data, &problem)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return problem
// }

type EvalResponse struct {
	Status  string    `json:"status"`
	Outputs *[]string `json:"outputs"`
	Message *string   `json:"message"`
}

func decodeHex(s string) uint64 {
	var b []byte
	var err error

	if strings.HasPrefix(s, "0x") {
		b, err = hex.DecodeString(s[2:])
	} else {
		b, err = hex.DecodeString(s)
	}

	if err != nil {
		panic(err)
	}

	result := uint64(0)

	for j, d := range b {
		result |= uint64(d) << uint((len(b)-j-1)*8)
	}

	return result
}

func (s *Server) Eval(id string, args []uint64) []uint64 {
	buffer := bytes.NewBufferString(
		fmt.Sprintf(`{
			"id": "%s",
			"arguments": ["0x%X"`, id, args[0]))

	for _, a := range args[1:] {
		buffer.WriteString(fmt.Sprintf(`, "0x%X"`, a))
	}

	buffer.WriteString("]}")

	// fmt.Println(buffer.String())

	var resp EvalResponse

	respData := s.Post("eval", buffer.String())

	if string(respData) == "already solved" {
		return nil
	}

	err := json.Unmarshal(respData, &resp)
	if err != nil {
		panic(string(respData))
	}

	if resp.Outputs == nil {
		panic(resp.Message)
	}

	outputs := make([]uint64, len(*resp.Outputs))

	for i, s := range *resp.Outputs {
		outputs[i] = decodeHex(s)
	}

	return outputs
}

type GuessResponse struct {
	Status    string    `json:"status"`
	Values    *[]string `json:"values"`
	Message   *string   `json:"message"`
	Lightning *bool     `json:"lightning"`
}

func (s *Server) Guess(id string, b []byte) (bool, []uint64) {

	text := ProgramToString(b)

	fmt.Println("guessing: ", text)

	data := s.Post("guess", fmt.Sprintf(
		`{ "id": "%s", "program": "%s" }`, id, text))

	var resp GuessResponse

	err := json.Unmarshal(data, &resp)
	if err != nil {
		panic(fmt.Errorf("error in response '%s': %v", string(data), err))
	}

	if resp.Status == "error" {
		fmt.Printf("error: %s", *resp.Message)
	}

	var values []uint64

	if resp.Values != nil {
		values = make([]uint64, len(*resp.Values))
		for i, s := range *resp.Values {
			values[i] = decodeHex(s)
		}
	}

	return resp.Status == "win", values
}

func (s *Server) Train(size int, typ string) Problem {

	var query string

	if len(typ) > 0 {
		query = fmt.Sprintf(`{ "size": "%d", "operators": ["%s"] }`, size, typ)
	} else {
		query = fmt.Sprintf(`{ "size": "%d" }`, size)
	}
	data := s.Post("train", query)

	var problem Problem

	fmt.Println(string(data))

	err := json.Unmarshal(data, &problem)
	if err != nil {
		panic(fmt.Errorf("error in '%s': %v", string(data), err))
	}

	return problem
}
