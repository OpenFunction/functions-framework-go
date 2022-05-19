package main

import (
	"encoding/json"
	"net/http"

	"github.com/OpenFunction/functions-framework-go/functions"
)

func init() {
	functions.HTTP("HelloWorld", helloWorld, functions.WithFunctionPath("/helloworld"))
	functions.HTTP("Foo", foo, functions.WithFunctionPath("/foo"))
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"hello": "world",
	}
	responseBytes, _ := json.Marshal(response)
	w.Header().Set("Content-type", "application/json")
	w.Write(responseBytes)
}

func foo(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"hello": "foo",
	}
	responseBytes, _ := json.Marshal(response)
	w.Header().Set("Content-type", "application/json")
	w.Write(responseBytes)
}
