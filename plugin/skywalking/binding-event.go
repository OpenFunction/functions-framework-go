package skywalking

import (
	"fmt"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/SkyAPM/go2sky"
	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

func preBindingEventLogic(ctx ofctx.Context, tracer *go2sky.Tracer) error {
	event := ctx.EventMeta.BindingEvent

	span, nCtx, err := tracer.CreateEntrySpan(ctx.Ctx, fmt.Sprintf("%s/%s", ctx.Name, ctx.EventMeta.InputName), func(headerKey string) (string, error) {
		if value, ok := event.Metadata[headerKey]; ok {
			return value, nil
		}
		return "", nil
	})
	if err != nil {
		return err
	}
	ctx.Ctx = nCtx

	span.SetSpanLayer(agentv3.SpanLayer_FAAS)
	return nil
}

func postBindingEventLogic(ctx ofctx.Context, tracer *go2sky.Tracer) error {

	return nil
}
