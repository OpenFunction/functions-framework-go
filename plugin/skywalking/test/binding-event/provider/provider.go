package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/SkyAPM/go2sky"
	dapr "github.com/dapr/go-sdk/client"
	"k8s.io/klog/v2"
	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"

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
	if r.Method != http.MethodGet {
		return
	}

	initDaprClient()
	tracer := go2sky.GetGlobalTracer()
	metadata := make(map[string]string)
	span, err := tracer.CreateExitSpan(r.Context(), "sample-topic", "of:8081", func(headerKey, headerValue string) error {
		metadata[headerKey] = headerValue
		return nil
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Error(err)
		return
	}
	span.SetSpanLayer(agentv3.SpanLayer_MQ)
	defer span.End()

	klog.Info(metadata)

	in := &dapr.InvokeBindingRequest{
		Name:      "sample-topic",
		Operation: "create",
		Data:      []byte("Hello"),
		Metadata:  metadata,
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
