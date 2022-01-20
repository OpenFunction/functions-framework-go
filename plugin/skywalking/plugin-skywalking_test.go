package skywalking

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"syscall"
	"testing"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

// HelloWorld writes "Hello, World!" to the HTTP response.
func HelloWorldWithHttp(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprint(w, "Hello, World!\n")
	return nil
}

func HelloWorldWithOpenFunction(ctx ofctx.Context, data []byte) (ofctx.Out, error) {
	return ofctx.Out{
		Code: ofctx.InternalError,
		Data: []byte("Hello, World!\n"),
	}, errors.New("test error")
}

// env FUNC_CONTEXT={"runtime": "Knative", "prePlugins": ["skywalking-v1"], "postPlugins": ["skywalking-v1"]}
// [{"Refs":null,"StartTime":"2022-01-10T15:18:22.150835+08:00","EndTime":"2022-01-10T15:18:26.022976+08:00","OperationName":"/GET/","Peer":"","Layer":3,"ComponentID":1,"Tags":[{"key":"http.method","value":"GET"},{"key":"url","value":"127.0.0.1:8080/"}],"Logs":null,"IsError":false,"SpanType":0,"TraceID":"7e44773a71e511ec974b3c7c3f51d4d3","SegmentID":"7e44778a71e511ec974b3c7c3f51d4d3","SpanID":0,"ParentSpanID":-1,"ParentSegmentID":"","CorrelationContext":{}}]
func TestHTTP(t *testing.T) {
	if err := syscall.Setenv("FUNC_CONTEXT", "{\"runtime\": \"Knative\", \"prePlugins\": [\"skywalking-v1\"], \"postPlugins\": [\"skywalking-v1\"]}"); err != nil {
		t.Error(err)
	}
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		t.Error(err)
	}
	fwk.RegisterPlugins(map[string]plugin.Plugin{
		"skywalking-v1": &PluginSkywalking{},
	})

	err = fwk.Register(ctx, HelloWorldWithHttp)
	if err != nil {
		t.Error(err)
	}

	err = fwk.Register(ctx, HelloWorldWithOpenFunction)
	if err != nil {
		t.Error(err)
	}

	err = fwk.Start(ctx)
	if err != nil {
		t.Error(err)
	}
}

func TestOpenFunc(t *testing.T) {
	if err := syscall.Setenv("FUNC_CONTEXT", "{\"name\": \"test\",\"runtime\": \"Knative\", \"prePlugins\": [\"skywalking-v1\"], \"postPlugins\": [\"skywalking-v1\"]}"); err != nil {
		t.Error(err)
	}
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		t.Error(err)
	}
	fwk.RegisterPlugins(map[string]plugin.Plugin{
		"skywalking-v1": &PluginSkywalking{},
	})

	err = fwk.Register(ctx, HelloWorldWithOpenFunction)
	if err != nil {
		t.Error(err)
	}

	err = fwk.Start(ctx)
	if err != nil {
		t.Error(err)
	}
}
