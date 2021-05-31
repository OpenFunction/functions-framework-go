package user_functions

import (
	"bytes"
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"io"
	"log"
	"net/http"
)

func BindingsHTTPFunction(ctx *ofctx.OpenFunctionContext, in interface{}) int {
	input := in.(*http.Request)
	content, err := io.ReadAll(input.Body)
	if err != nil {
		return 500
	}
	log.Printf("binding - Data:%s, Header:%v", string(content), input.Header)

	if *ctx.Outputs.Enabled {

		if bytes.Equal(content, nil) {
			content = []byte("hello world")
		}

		type Payload struct {
			Data      string `json:"data"`
			Operation string `json:"operation"`
		}

		p := &Payload{}
		p.Operation = "create"
		p.Data = string(content)

		err = ctx.SendTo(p, "output_demo")
		log.Printf("Send %v to output_demo\n", string(content))
		if err != nil {
			log.Printf("Error: %v\n", err)

		}
	}
	if err != nil {
		return 500
	} else {
		return 200
	}
}
