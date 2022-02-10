package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
	"github.com/SkyAPM/go2sky"
	go2skyHTTP "github.com/SkyAPM/go2sky/plugins/http"
	"k8s.io/klog/v2"
)

var (
	initHttpClientOnce sync.Once
	client             *http.Client
)

func initHTTPClient() {
	initHttpClientOnce.Do(func() {
		client, _ = go2skyHTTP.NewClient(go2sky.GetGlobalTracer())
	})
}

func HelloWorldWithHttp(w http.ResponseWriter, r *http.Request) error {
	initHTTPClient()
	// call end service
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/helloserver", os.Getenv("PROVIDER_ADDRESS")), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Errorf("unable to create http request: %+v\n", err)
		return err
	}

	request = request.WithContext(r.Context())
	res, err := client.Do(request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Write(body)
	return nil
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
