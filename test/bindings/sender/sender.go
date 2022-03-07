package main

import (
	"context"
	"encoding/json"

	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

func main() {
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		klog.Exit(err)
	}
	fwk.RegisterPlugins(getLocalPlugins())
	if err := fwk.Register(ctx, Sender); err != nil {
		klog.Exit(err)
	}
	if err := fwk.Start(ctx); err != nil {
		klog.Exit(err)
	}
}

func getLocalPlugins() map[string]plugin.Plugin {
	localPlugins := map[string]plugin.Plugin{}

	if len(localPlugins) == 0 {
		return nil
	} else {
		return localPlugins
	}
}

func Sender(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	msg := map[string]string{
		"hello": "world",
	}

	msgBytes, _ := json.Marshal(msg)

	res, err := ctx.Send("target", msgBytes)
	if err != nil {
		klog.Error(err)
		return ctx.ReturnOnInternalError(), err
	}
	klog.Infof("send msg and receive result: %s", string(res))

	return ctx.ReturnOnSuccess(), nil
}
