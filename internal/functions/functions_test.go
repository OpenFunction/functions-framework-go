package functions

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"reflect"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func TestNewHTTPFunction(t *testing.T) {

	name := "foo"
	path := "/foo"
	methods := []string{"GET", "POST"}
	fn, err := New(WithFunctionName(name), WithFunctionPath(path), WithFunctionMethods(methods...), WithHTTP(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World!")
	}))
	if err != nil {
		t.Fatalf("Fail to Create http function with name: %s, path: %s", name, path)
	}

	if fn.GetFunctionType() != HTTPType {
		t.Errorf("Expected function type to be %s, got %s", HTTPType, fn.GetFunctionType())
	}

	if fn.GetName() != name {
		t.Errorf("Expected function name to be %s, got %s", name, fn.GetName())
	}

	if fn.GetPath() != path {
		t.Errorf("Expected function path to be %s, got %s", path, fn.GetPath())
	}

	if !reflect.DeepEqual(fn.GetFunctionMethods(), methods) {
		t.Errorf("Expected function methods to be %s, got %s", methods, fn.GetFunctionMethods())
	}
}

func TestNewCloudEventFunctionWithMethods(t *testing.T) {

	name := "foo"
	path := "/foo"
	methods := []string{"GET", "POST"}
	_, err := New(WithFunctionName(name), WithFunctionPath(path), WithFunctionMethods(methods...), WithCloudEvent(func(context.Context, cloudevents.Event) error {
		return nil
	}))
	if err == nil {
		t.Error("Expected fail to create CloudEvent function with Methods, but succeed")
	}
}

func TestNewCloudEventFunction(t *testing.T) {

	name := "foo"
	path := "/foo"
	fn, err := New(WithFunctionName(name), WithFunctionPath(path), WithCloudEvent(func(context.Context, cloudevents.Event) error {
		return nil
	}))
	if err != nil {
		t.Fatalf("Fail to Create cloudevent function with name: %s, path: %s, error: %s", name, path, err)
	}

	if fn.GetFunctionType() != CloudEventType {
		t.Errorf("Expected function type to be %s, got %s", CloudEventType, fn.GetFunctionType())
	}

	if fn.GetName() != name {
		t.Errorf("Expected function name to be %s, got %s", name, fn.GetName())
	}

	if fn.GetPath() != path {
		t.Errorf("Expected function path to be %s, got %s", path, fn.GetPath())
	}
}

func TestNewOpenFunctionFunction(t *testing.T) {

	name := "foo"
	path := "/foo"
	methods := []string{"GET", "POST"}
	fn, err := New(WithFunctionName(name), WithFunctionPath(path), WithFunctionMethods(methods...), WithOpenFunction(func(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
		return ctx.ReturnOnSuccess(), nil
	}))
	if err != nil {
		t.Fatalf("Fail to Create openfunction function with name: %s, path: %s", name, path)
	}

	if fn.GetFunctionType() != OpenFunctionType {
		t.Errorf("Expected function type to be %s, got %s", OpenFunctionType, fn.GetFunctionType())
	}

	if fn.GetName() != name {
		t.Errorf("Expected function name to be %s, got %s", name, fn.GetName())
	}

	if fn.GetPath() != path {
		t.Errorf("Expected function path to be %s, got %s", path, fn.GetPath())
	}

	if !reflect.DeepEqual(fn.GetFunctionMethods(), methods) {
		t.Errorf("Expected function methods to be %s, got %s", methods, fn.GetFunctionMethods())
	}
}
