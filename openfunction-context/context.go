package openfunctioncontext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	dapr "github.com/dapr/go-sdk/client"
)

// ContextInterface represents Dapr callback service
type ContextInterface interface {
	SendTo(data []byte, outputName string) error
	GetInput() (interface{}, error)
}

func GetOpenFunctionContext() (*OpenFunctionContext, error) {
	ctx := &OpenFunctionContext{
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
	case OpenFuncAsync, Knative:
		break
	default:
		return nil, fmt.Errorf("invalid runtime: %s", ctx.Runtime)
	}

	if ctx.Runtime == OpenFuncAsync {
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
	}
	ctx.Event = &EventMetadata{}

	return ctx, nil
}

func (ctx *OpenFunctionContext) Send(outputName string, data []byte) ([]byte, error) {
	if ctx.OutputIsEmpty() {
		return nil, errors.New("no output")
	}

	var err error
	var output *Output
	var client dapr.Client
	var response *dapr.BindingEvent
	if v, ok := ctx.Outputs[outputName]; ok {
		output = v
	} else {
		return nil, fmt.Errorf("output %s not found", outputName)
	}

	if ctx.Runtime == OpenFuncAsync {
		c, e := dapr.NewClient()
		if e != nil {
			panic(e)
		}
		client = c
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

	} else {
		err = errors.New("the SendTo need OpenFuncAsync runtime")
	}

	if err != nil && client != nil {
		client.Close()
		return nil, err
	}

	if response != nil {
		return response.Data, nil
	}
	return nil, nil
}

func (ctx *OpenFunctionContext) SendTo(data []byte, outputName string) error {
	if ctx.OutputIsEmpty() {
		return errors.New("no output")
	}

	var err error
	var output *Output
	var client dapr.Client
	if v, ok := ctx.Outputs[outputName]; ok {
		output = v
	} else {
		return fmt.Errorf("output %s not found", outputName)
	}

	if ctx.Runtime == OpenFuncAsync {
		c, e := dapr.NewClient()
		if e != nil {
			panic(e)
		}
		client = c
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
			_, err = client.InvokeBinding(context.Background(), in)
		}

	} else {
		err = errors.New("the SendTo need OpenFuncAsync runtime")
	}

	if err != nil && client != nil {
		client.Close()
		return err
	}

	return nil
}

func (ctx *OpenFunctionContext) InputsIsEmpty() bool {
	nilInputs := map[string]*Input{}
	if reflect.DeepEqual(ctx.Inputs, nilInputs) {
		return true
	}
	return false
}

func (ctx *OpenFunctionContext) OutputIsEmpty() bool {
	nilOutputs := map[string]*Output{}
	if reflect.DeepEqual(ctx.Outputs, nilOutputs) {
		return true
	}
	return false
}

func (ctx *OpenFunctionContext) ReturnWithSuccess() Return {
	return Return{
		Code: Success,
	}
}

func (ctx *OpenFunctionContext) ReturnWithInternalError() Return {
	return Return{
		Code: InternalError,
	}
}
