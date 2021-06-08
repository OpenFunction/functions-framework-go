package user_functions

import (
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"github.com/dapr/go-sdk/service/common"
	"log"
)

func BindingsGRPCNoOutput(ctx *ofctx.OpenFunctionContext, in interface{}) int {
	input := in.(*common.BindingEvent)
	log.Printf("binding - Data:%s, Meta:%v", input.Data, input.Metadata)
	return 200
}
