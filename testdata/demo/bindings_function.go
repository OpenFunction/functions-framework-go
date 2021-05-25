package demo

import (
	"fmt"
	"github.com/OpenFunction/functions-framework-go/functionframeworks"
	"io"
	"net/http"
	"time"
)

func InputOnlyFunction(ctx *functionframeworks.OpenFunctionContext, r *http.Request) int {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Errorf("Failed to get data: %v\n", err)
		return 500
	}

	if string(data) == "Hello" {
		fmt.Println("User Success!")
		return 200
	} else {
		fmt.Println("User Failed!")
		return 500
	}
}

func BindingsFunction(ctx *functionframeworks.OpenFunctionContext, r *http.Request) int {
	// ctx.GetInput is equal to "content, err := io.ReadAll(r.Body)"
	content, err := ctx.GetInput(r)

	type Data struct {
		OrderId int `json:"order_id"`
		Content string `json:"content"`
	}
	type Payload struct {
		Data *Data `json:"data"`
		Operation string `json:"operation"`
	}

	n := 0

	for {
		n++
		p := &Payload{}
		p.Data = &Data{
			OrderId: n,
			Content: content.(string),
		}
		p.Operation = "create"
		time.Sleep(1 * time.Second)

		err := ctx.SendTo(p, "OUTPUT1")
		if err != nil {
			fmt.Errorf("Error: %v\n", err)
		}
	}

	if err != nil {
		return 500
	} else {
		return 200
	}
}
