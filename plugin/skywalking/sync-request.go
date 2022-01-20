package skywalking

import (
	"fmt"
	"time"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/SkyAPM/go2sky"
	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

func preSyncRequestLogic(ctx ofctx.Context, tracer *go2sky.Tracer) error {
	request := ctx.SyncRequestMeta.Request

	span, nCtx, err := tracer.CreateEntrySpan(ctx.SyncRequestMeta.Request.Context(), ctx.Name, func(key string) (string, error) {
		return request.Header.Get(key), nil
	})
	if err != nil {
		return err
	}
	ctx.SyncRequestMeta.Request = request.WithContext(nCtx)

	span.SetComponent(5004)
	span.Tag(go2sky.TagHTTPMethod, request.Method)
	span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", request.Host, request.URL.Path))
	span.SetSpanLayer(agentv3.SpanLayer_FAAS)
	return nil
}

func postSyncRequestLogic(ctx ofctx.Context) error {
	request := ctx.SyncRequestMeta.Request
	span := go2sky.ActiveSpan(request.Context())
	if span == nil {
		return nil
	}
	if ofctx.InternalError == ctx.Out.Code {
		span.Error(time.Now(), "Error on handling request")
	}

	if ctx.Error != nil {
		span.Error(time.Now(), ctx.Error.Error())
	}

	span.End()
	return nil
}
