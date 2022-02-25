package main

import (
	"context"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
	"github.com/SkyAPM/go2sky"
	"k8s.io/klog/v2"
)

func Handler(ofCtx ofctx.Context, in []byte) (out ofctx.Out, err error) {
	tracer := go2sky.GetGlobalTracer()
	if tracer == nil {
		return out.WithData([]byte("go2sky is not enabled")), nil
	}

	span, err := tracer.CreateExitSpan(ofCtx.GetNativeContext(), "sample-topic", "sample-topic", func(headerKey, headerValue string) error {
		ofCtx.GetInnerEvent().SetMetadata(headerValue, headerValue)
		return nil
	})
	if err != nil {
		return out.WithData([]byte(err.Error())), nil
	}
	defer span.End()

	resp, err := ofCtx.Send("sample-topic", in)
	if err != nil {
		return out.WithData([]byte(err.Error())), nil
	}

	return out.WithData(resp), err
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

	err = fwk.Register(ctx, Handler)
	if err != nil {
		klog.Fatal(err)
	}

	err = fwk.Start(ctx)
	if err != nil {
		klog.Fatal(err)
	}
}
