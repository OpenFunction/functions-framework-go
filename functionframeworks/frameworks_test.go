package functionframeworks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gofrs/uuid"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRegisterHTTPFunction(t *testing.T) {
	h := http.NewServeMux()
	err := registerHTTPFunction("/", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "Hello World!")
		return nil
	}, h)
	if err != nil {
		t.Fatalf("Error: %v\n", err)
	}

	srv := httptest.NewServer(h)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll: %v", err)
	}

	if got, want := string(body), "Hello World!"; got != want {
		t.Fatalf("TestHTTPFunction: got %v; want %v", got, want)
	}
}

func TestCloudEventFunction(t *testing.T) {
	cloudeventsJSON := []byte(`{
		"specversion" : "1.0",
		"type" : "com.github.pull.create",
		"source" : "https://github.com/cloudevents/spec/pull",
		"subject" : "123",
		"id" : "A234-1234-1234",
		"time" : "2018-04-05T17:31:00Z",
		"comexampleextension1" : "value",
		"datacontenttype" : "application/xml",
		"data" : "<much wow=\"xml\"/>"
	}`)
	var testCE cloudevents.Event
	err := json.Unmarshal(cloudeventsJSON, &testCE)
	if err != nil {
		t.Fatalf("TestCloudEventFunction: unable to create Event from JSON: %v", err)
	}

	var tests = []struct {
		name      string
		data      []byte
		fn        func(context.Context, cloudevents.Event) error
		status    int
		header    string
		ceHeaders map[string]string
	}{
		{
			name: "binary cloudevent",
			data: []byte("<much wow=\"xml\"/>"),
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(binary cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
			ceHeaders: map[string]string{
				"ce-specversion":          "1.0",
				"ce-type":                 "com.github.pull.create",
				"ce-source":               "https://github.com/cloudevents/spec/pull",
				"ce-subject":              "123",
				"ce-id":                   "A234-1234-1234",
				"ce-time":                 "2018-04-05T17:31:00Z",
				"ce-comexampleextension1": "value",
				"Content-Type":            "application/xml",
			},
		},
		{
			name: "structured cloudevent",
			data: cloudeventsJSON,
			fn: func(ctx context.Context, e cloudevents.Event) error {
				if e.String() != testCE.String() {
					return fmt.Errorf("TestCloudEventFunction(structured cloudevent): got: %v, want: %v", e, testCE)
				}
				return nil
			},
			status: http.StatusOK,
			header: "",
			ceHeaders: map[string]string{
				"Content-Type": "application/cloudevents+json",
			},
		},
	}

	for _, tc := range tests {
		ctx := context.Background()
		h := http.NewServeMux()
		if err := registerCloudEventFunction(ctx, tc.fn, h); err != nil {
			t.Fatalf("registerCloudEventFunction(): %v", err)
		}

		srv := httptest.NewServer(h)
		defer srv.Close()

		req, err := http.NewRequest("POST", srv.URL, bytes.NewBuffer(tc.data))
		for k, v := range tc.ceHeaders {
			req.Header.Add(k, v)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("client.Do(%s): %v", tc.name, err)
			continue
		}

		if resp.StatusCode != tc.status {
			gotBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read got request body: %v", err)
			}
			t.Errorf("TestCloudEventFunction(%s): response status = %v, want %v, %q.", tc.name, resp.StatusCode, tc.status, string(gotBody))
		}
		if resp.Header.Get(functionStatusHeader) != tc.header {
			t.Errorf("TestCloudEventFunction(%s): response header = %q, want %q", tc.name, resp.Header.Get(functionStatusHeader), tc.header)
		}
	}
}

func TestRegisterOpenFunction(t *testing.T) {
	h := http.NewServeMux()
	requestID, err := uuid.NewV4()
	ofctx := &OpenFunctionContext{
		Name:      "Bindings",
		Version:   "v1",
		RequestID: requestID.String(),
		Input: &Input{
			Enabled: func(v bool) *bool { return &v }(true),
			Url:     "/sample",
		},
		Outputs: &Outputs{
			Enabled: func(v bool) *bool { return &v }(true),
			OutputObjects: map[string]*Output{
				"OP1": {
					Url:    "http://localhost:3500/bingdings/op1",
					Method: "POST",
				},
				"OP2": {
					Url:    "http://localhost:3500/pubsub/op2",
					Method: "PUT",
				},
			},
		},
		Runtime: "Dapr",
	}
	ofctxByte, err := json.Marshal(ofctx)
	os.Setenv("FUNC_CONTEXT", string(ofctxByte))

	err = registerOpenFunction(func(ctx *OpenFunctionContext, r *http.Request) int {
		_, e := ctx.GetInput(r)
		if e != nil {
			return 500
		}

		e = ctx.SendTo("Hello", "OP1")
		if e != nil {
			return 500
		}
		return 200
	}, h)
	if err != nil {
		t.Fatalf("Error: %v\n", err)
	}

	srv := httptest.NewServer(h)

	inputData, err := json.Marshal("Hello World!")
	url := srv.URL + ofctx.Input.Url
	resp, err := http.Post(url, "application/json", strings.NewReader(string(inputData)))
	if err != nil {
		t.Fatalf("http.Post: %v", err)
	}

	if got, want := resp.StatusCode, 500; got != want {
		t.Fatalf("TestOpenFunction: got %v; want %v", got, want)
	}
}
