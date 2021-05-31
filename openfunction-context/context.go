package openfunctioncontext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	applicationJson   = "application/json"
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
	SendTo(data interface{}, outputName string) error
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
		switch ctx.Input.Kind {
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

	if *ctx.Outputs.Enabled && ctx.Outputs.OutputObjects != nil {
		for _, op := range ctx.Outputs.OutputObjects {
			switch op.ReqMethod {
			case HTTPPost:
				err = nil
			case HTTPPut:
				err = nil
			case HTTPGet:
				err = nil
			case HTTPDelete:
				err = nil
			default:
				err = errors.New("invalid output method")
			}
			if err != nil {
				return nil, err
			}
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

func (ctx *OpenFunctionContext) SendTo(data interface{}, outputName string) error {
	if !*ctx.Outputs.Enabled {
		return errors.New("no output")
	}

	var op *Output
	if v, ok := ctx.Outputs.OutputObjects[outputName]; ok {
		op = v
	} else {
		return fmt.Errorf("output %s not found", outputName)
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if ctx.Runtime == Dapr {
		pattern := strings.Trim(op.Pattern, "/")
		_, err = doHttpRequest(op.ReqMethod, pattern, applicationJson, strings.NewReader(string(body)))
	}
	if err != nil {
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

func doHttpRequest(method OutputMethod, url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeoutSecond*time.Second)
	defer cancel()

	req, err := http.NewRequest(string(method), url, body)
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
