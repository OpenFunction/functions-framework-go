package main

import (
	"context"
	"encoding/json"
	"net/http"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"k8s.io/klog/v2"
)

func init() {
	functions.HTTP("Hello", hello,
		functions.WithFunctionPath("/hello/{name}"),
		functions.WithFunctionMethods("GET", "POST"),
	)

	functions.HTTP("Hellov2", hellov2,
		functions.WithFunctionPath("/hellov2/{name}"),
		functions.WithFunctionMethods("GET", "POST"),
	)

	functions.CloudEvent("Foo", foo,
		functions.WithFunctionPath("/foo/{name}"),
	)

	functions.CloudEvent("Foov2", foov2,
		functions.WithFunctionPath("/foov2/{name}"),
	)

	functions.OpenFunction("Bar", bar,
		functions.WithFunctionPath("/bar/{name}"),
		functions.WithFunctionMethods("GET", "POST"),
	)

	functions.OpenFunction("Barv2", barv2,
		functions.WithFunctionPath("/barv2/{name}"),
		functions.WithFunctionMethods("GET", "POST"),
	)
}

type Message struct {
	Data string `json:"data"`
}

func hello(w http.ResponseWriter, r *http.Request) {
	name := ofctx.URLParamFromCtx(r.Context(), "name")
	response := map[string]string{
		"hello": name,
	}
	responseBytes, _ := json.Marshal(response)
	w.Header().Set("Content-type", "application/json")
	w.Write(responseBytes)
}

func hellov2(w http.ResponseWriter, r *http.Request) {
	// keep for backward compatibility, same for example below
	// suggest to use ofctx.URLParamFromCtx(...) to get vars
	vars := ofctx.VarsFromCtx(r.Context())
	response := map[string]string{
		"hello": vars["name"],
	}
	responseBytes, _ := json.Marshal(response)
	w.Header().Set("Content-type", "application/json")
	w.Write(responseBytes)
}

func foo(ctx context.Context, ce cloudevents.Event) error {
	msg := &Message{}
	err := json.Unmarshal(ce.Data(), msg)
	if err != nil {
		return err
	}

	name := ofctx.URLParamFromCtx(ctx, "name")
	response := map[string]string{
		msg.Data: name,
	}
	responseBytes, _ := json.Marshal(response)
	klog.Infof("cloudevent - Data: %s", string(responseBytes))
	return nil
}

func foov2(ctx context.Context, ce cloudevents.Event) error {
	vars := ofctx.VarsFromCtx(ctx)

	msg := &Message{}
	err := json.Unmarshal(ce.Data(), msg)
	if err != nil {
		return err
	}

	response := map[string]string{
		msg.Data: vars["name"],
	}
	responseBytes, _ := json.Marshal(response)
	klog.Infof("cloudevent - Data: %s", string(responseBytes))
	return nil
}

func bar(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	msg := &Message{}
	err := json.Unmarshal(in, msg)
	if err != nil {
		return ctx.ReturnOnInternalError(), err
	}

	name := ofctx.URLParamFromCtx(ctx.GetNativeContext(), "name")
	response := map[string]string{
		msg.Data: name,
	}
	responseBytes, _ := json.Marshal(response)
	return ctx.ReturnOnSuccess().WithData(responseBytes), nil
}

func barv2(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	vars := ofctx.VarsFromCtx(ctx.GetNativeContext())
	msg := &Message{}
	err := json.Unmarshal(in, msg)
	if err != nil {
		return ctx.ReturnOnInternalError(), err
	}

	response := map[string]string{
		msg.Data: vars["name"],
	}
	responseBytes, _ := json.Marshal(response)
	return ctx.ReturnOnSuccess().WithData(responseBytes), nil
}
