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
func HelloWorld(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprint(w, "Hello, World!\n")
	return nil
}

// env FUNC_CONTEXT={"runtime": "Knative", "prePlugins": ["skywalking-v1"], "postPlugins": ["skywalking-v1"]}
// [{"Refs":null,"StartTime":"2022-01-10T15:18:22.150835+08:00","EndTime":"2022-01-10T15:18:26.022976+08:00","OperationName":"/GET/","Peer":"","Layer":3,"ComponentID":1,"Tags":[{"key":"http.method","value":"GET"},{"key":"url","value":"127.0.0.1:8080/"}],"Logs":null,"IsError":false,"SpanType":0,"TraceID":"7e44773a71e511ec974b3c7c3f51d4d3","SegmentID":"7e44778a71e511ec974b3c7c3f51d4d3","SpanID":0,"ParentSpanID":-1,"ParentSegmentID":"","CorrelationContext":{}}]
func TestPluginSkywalking(t *testing.T) {
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		t.Error(err)
	}
	err = fwk.Register(ctx, HelloWorld)
	if err != nil {
		t.Error(err)
	}

	fwk.RegisterPlugins(map[string]plugin.Plugin{
		"skywalking-v1": New(),
	})

	err = fwk.Start(ctx)
	if err != nil {
		t.Error(err)
	}
}
