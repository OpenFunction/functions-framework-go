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
	ofCtx.GetSyncRequest().Request = request.WithContext(nCtx)

	span.Tag(go2sky.TagHTTPMethod, request.Method)
	span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", request.Host, request.URL.Path))
	setPublicAttrs(nCtx, ofCtx, span)
	return nil
}

func postSyncRequestLogic(ctx ofctx.RuntimeContext) error {
	request := ctx.GetSyncRequest().Request
	span := go2sky.ActiveSpan(request.Context())
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
