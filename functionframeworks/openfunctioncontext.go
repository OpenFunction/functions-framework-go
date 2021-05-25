package functionframeworks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	applicationJson                        = "application/json"
	httpPost              HttpMethod       = "Post"
	httpPut               HttpMethod       = "Put"
	Dapr                  Runtime          = "Dapr"
	Knative               Runtime          = "Knative"
	httpRequestSuccessful HttpResponseCode = 200
	httpEmptyResponse     HttpResponseCode = 204
	httpNotFound          HttpResponseCode = 404
	httpMalformedRequest  HttpResponseCode = 400
	httpRequestFailed     HttpResponseCode = 500
)

var (
	contentType string
)

type Runtime string

type HttpMethod string

type HttpResponseCode int

type Output struct {
	Url    string     `json:"url"`
	Method HttpMethod `json:"method"`
}

type Input struct {
	Enabled *bool  `json:"enabled"`
	Url     string `json:"url"`
}

type Outputs struct {
	Enabled       *bool              `json:"enabled"`
	OutputObjects map[string]*Output `json:"output_objects"`
}

type Response struct {
	Code HttpResponseCode
}

type OpenFunctionContext struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	RequestID string   `json:"request_id,omitempty"`
	Input     *Input   `json:"input,omitempty"`
	Outputs   *Outputs `json:"outputs,omitempty"`
	// Runtime, Knative or Dapr
	Runtime  Runtime   `json:"runtime"`
	Response *Response `json:"response"`
}

type OpenFunctionContextInterface interface {
	SendTo(data interface{}, outputName string) error
	SendAll(data interface{}) error
	GetInput(r *http.Request) (interface{}, error)
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
			if op.Method == httpPost {
				_, err = http.Post(op.Url, applicationJson, strings.NewReader(string(body)))
			} else if op.Method == httpPut {
				_, err = doHttpPut(op.Url, applicationJson, strings.NewReader(string(body)))
			}
		} else {
			err = fmt.Errorf("output %s not found", outputName)
		}
	}
	if err != nil {
		return err
	}

	return nil
}

func (ctx *OpenFunctionContext) SendAll(data interface{}) error {
	if !*ctx.Outputs.Enabled {
		return errors.New("no output")
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if ctx.Runtime == Dapr {
		for _, op := range ctx.Outputs.OutputObjects {
			if op.Method == httpPost {
				_, err = http.Post(op.Url, applicationJson, strings.NewReader(string(body)))
			} else if op.Method == httpPut {
				_, err = doHttpPut(op.Url, applicationJson, strings.NewReader(string(body)))
			}
		}
	}
	if err != nil {
		return err
	}

	return nil
}

func (ctx *OpenFunctionContext) GetInput(r *http.Request) (interface{}, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func doHttpPut(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return http.DefaultClient.Do(req)
}

func (ctx *OpenFunctionContext) MakeResponse(code int) {
	ctx.Response.Code = HttpResponseCode(code)
}
