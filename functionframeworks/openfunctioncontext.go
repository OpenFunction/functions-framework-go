package functionframeworks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	applicationJson                = "application/json"
	Dapr              Runtime      = "Dapr"
	Knative           Runtime      = "Knative"
	HTTPTimeoutSecond              = 60
	HTTPPut           OutputMethod = "PUT"
	HTTPPost          OutputMethod = "POST"
)

var (
	contentType        string
	httpNormalResponse = map[int]string{
		200: "Request successful",
		204: "Empty Response",
	}
	httpErrorResponse = map[int]string{
		404: "Not Found",
		400: "Malformed request",
		500: "Request failed",
	}
)

type Runtime string

type OutputMethod string

type Output struct {
	Url string `json:"url"`
	// Method indicates the way Output is used. Optional value for OutputMethod is HTTPPut, HTTPPost
	Method OutputMethod `json:"method"`
}

type Input struct {
	Enabled *bool  `json:"enabled"`
	Url     string `json:"url"`
}

type Outputs struct {
	Enabled       *bool              `json:"enabled"`
	OutputObjects map[string]*Output `json:"output_objects"`
}

type OpenFunctionContext struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	RequestID string   `json:"request_id,omitempty"`
	Input     *Input   `json:"input,omitempty"`
	Outputs   *Outputs `json:"outputs,omitempty"`
	// Runtime, Knative or Dapr
	Runtime Runtime `json:"runtime"`
}

type OpenFunctionContextInterface interface {
	SendTo(data interface{}, outputName string) error
	GetInput(r *http.Request) (interface{}, error)
	// SendAll(data interface{}) error
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

	for _, op := range ctx.Outputs.OutputObjects {
		switch op.Method {
		case HTTPPost:
			err = nil
		case HTTPPut:
			err = nil
		default:
			err = errors.New("invalid output method")
		}
	}
	if err != nil {
		return nil, err
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

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if ctx.Runtime == Dapr {
		if op, ok := ctx.Outputs.OutputObjects[outputName]; ok {
			_, err = doHttpRequest(op.Method, op.Url, applicationJson, strings.NewReader(string(body)))
		} else {
			err = fmt.Errorf("output %s not found", outputName)
		}
	}
	if err != nil {
		return err
	}

	return nil
}

//func (ctx *OpenFunctionContext) SendAll(data interface{}) error {
//	if !*ctx.Outputs.Enabled {
//		return errors.New("no output")
//	}
//
//	body, err := json.Marshal(data)
//	if err != nil {
//		return err
//	}
//
//	if ctx.Runtime == Dapr {
//		for _, op := range ctx.Outputs.OutputObjects {
//			_, err = doHttpRequest(op.Method, op.Url, applicationJson, strings.NewReader(string(body)))
//		}
//	}
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

func (ctx *OpenFunctionContext) GetInput(r *http.Request) (interface{}, error) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var data interface{}
	json.Unmarshal([]byte(content), &data)

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
