package user_functions

import (
	"bytes"
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"io"
	"log"
	"net/http"
)

func BindingsHTTPOutput(ctx *ofctx.OpenFunctionContext, in interface{}) int {
	input := in.(*http.Request)
	content, err := io.ReadAll(input.Body)
	if err != nil {
		return 500
	}
	log.Printf("binding - Data:%s, Header:%v", string(content), input.Header)

	if bytes.Equal(content, nil) {
		content = []byte("hello world")
	}

	err = ctx.SendTo(content, "echo")
	log.Printf("Send %v to output_demo\n", string(content))
	if err != nil {
		log.Printf("Error: %v\n", err)
		return 500
	}
	return 200
}
