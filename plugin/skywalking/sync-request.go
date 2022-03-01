package skywalking

import (
	"fmt"
	"time"

	"github.com/SkyAPM/go2sky"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
)

func preSyncRequestLogic(ofCtx ofctx.RuntimeContext, tracer *go2sky.Tracer) error {
	request := ofCtx.GetSyncRequest().Request

	span, nCtx, err := tracer.CreateEntrySpan(ofCtx.GetSyncRequest().Request.Context(), ofCtx.GetName(), func(key string) (string, error) {
		return request.Header.Get(key), nil
	})
	if err != nil {
		return err
	}
	ofCtx.GetSyncRequest().Request = request.WithContext(nCtx)              // HTTPFunction
	ofCtx.SetNativeContext(go2sky.WithSpan(ofCtx.GetNativeContext(), span)) // OpenFunction

	span.Tag(go2sky.TagHTTPMethod, request.Method)
	span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", request.Host, request.URL.Path))
	span.Tag(tagRuntime, string(ofctx.Knative))
	setPublicAttrs(nCtx, ofCtx, span)
	return nil
}

func postSyncRequestLogic(ctx ofctx.RuntimeContext) error {
	span := go2sky.ActiveSpan(ctx.GetNativeContext())
	if span == nil {
		return nil
	}

	defer span.End()

	if ofctx.InternalError == ctx.GetOut().GetCode() {
		span.Error(time.Now(), "Error on handling request")
	}

	if ctx.GetError() != nil {
		span.Error(time.Now(), ctx.GetError().Error())
	}
	return nil
}
