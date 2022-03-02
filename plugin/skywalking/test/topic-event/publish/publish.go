package main

import (
	"context"
	"time"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
	"github.com/SkyAPM/go2sky"
	"k8s.io/klog/v2"
	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

func pubsubFunction(ofCtx ofctx.Context, in []byte) (ofctx.Out, error) {
	tracer := go2sky.GetGlobalTracer()
	if tracer == nil {
		klog.Warningf("go2sky is not enabled")
		return ofCtx.ReturnOnInternalError().WithData([]byte("go2sky is not enabled")), nil
	}

	span, err := tracer.CreateExitSpan(ofCtx.GetNativeContext(), "publish-topic", "publish-topic", func(headerKey, headerValue string) error {
		ofCtx.GetInnerEvent().SetMetadata(headerKey, headerValue)
		return nil
	})
	if err != nil {
		klog.Error(err)
		return ofCtx.ReturnOnInternalError().WithData([]byte(err.Error())), err
	}
	defer span.End()

	span.SetSpanLayer(agentv3.SpanLayer_FAAS)
	span.SetComponent(5013)

	// topic
	_, err = ofCtx.Send("publish-topic", []byte(time.Now().String()))

	if err != nil {
		klog.Error(err)
		return ofCtx.ReturnOnInternalError().WithData([]byte(err.Error())), err
	}
	return ofCtx.ReturnOnSuccess().WithData([]byte("hello there")), nil
}

func main() {
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		klog.Fatal(err)
	}
	fwk.RegisterPlugins(map[string]plugin.Plugin{
		"skywalking": &skywalking.PluginSkywalking{},
	})

	err = fwk.Register(ctx, pubsubFunction)
	if err != nil {
		klog.Fatal(err)
	}

	err = fwk.Start(ctx)
	if err != nil {
		klog.Fatal(err)
	}
}
