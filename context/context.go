package context

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync"

	dapr "github.com/dapr/go-sdk/client"
)

var (
	mu             sync.RWMutex
	clientGRPCPort string
)

func GetOpenFunctionContext() (*Context, error) {
	ctx := &Context{
		Inputs:  make(map[string]*Input),
		Outputs: make(map[string]*Output),
	}

	data := os.Getenv("FUNC_CONTEXT")
	if data == "" {
		return nil, errors.New("env FUNC_CONTEXT not found")
	}

	err := json.Unmarshal([]byte(data), ctx)
	if err != nil {
		return nil, err
	}

	switch ctx.Runtime {
	case Async, Knative:
		break
	default:
		return nil, fmt.Errorf("invalid runtime: %s", ctx.Runtime)
	}

	ctx.EventMeta = &EventMetadata{}
	ctx.SyncRequestMeta = &SyncRequestMetadata{}

	if !ctx.InputsIsEmpty() {
		for name, in := range ctx.Inputs {
			switch in.Type {
			case OpenFuncBinding, OpenFuncTopic:
				break
			default:
				return nil, fmt.Errorf("invalid input type %s: %s", name, in.Type)
			}
		}
	}

	if !ctx.OutputIsEmpty() {
		for name, out := range ctx.Outputs {
			switch out.Type {
			case OpenFuncBinding, OpenFuncTopic:
				break
			default:
				return nil, fmt.Errorf("invalid output type %s: %s", name, out.Type)
			}
		}
	}

	if ctx.Port == "" {
		ctx.Port = defaultPort
	} else {
		if _, err := strconv.Atoi(ctx.Port); err != nil {
			return nil, fmt.Errorf("error parsing port: %s", err.Error())
		}
	}

	// When using self-hosted mode, configure the client port via env,
	// refer to https://docs.dapr.io/reference/environment/
	port := os.Getenv("DAPR_GRPC_PORT")
	if port == "" {
		clientGRPCPort = daprSidecarGRPCPort
	} else {
		clientGRPCPort = port
	}

	return ctx, nil
}

func (ctx *Context) Send(outputName string, data []byte) ([]byte, error) {
	if ctx.OutputIsEmpty() {
		return nil, errors.New("no output")
	}

	var err error
	var output *Output
	var response *dapr.BindingEvent

	client := ctx.GetDaprClient()

	if v, ok := ctx.Outputs[outputName]; ok {
		output = v
	} else {
		return nil, fmt.Errorf("output %s not found", outputName)
	}

	switch output.Type {
	case OpenFuncTopic:
		err = client.PublishEvent(context.Background(), output.Component, output.Uri, data)
	case OpenFuncBinding:
		in := &dapr.InvokeBindingRequest{
			Name:      output.Component,
			Operation: output.Operation,
			Data:      data,
			Metadata:  output.Metadata,
		}
		response, err = client.InvokeBinding(context.Background(), in)
	}

	if err != nil {
		return nil, err
	}

	if response != nil {
		return response.Data, nil
	}
	return nil, nil
}

func (ctx *Context) InputsIsEmpty() bool {
	nilInputs := map[string]*Input{}
	if reflect.DeepEqual(ctx.Inputs, nilInputs) {
		return true
	}
	return false
}

func (ctx *Context) OutputIsEmpty() bool {
	nilOutputs := map[string]*Output{}
	if reflect.DeepEqual(ctx.Outputs, nilOutputs) {
		return true
	}
	return false
}

func (ctx *Context) ReturnOnSuccess() Out {
	return Out{
		Code: Success,
	}
}

func (ctx *Context) ReturnOnInternalError() Out {
	return Out{
		Code: InternalError,
	}
}

func (ctx *Context) GetDaprClient() dapr.Client {
	return ctx.daprClient
}

func InitDaprClientIfNil(ctx *Context) {
	if ctx.daprClient == nil {
		mu.Lock()
		defer mu.Unlock()
		c, e := dapr.NewClientWithPort(clientGRPCPort)
		if e != nil {
			panic(e)
		}
		ctx.daprClient = c
	}
}

func DestroyDaprClient(ctx *Context) {
	if ctx.daprClient != nil {
		mu.Lock()
		defer mu.Unlock()
		ctx.daprClient.Close()
		ctx.daprClient = nil
	}
}
