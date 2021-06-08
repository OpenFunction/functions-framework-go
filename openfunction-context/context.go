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
	ctx := &OpenFunctionContext{}

	data := os.Getenv("FUNC_CONTEXT")
	if data == "" {
		return nil, errors.New("FUNC_CONTEXT not found")
	}

	err := json.Unmarshal([]byte(data), ctx)
	if err != nil {
		return nil, err
	}

	if *ctx.Input.Enabled {
		switch ctx.Protocol {
		case GRPC:
			if ctx.Input.Pattern == "" {
				ctx.Input.Pattern = ctx.Input.Name
			}
			err = nil
		case HTTP:
			if ctx.Input.Pattern == "" {
				ctx.Input.Pattern = "/" + ctx.Input.Name
			}
			err = nil
		default:
			err = errors.New("invalid input kind")
		}
		if err != nil {
			return nil, err
		}
	}

	switch ctx.Runtime {
	case Dapr:
		err = nil
	case Knative:
		err = nil
	default:
		err = errors.New("invalid runtime")
	}
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

func (ctx *OpenFunctionContext) SendTo(data []byte, outputName string) error {
	if !*ctx.Outputs.Enabled {
		return errors.New("no output")
	}

	var err error
	var op *Output
	var client dapr.Client
	var method = ""
	if v, ok := ctx.Outputs.OutputObjects[outputName]; ok {
		op = v
	} else {
		return fmt.Errorf("output %s not found", outputName)
	}

	if m, ok := op.Params["method"]; ok {
		method = m
	}

	if ctx.Runtime == Dapr {
		c, err := dapr.NewClient()
		if err != nil {
			panic(err)
		}
		client = c
		switch op.OutType {
		case DaprTopic:
			err = client.PublishEvent(context.Background(), outputName, op.Pattern, data)
		case DaprService:
			if method != "" {
				content := &dapr.DataContent{
					ContentType: "application/json",
					Data:        data,
				}
				_, err = client.InvokeMethodWithContent(context.Background(), outputName, op.Pattern, method, content)
			} else {
				err = errors.New("output method is empty or invalid")
			}
		case DaprBinding:
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
			_, err = doHttpRequest(method, op.Pattern, ApplicationJson, bytes.NewReader(data))
		} else {
			err = errors.New("output method is empty or invalid")
		}
	}

	if err != nil {
		client.Close()
		return err
	}

	return nil
}

func (ctx *OpenFunctionContext) GetInput() (interface{}, error) {
	var data interface{}
	//content, err := ioutil.ReadAll(v)
	//json.Unmarshal(content, &data)
	//if err != nil {
	//	return nil, err
	//}

	return data, nil
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
