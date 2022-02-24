package main

import (
	"context"
	"net/http"
	"sync"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
	dapr "github.com/dapr/go-sdk/client"
	"k8s.io/klog/v2"
)

var (
	client             dapr.Client
	initDaprClientOnce sync.Once
)

func initDaprClient() {
	initDaprClientOnce.Do(func() {
		var err error
		client, err = dapr.NewClient()
		if err != nil {
			panic(err)
		}
	})
}

func Handler(ofCtx ofctx.Context, in []byte) (out ofctx.Out, err error) {

	ofCtx.Send("sample-topic", []byte("hello"))

	return nil, err
}

func HelloWorldWithHttp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	klog.Infof("received :%s", r.URL.Path)
	initDaprClient()

	in := &dapr.InvokeBindingRequest{
		Name:      "sample-topic",
		Operation: "create",
		Data:      []byte(r.URL.Path),
	}
	_, err := client.InvokeBinding(r.Context(), in)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Error(err)
		return
	}
	w.Write([]byte("successful"))
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

	err = fwk.Register(ctx, HelloWorldWithHttp)
	if err != nil {
		klog.Fatal(err)
	}

	err = fwk.Start(ctx)
	if err != nil {
		klog.Fatal(err)
	}
}
