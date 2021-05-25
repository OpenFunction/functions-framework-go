package main

import (
	"context"
	"github.com/OpenFunction/functions-framework-go/functionframeworks"
	"github.com/OpenFunction/functions-framework-go/testdata/demo"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"log"
	"net/http"
	"os"
)

func register(fn interface{}) error {
	ctx := context.Background()
	if fnHTTP, ok := fn.(func(http.ResponseWriter, *http.Request)); ok {
		if err := functionframeworks.RegisterHTTPFunction(ctx, fnHTTP); err != nil {
			return err
		}
	} else if fnCloudEvent, ok := fn.(func(context.Context, cloudevents.Event)); ok {
		if err := functionframeworks.RegisterCloudEventFunction(ctx, fnCloudEvent); err != nil {
			return err
		}
	} else if fnOpenFunction, ok := fn.(func(*functionframeworks.OpenFunctionContext, *http.Request) int); ok {
		if err := functionframeworks.RegisterOpenFunction(fnOpenFunction); err != nil {
			return err
		}
	}
	return nil
}

func mainInput() {

	if err := register(demo.InputOnlyFunction); err != nil {
		log.Fatalf("Failed to register: %v\n", err)
	}

	port := "3000"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := functionframeworks.Start(port); err != nil {
		log.Fatalf("Failed to start: %v\n", err)
	}
}

func mainBindings() {

	if err := register(demo.BindingsFunction); err != nil {
		log.Fatalf("Failed to register: %v\n", err)
	}

	port := "3000"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := functionframeworks.Start(port); err != nil {
		log.Fatalf("Failed to start: %v\n", err)
	}
}
