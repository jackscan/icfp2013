package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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

	fmt.Println(reqStr)
	fmt.Println(payload)

	resp, err := http.Post(reqStr, "application/json", strings.NewReader(payload))
	defer resp.Body.Close()

	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))

	return body
}

type Problem struct {
	Id        string   `json:"id"`
	Size      int32    `json:"size"`
	Operators []string `json:"operators"`
	Solved    *bool    `json:"solved",omitempty`
	TimeLeft  *float32 `json:"timeLeft",omitempty`
	Challenge *string  `json:"challenge",omitempty`
}

func (s *Server) Train(size int) Problem {
	var problem Problem
	data := s.Post("train", fmt.Sprintf(
		`{ "size": %d }`, size))

	err := json.Unmarshal(data, &problem)
	if err != nil {
		panic(err)
	}

	return problem
}

type EvalResponse struct {
	Status  string    `json:"status"`
	Outputs *[]string `json:"outputs"`
	Message *string   `json:"message"`
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

	fmt.Println(buffer.String())

	var resp EvalResponse

	respData := s.Post("eval", buffer.String())

	err := json.Unmarshal(respData, &resp)
	if err != nil {
		panic(err)
	}

	if resp.Outputs == nil {
		panic(resp.Message)
	}

	outputs := make([]uint64, len(*resp.Outputs))

	for i, s := range *resp.Outputs {
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

		outputs[i] = 0

		for j, d := range b {
			outputs[i] |= uint64(d) << uint((len(b)-j-1)*8)
		}
	}

	return outputs
}
