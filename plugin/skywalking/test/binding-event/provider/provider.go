package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/SkyAPM/go2sky"
	dapr "github.com/dapr/go-sdk/client"
	"k8s.io/klog/v2"

	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
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

func HelloWorldWithHttp(w http.ResponseWriter, r *http.Request) {
	initDaprClient()

	tracer := go2sky.GetGlobalTracer()

	metadata := make(map[string]string)

	span, err := tracer.CreateExitSpan(r.Context(), "", "", func(headerKey, headerValue string) error {
		metadata[headerKey] = headerValue
		return nil
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Error(err)
		return
	}
	defer span.End()

	in := &dapr.InvokeBindingRequest{
		Name:     "",
		Metadata: metadata,
	}
	out, err := client.InvokeBinding(r.Context(), in)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Error(err)
		return
	}
	w.Write(out.Data)
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
