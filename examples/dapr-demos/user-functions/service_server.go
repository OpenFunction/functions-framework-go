package user_functions

import (
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"github.com/dapr/go-sdk/service/common"
	"log"
)

func ServiceGRPCServer(ctx *ofctx.OpenFunctionContext, in interface{}) int {
	input := in.(*common.InvocationEvent)
	if input == nil {
		log.Printf("nil invocation parameter")
		return 500
	}
	log.Printf(
		"echo - ContentType:%s, Verb:%s, QueryString:%s, %s",
		input.ContentType, input.Verb, input.QueryString, input.Data,
	)
	return 200
}
