package main

import (
	"context"

	"github.com/OpenFunction/functions-framework-go/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"k8s.io/klog/v2"
)

func init() {
	functions.CloudEvent("HelloWorld", HelloWorld, functions.WithFunctionPath("/"))
}

func HelloWorld(ctx context.Context, ce cloudevents.Event) error {
	klog.Infof("cloudevent - Data: %s", ce.Data())
	return nil
}
