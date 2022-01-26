package skywalking

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/SkyAPM/go2sky"
	httpP "github.com/SkyAPM/go2sky/plugins/http"
	"k8s.io/klog/v2"
)

var (
	initHttpClientOnce sync.Once
	client             *http.Client
)

func initHTTPClient() {
	initHttpClientOnce.Do(func() {
		client, _ = httpP.NewClient(go2sky.GetGlobalTracer())
	})
}

// HelloWorld writes "Hello, World!" to the HTTP response.
func HelloWorldWithHttp(w http.ResponseWriter, r *http.Request) error {
	initHTTPClient()
	// call end service
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/correlation", "http://127.0.0.1:9090"), nil)
	if err != nil {
		klog.Errorf("unable to create http request: %+v\n", err)
		return err
	}

	request = request.WithContext(r.Context())
	res, err := client.Do(request)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	fmt.Fprint(w, string(body))
	return nil
}

func TestHTTP(t *testing.T) {
	t.Setenv("FUNC_CONTEXT", `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Knative",
  "port": "12345",
  "prePlugins": [],
  "postPlugins": [],
  "pluginsTracing": {
    "enable": true,
    "provider": {
      "name": "skywalking",
      "oapServer": "127.0.0.1:11800"
    },
    "tags": {
      "func": "function-test",
      "layer": "faas",
      "tag1": "value1",
      "tag2": "value2"
    },
    "baggage": {
      "key": "CONSUMER_KEY",
      "value": "test"
    }
  }
}
`)
	t.Setenv("POD_NAME", "function-test-vhct4")
	t.Setenv("POD_NAMESPACE", "test")
	t.Setenv("SW_AGENT_COLLECTOR_GET_AGENT_DYNAMIC_CONFIG_INTERVAL", "-1")
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		t.Error(err)
	}
	fwk.RegisterPlugins(map[string]plugin.Plugin{
		"skywalking": &PluginSkywalking{},
	})

	err = fwk.Register(ctx, HelloWorldWithHttp)
	if err != nil {
		t.Error(err)
	}

	err = fwk.Start(ctx)
	if err != nil {
		t.Error(err)
	}
}
