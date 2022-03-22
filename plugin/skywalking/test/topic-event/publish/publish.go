package main

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
)

func pubsubFunction(ofCtx ofctx.Context, in []byte) (ofctx.Out, error) {

	// topic
	_, err := ofCtx.Send("publish-topic", []byte(time.Now().String()))

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
