package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/dapr/dapr/pkg/proto/runtime/v1"
	"github.com/dapr/go-sdk/service/common"
	"github.com/stretchr/testify/assert"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/functions"
	"github.com/OpenFunction/functions-framework-go/runtime/async"
)

func fakeHTTPFunction(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello World!")
}

func fakeCloudEventsFunction(ctx context.Context, ce cloudevents.Event) error {
	fmt.Println(string(ce.Data()))
	return nil
}

func fakeBindingsFunction(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	if in != nil {
		log.Printf("binding - Data: %s", in)
	} else {
		log.Print("binding - Data: Received")
	}
	return ctx.ReturnOnSuccess().WithData([]byte("hello there")), nil
}

func fakePubsubFunction(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	if in != nil {
		log.Printf("pubsub - Data: %s", in)
	} else {
		log.Print("pubsub - Data: Received")
	}
	return ctx.ReturnOnSuccess().WithData([]byte("hello there")), nil
}

func TestHTTPFunction(t *testing.T) {
	env := `{
  "name": "function-demo",
  "version": "v1.0.0",
  "port": "8080",
  "runtime": "Knative",
  "httpPattern": "/http"
}`
	ctx := context.Background()
	fwk, err := createFramework(env)
	if err != nil {
		t.Fatalf("failed to create framework: %v", err)
	}

	fwk.RegisterPlugins(nil)

	if err := fwk.Register(ctx, fakeHTTPFunction); err != nil {
		t.Fatalf("failed to register HTTP function: %v\n", err)
	}

	if fwk.GetRuntime() == nil {
		t.Fatal("failed to create runtime")
	}
	handler := fwk.GetRuntime().GetHandler()
	if handler == nil {
		t.Fatal("handler is nil")
	}

	srv := httptest.NewServer(handler.(http.Handler))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/http")
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
	env := `{
  "name": "function-demo",
  "version": "v1.0.0",
  "port": "8080",
  "runtime": "Knative",
  "httpPattern": "/ce"
}`
	var ceDemo = struct {
		message map[string]string
		headers map[string]string
	}{
		message: map[string]string{
			"msg": "Hello World!",
		},
		headers: map[string]string{
			"Ce-Specversion": "1.0",
			"Ce-Type":        "cloudevents.openfunction.samples.helloworld",
			"Ce-Source":      "cloudevents.openfunction.samples/helloworldsource",
			"Ce-Id":          "536808d3-88be-4077-9d7a-a3f162705f79",
		},
	}

	ctx := context.Background()
	fwk, err := createFramework(env)
	if err != nil {
		t.Fatalf("failed to create framework: %v", err)
	}

	fwk.RegisterPlugins(nil)

	if err := fwk.Register(ctx, fakeCloudEventsFunction); err != nil {
		t.Fatalf("failed to register CloudEvents function: %v", err)
	}

	if fwk.GetRuntime() == nil {
		t.Fatal("failed to create runtime")
	}

	handler := fwk.GetRuntime().GetHandler()
	if handler == nil {
		t.Fatal("handler is nil")
	}

	srv := httptest.NewServer(handler.(http.Handler))
	defer srv.Close()

	messageByte, err := json.Marshal(ceDemo.message)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	req, err := http.NewRequest("POST", srv.URL+"/ce", bytes.NewBuffer(messageByte))
	if err != nil {
		t.Fatalf("error creating HTTP request for test: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range ceDemo.headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to do client.Do: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to test cloudevents function: response status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

func TestMultipleFunctions(t *testing.T) {
	env := `{
  "name": "function-demo",
  "version": "v1.0.0",
  "port": "8080",
  "runtime": "Knative",
  "httpPattern": "/"
}`
	var ceDemo = struct {
		message map[string]string
		headers map[string]string
	}{
		message: map[string]string{
			"msg": "Hello World!",
		},
		headers: map[string]string{
			"Ce-Specversion": "1.0",
			"Ce-Type":        "cloudevents.openfunction.samples.helloworld",
			"Ce-Source":      "cloudevents.openfunction.samples/helloworldsource",
			"Ce-Id":          "536808d3-88be-4077-9d7a-a3f162705f79",
		},
	}

	ctx := context.Background()
	fwk, err := createFramework(env)
	if err != nil {
		t.Fatalf("failed to create framework: %v", err)
	}

	fwk.RegisterPlugins(nil)

	// register multiple functions
	functions.HTTP("http", fakeHTTPFunction,
		functions.WithFunctionPath("/http"),
		functions.WithFunctionMethods("GET"),
	)

	functions.CloudEvent("ce", fakeCloudEventsFunction,
		functions.WithFunctionPath("/ce"),
	)

	functions.OpenFunction("ofn", fakeBindingsFunction,
		functions.WithFunctionPath("/ofn"),
		functions.WithFunctionMethods("GET", "POST"),
	)

	if err := fwk.TryRegisterFunctions(ctx); err != nil {
		t.Fatalf("failed to start registering functions: %v", err)
	}

	if fwk.GetRuntime() == nil {
		t.Fatal("failed to create runtime")
	}
	handler := fwk.GetRuntime().GetHandler()
	if handler == nil {
		t.Fatal("handler is nil")
	}

	srv := httptest.NewServer(handler.(http.Handler))
	defer srv.Close()

	// test http
	t.Run("sending http", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/http")
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
	})

	// test http to openfunction
	t.Run("sending http to openfunction", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/ofn")
		if err != nil {
			t.Fatalf("http.Get: %v", err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ioutil.ReadAll: %v", err)
		}

		if got, want := string(body), "hello there"; got != want {
			t.Fatalf("TestHTTPFunction: got %v; want %v", got, want)
		}
	})

	// test cloudevent
	t.Run("sending cloudevent", func(t *testing.T) {
		messageByte, err := json.Marshal(ceDemo.message)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		req, err := http.NewRequest("POST", srv.URL+"/ce", bytes.NewBuffer(messageByte))
		if err != nil {
			t.Fatalf("error creating HTTP request for test: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range ceDemo.headers {
			req.Header.Set(k, v)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to do client.Do: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("failed to test cloudevents function: response status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	})

	// test cloudevent to openfunction
	t.Run("sending cloudevent to openfunction", func(t *testing.T) {
		messageByte, err := json.Marshal(ceDemo.message)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		req, err := http.NewRequest("POST", srv.URL+"/ofn", bytes.NewBuffer(messageByte))
		if err != nil {
			t.Fatalf("error creating HTTP request for test: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range ceDemo.headers {
			req.Header.Set(k, v)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to do client.Do: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("failed to test cloudevents function: response status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	})

}

func TestAsyncBindingsFunction(t *testing.T) {
	env := `{
  "name": "function-demo",
  "version": "v1",
  "runtime": "Async",
  "requestID": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "port": "50003",
  "inputs": {
    "cron": {
      "uri": "test",
      "componentName": "test",
      "componentType": "bindings.Kafka"
    }
  }
}`
	methodName := "test"
	ctx := context.Background()
	fwk, err := createFramework(env)
	if err != nil {
		t.Fatalf("failed to create framework: %v", err)
	}

	fwk.RegisterPlugins(nil)

	if err := fwk.Register(ctx, fakeBindingsFunction); err != nil {
		t.Fatalf("failed to register CloudEvents function: %v", err)
	}

	if fwk.GetRuntime() == nil {
		t.Fatal("failed to create runtime")
	}

	server := fwk.GetRuntime().GetHandler()
	if server == nil {
		t.Fatal("server is nil")
	}
	s := server.(*async.FakeServer)
	startTestServer(s)

	t.Run("binding without event", func(t *testing.T) {
		_, err := s.OnBindingEvent(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("binding event for wrong method", func(t *testing.T) {
		in := &runtime.BindingEventRequest{Name: "invalid"}
		_, err := s.OnBindingEvent(ctx, in)
		assert.Error(t, err)
	})

	t.Run("binding event without data", func(t *testing.T) {
		in := &runtime.BindingEventRequest{Name: methodName}
		out, err := s.OnBindingEvent(ctx, in)
		assert.NoError(t, err)
		assert.NotNil(t, out)
	})

	t.Run("binding event with data", func(t *testing.T) {
		data := "hello there"
		in := &runtime.BindingEventRequest{
			Name: methodName,
			Data: []byte(data),
		}
		out, err := s.OnBindingEvent(ctx, in)
		assert.NoError(t, err)
		assert.NotNil(t, out)
		assert.Equal(t, data, string(out.Data))
	})

	t.Run("binding event with metadata", func(t *testing.T) {
		in := &runtime.BindingEventRequest{
			Name:     methodName,
			Metadata: map[string]string{"k1": "v1", "k2": "v2"},
		}
		out, err := s.OnBindingEvent(ctx, in)
		assert.NoError(t, err)
		assert.NotNil(t, out)
	})

	stopTestServer(t, s)

}

func TestAsyncPubsubTopic(t *testing.T) {
	env := `{
  "name": "function-demo",
  "version": "v1",
  "runtime": "Async",
  "requestID": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "port": "50003",
  "inputs": {
    "sub": {
      "uri": "my_topic",
      "componentName": "msg",
      "componentType": "pubsub.kafka"
    }
  }
}`

	sub := &common.Subscription{
		PubsubName: "msg",
		Topic:      "my_topic",
	}
	ctx := context.Background()
	fwk, err := createFramework(env)
	if err != nil {
		t.Fatalf("failed to create framework: %v", err)
	}

	fwk.RegisterPlugins(nil)

	if err := fwk.Register(ctx, fakeBindingsFunction); err != nil {
		t.Fatalf("failed to register CloudEvents function: %v", err)
	}

	if fwk.GetRuntime() == nil {
		t.Fatal("failed to create runtime")
	}

	server := fwk.GetRuntime().GetHandler()
	if server == nil {
		t.Fatal("server is nil")
	}
	s := server.(*async.FakeServer)
	startTestServer(s)

	t.Run("topic event without request", func(t *testing.T) {
		_, err := s.OnTopicEvent(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("topic event for wrong topic", func(t *testing.T) {
		in := &runtime.TopicEventRequest{
			Topic: "invalid",
		}
		_, err := s.OnTopicEvent(ctx, in)
		assert.Error(t, err)
	})

	t.Run("topic event for valid topic", func(t *testing.T) {
		in := &runtime.TopicEventRequest{
			Id:              "a123",
			Source:          "test",
			Type:            "test",
			SpecVersion:     "v1.0",
			DataContentType: "text/plain",
			Data:            []byte("test"),
			Topic:           sub.Topic,
			PubsubName:      sub.PubsubName,
		}
		_, err := s.OnTopicEvent(ctx, in)
		assert.NoError(t, err)
	})

	stopTestServer(t, s)
}

func createFramework(env string) (Framework, error) {
	os.Setenv(ofctx.ModeEnvName, ofctx.SelfHostMode)
	os.Setenv(ofctx.TestModeEnvName, ofctx.TestModeOn)
	os.Setenv(ofctx.FunctionContextEnvName, env)
	fwk, err := NewFramework()
	if err != nil {
		return nil, err
	} else {
		return fwk, nil
	}
}

func startTestServer(server common.Service) {
	go func() {
		if err := server.Start(); err != nil && err.Error() != "closed" {
			panic(err)
		}
	}()
}

func stopTestServer(t *testing.T, server common.Service) {
	assert.NotNil(t, server)
	err := server.Stop()
	assert.Nilf(t, err, "error stopping server")
}
