package skywalking

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

// HelloWorld writes "Hello, World!" to the HTTP response.
func HelloWorldWithHttp(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprint(w, "Hello, World!\n")
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
      "key": "sw8-correlation",
      "value": "base64(string key):base64(string value),base64(string key2):base64(string value2)"
    }
  }
}
`)
	t.Setenv("POD_NAME", "prometheus-node-exporter-vhct4")
	t.Setenv("POD_NAMESPACE", "test")
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
