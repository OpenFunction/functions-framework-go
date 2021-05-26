package demo

import (
	"fmt"
	"github.com/OpenFunction/functions-framework-go/functionframeworks"
	"log"
	"net/http"
)

func BindingsFunction(ctx *functionframeworks.OpenFunctionContext, r *http.Request) int {
	// ctx.GetInput is equal to "content, err := io.ReadAll(r.Body)"
	content, err := ctx.GetInput(r)

	type Data struct {
		Type string `json:"type"`
		Content interface{} `json:"content"`
	}
	type Payload struct {
		Data *Data `json:"data"`
		Operation string `json:"operation"`
	}

	p := &Payload{}
	p.Operation = "create"

	switch v := content.(type) {
	case string:
		p.Data = &Data{
			Type: fmt.Sprintln(v),
			Content: content.(string),
		}
		err = ctx.SendTo(p, "op1")
		log.Printf("Send %v to op1\n", content)
		if err != nil {
			log.Printf("Error: %v\n", err)
		}
	case int:
		p.Data = &Data{
			Type: fmt.Sprintln(v),
			Content: content.(int),
		}
		err = ctx.SendTo(p, "op2")
		log.Printf("Send %v to op2\n", content)
		if err != nil {
			log.Printf("Error: %v\n", err)
		}
	default:
		p.Data = &Data{
			Type: fmt.Sprintln(v),
			Content: content,
		}
		err = ctx.SendTo(p, "op2")
		log.Printf("Send %v to op2\n", content)
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
