package user_functions

import (
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"log"
)

func ServiceGRPCClient(ctx *ofctx.OpenFunctionContext, in interface{}) int {
	greeting := []byte("hello")
	err := ctx.SendTo(greeting, "server")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return 500
	}
	return 200
}
