package openfunctioncontext

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	dapr "github.com/dapr/go-sdk/client"
	"io"
	"net/http"
	"os"
	"reflect"
	"time"
)

const (
	ApplicationJson   = "application/json"
	HTTPTimeoutSecond = 60
)

var (
	httpNormalResponse = map[int]string{
		200: "Request successful",
		204: "Empty Response",
	}
	httpErrorResponse = map[int]string{
		403: "Invocation forbidden by access control",
		404: "Not Found",
		400: "Method name not given",
		500: "Request failed",
	}
)

// ContextInterface represents Dapr callback service
type ContextInterface interface {
	SendTo(data []byte, outputName string) error
	GetInput() (interface{}, error)
}

func GetOpenFunctionContext() (*OpenFunctionContext, error) {
	ctx := &OpenFunctionContext{
		Input:   Input{},
		Outputs: make(map[string]*Output),
	}

	data := os.Getenv("FUNC_CONTEXT")
	if data == "" {
		return nil, errors.New("FUNC_CONTEXT not found")
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

	if !ctx.InputIsEmpty() {
		if ctx.Runtime == OpenFuncAsync {
			if _, ok := ctx.Input.Params["type"]; !ok {
				return nil, errors.New("invalid input: missing type")
			}
		}
	}

	if !ctx.OutputIsEmpty() {
		if ctx.Runtime == OpenFuncAsync {
			for name, out := range ctx.Outputs {
				if _, ok := out.Params["type"]; !ok {
					return nil, fmt.Errorf("invalid output %s: missing type", name)
				}
			}
		}
	}

	return ctx, nil
}

func (ctx *OpenFunctionContext) SendTo(data []byte, outputName string) error {
	if ctx.OutputIsEmpty() {
		return errors.New("no output")
	}

	var err error
	var op *Output
	var client dapr.Client
	var method = ""
	if v, ok := ctx.Outputs[outputName]; ok {
		op = v
	} else {
		return fmt.Errorf("output %s not found", outputName)
	}

	if m, ok := op.Params["method"]; ok {
		method = m
	}

	if ctx.Runtime == OpenFuncAsync {
		c, err := dapr.NewClient()
		if err != nil {
			panic(err)
		}
		client = c
		outType := op.Params["type"]
		switch ResourceType(outType) {
		case OpenFuncTopic:
			err = client.PublishEvent(context.Background(), outputName, op.Uri, data)
		case OpenFuncService:
			if method != "" {
				content := &dapr.DataContent{
					ContentType: "application/json",
					Data:        data,
				}
				_, err = client.InvokeMethodWithContent(context.Background(), outputName, op.Uri, method, content)
			} else {
				err = errors.New("output method is empty or invalid")
			}
		case OpenFuncBinding:
			var metadata map[string]string
			if md, ok := op.Params["metadata"]; ok {
				err = json.Unmarshal([]byte(md), &metadata)
				if err != nil {
					break
				}
			}

			in := &dapr.InvokeBindingRequest{
				Name:      outputName,
				Operation: op.Params["operation"],
				Data:      data,
				Metadata:  metadata,
			}
			err = client.InvokeOutputBinding(context.Background(), in)
		}

	} else {
		if method != "" {
			_, err = doHttpRequest(method, op.Uri, ApplicationJson, bytes.NewReader(data))
		} else {
			err = errors.New("output method is empty or invalid")
		}
	}

	if err != nil && client != nil {
		client.Close()
		return err
	}

	return nil
}

func (ctx *OpenFunctionContext) InputIsEmpty() bool {
	nilInput := Input{}
	if reflect.DeepEqual(ctx.Input, nilInput) {
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

func doHttpRequest(method string, url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeoutSecond*time.Second)
	defer cancel()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	rsp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if _, ok := httpNormalResponse[rsp.StatusCode]; ok {
		return rsp, nil
	}

	if rspText, ok := httpErrorResponse[rsp.StatusCode]; ok {
		return rsp, errors.New(rspText)
	}

	return rsp, errors.New("unrecognized response code")
}
