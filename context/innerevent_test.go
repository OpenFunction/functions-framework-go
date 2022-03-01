package context

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/dapr/go-sdk/service/common"
)

var funcCtx = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "12345",
  "inputs": {
    "cron": {
      "uri": "cron_input",
      "componentName": "cron_input",
      "componentType": "bindings.cron"
    },
    "eventbus": {
      "uri": "default",
      "componentName": "nats_eventbus",
      "componentType": "pubsub.natsstreaming"
    }
  },
  "outputs": {
    "echo": {
      "uri": "echo",
      "operation": "create",
      "componentName": "echo",
      "componentType": "bindings.kafka",
      "metadata": {
        "path": "echo",
        "Content-Type": "application/json; charset=utf-8"
      }
    },
    "target": {
      "uri": "sample",
      "operation": "create",
      "componentName": "kafka-server",
      "componentType": "pubsub.kafka"
    },
    "target2": {
      "uri": "cron_output",
      "componentName": "cron_output",
      "componentType": "bindings.cron"
    }
  }
}`

// TestInnerEvent tests and verifies the function that parses the function FunctionContext
func TestInnerEvent(t *testing.T) {

	var ctx RuntimeContext

	if err := os.Setenv(PodNameEnvName, "test-pod"); err != nil {
		t.Fatal("Error set pod name env")
	}

	if err := os.Setenv(PodNamespaceEnvName, "test"); err != nil {
		t.Fatal("Error set pod namespace env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtx); err != nil {
		t.Fatal("Error set function context env")

	}

	ctx, err := GetRuntimeContext()
	if err != nil {
		t.Fatalf("Error parse function context: %v", err)
	}

	data := map[string]string{
		"foo1": "bar1",
	}

	byteData, _ := json.Marshal(data)

	ie := NewInnerEvent(ctx)
	ie.SetMetadata("k2", "v2")
	ie.SetUserData(data)

	// test bindingEvent
	be1 := &common.BindingEvent{
		Data: byteData,
	}
	eventTest(t, ctx, be1, byteData)

	be2 := &common.BindingEvent{
		Data: ie.GetCloudEventJSON(),
	}
	eventTest(t, ctx, be2, byteData)

	// test topicEvent
	te1 := &common.TopicEvent{
		Data: byteData,
	}
	eventTest(t, ctx, te1, byteData)

	te2 := &common.TopicEvent{
		Data: ie.GetCloudEventJSON(),
	}
	eventTest(t, ctx, te2, byteData)

	// test if we need wrap user data
	outputs := ctx.GetOutputs()
	if outputs == nil {
		t.Fatal("Error get outputs in context")
	}

	if output, exist := outputs["target"]; exist && output != nil && traceable(output.ComponentType) {

	} else {
		t.Fatal("Error determining whether user data needs to be wrapped")
	}

	if output, exist := outputs["target2"]; exist && output != nil && !traceable(output.ComponentType) {

	} else {
		t.Fatal("Error determining whether user data needs to be wrapped")
	}
}

func eventTest(t *testing.T, ctx RuntimeContext, event interface{}, target []byte) {
	// receive test
	ctx.SetEvent("cron", event)
	ie := ctx.GetInnerEvent()
	if !bytes.Equal(ie.GetUserData(), target) {
		t.Fatal("Error get user data in innerEvent")
	}
	ie.SetMetadata("k1", "v1")

	// send test

	data := map[string]string{
		"foo2": "bar2",
	}

	ie2 := NewInnerEvent(ctx)
	ie2.MergeMetadata(ctx.GetInnerEvent())
	ie2.SetUserData(data)
	if ie2.GetMetadata() != nil {
		if v, exist := ie2.GetMetadata()["k1"]; exist && v == "v1" {

		} else {
			t.Fatal("Error set inner event metadata")
		}
	} else {
		t.Fatal("Error set inner event metadata")
	}

	udInEvent := map[string]string{}
	if ie2.GetUserData() != nil {
		if err := json.Unmarshal(ie2.GetUserData(), &udInEvent); err != nil {
			t.Fatal("Error unmarshal user data in inner event")
		}
		if v, exist := udInEvent["foo2"]; exist && v == "bar2" {

		} else {
			t.Fatal("Error set inner event userdata")
		}
	} else {
		t.Fatal("Error set inner event userdata")
	}

	// cloudevent test
	ce := ie2.GetCloudEvent()
	if ce.Data() == nil {
		t.Fatal("Error set inner event cloudevent")
	}

	ieData := &innerEventData{}
	if err := ce.DataAs(ieData); err != nil {
		t.Fatalf("Error save inner event data: %v", err)
	}

	ud := map[string]string{}
	if err := json.Unmarshal(ieData.UserData, &ud); err != nil {
		t.Fatal("Error unmarshal user data in inner event")
	}

	if v, exist := ud["foo2"]; exist && v == "bar2" {

	} else {
		t.Fatal("Error save inner event userdata")
	}
}
