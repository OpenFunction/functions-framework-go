package skywalking

import (
	"time"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/SkyAPM/go2sky"
)

func preAsyncRequestCommonLogic(ofCtx ofctx.RuntimeContext, tracer *go2sky.Tracer) (go2sky.Span, error) {
	event := ofCtx.GetInnerEvent()

	span, nCtx, err := tracer.CreateEntrySpan(ofCtx.GetNativeContext(), ofCtx.GetName(), func(headerKey string) (string, error) {
		value, _ := event.GetMetadata()[headerKey]
		return value, nil
	})
	if err != nil {
		return nil, err
	}
	ofCtx.SetNativeContext(nCtx)
	span.Tag(tagRuntime, string(ofctx.Async))
	setPublicAttrs(nCtx, ofCtx, span)

	return span, err
}

func preTopicEventLogic(ofCtx ofctx.RuntimeContext, tracer *go2sky.Tracer) error {
	span, err := preAsyncRequestCommonLogic(ofCtx, tracer)
	if err != nil {
		return err
	}
	span.Tag(tagComponentType, string(ofctx.OpenFuncTopic))
	return nil
}

func preBindingEventLogic(ofCtx ofctx.RuntimeContext, tracer *go2sky.Tracer) error {
	span, err := preAsyncRequestCommonLogic(ofCtx, tracer)
	if err != nil {
		return err
	}
	span.Tag(tagComponentType, string(ofctx.OpenFuncBinding))
	return nil
}

func postAsyncRequestLogic(ctx ofctx.RuntimeContext) error {
	span := go2sky.ActiveSpan(ctx.GetNativeContext())
	if span == nil {
		return nil
	}
	defer span.End()

	if ofctx.InternalError == ctx.GetOut().GetCode() {
		span.Error(time.Now(), "Error on binding event")
	}

	if ctx.GetError() != nil {
		span.Error(time.Now(), ctx.GetError().Error())
	}
	return nil
}
