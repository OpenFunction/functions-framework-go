package skywalking

import (
	"time"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/SkyAPM/go2sky"
	"k8s.io/klog/v2"
)

func preBindingEventLogic(ofCtx ofctx.RuntimeContext, tracer *go2sky.Tracer) error {
	event := ofCtx.GetInnerEvent()

	span, nCtx, err := tracer.CreateEntrySpan(ofCtx.GetNativeContext(), ofCtx.GetName(), func(headerKey string) (string, error) {
		value, _ := event.GetMetadata()[headerKey]
		return value, nil
	})
	if err != nil {
		return err
	}
	ofCtx.SetNativeContext(nCtx)
	setPublicAttrs(nCtx, ofCtx, span)
	return nil
}

func postBindingEventLogic(ctx ofctx.RuntimeContext) error {
	span := go2sky.ActiveSpan(ctx.GetNativeContext())
	if span != nil {
		klog.Warning("ActiveSpan is nil")
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
