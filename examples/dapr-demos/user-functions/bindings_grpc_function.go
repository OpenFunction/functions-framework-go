package user_functions

import (
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"github.com/dapr/go-sdk/service/common"
	"log"
)

func BindingsGRPCFunction(ctx *ofctx.OpenFunctionContext, in interface{}) int {
	input := in.(*common.BindingEvent)
	log.Printf("binding - Data:%s, Meta:%v", input.Data, input.Metadata)

	if *ctx.Outputs.Enabled {
		err := ctx.SendTo(in, "output_demo")
		if err != nil {
			return 500
		}
	}

	return 200
}
