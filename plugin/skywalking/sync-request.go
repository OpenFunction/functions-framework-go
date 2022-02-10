package skywalking

import (
	"fmt"
	"time"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/SkyAPM/go2sky"
)

func preSyncRequestLogic(ofCtx *ofctx.Context, tracer *go2sky.Tracer) error {
	request := ofCtx.SyncRequestMeta.Request

	span, nCtx, err := tracer.CreateEntrySpan(ofCtx.SyncRequestMeta.Request.Context(), ofCtx.Name, func(key string) (string, error) {
		return request.Header.Get(key), nil
	})
	if err != nil {
		return err
	}
	ofCtx.SyncRequestMeta.Request = request.WithContext(nCtx)

	span.Tag(go2sky.TagHTTPMethod, request.Method)
	span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", request.Host, request.URL.Path))
	setPublicAttrs(nCtx, ofCtx, span)
	return nil
}

func postSyncRequestLogic(ctx *ofctx.Context) error {
	request := ctx.SyncRequestMeta.Request
	span := go2sky.ActiveSpan(request.Context())
	if span == nil {
		return nil
	}
	defer span.End()

	if ofctx.InternalError == ctx.Out.Code {
		span.Error(time.Now(), "Error on handling request")
	}

	if ctx.Error != nil {
		span.Error(time.Now(), ctx.Error.Error())
	}
	return nil
}
