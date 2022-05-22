// Package functions provides a way to declaratively register functions
// that can be used to handle incoming requests.
package functions

import (
	"context"
	"log"
	"net/http"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/internal/registry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// HTTP registers an HTTP function that becomes the function handler served
// at "/" when environment variable `FUNCTION_TARGET=name`
func HTTP(name string, fn func(http.ResponseWriter, *http.Request), options ...FunctionOption) {
	if err := registry.Default().RegisterHTTP(name, fn, options...); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

// CloudEvent registers a CloudEvent function that becomes the function handler
// served at "/" when environment variable `FUNCTION_TARGET=name`
func CloudEvent(name string, fn func(context.Context, cloudevents.Event) error, options ...FunctionOption) {
	if err := registry.Default().RegisterCloudEvent(name, fn, options...); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}

// OpenFunction registers a OpenFunction function that becomes the function handler
// served at "/" when environment variable `FUNCTION_TARGET=name`
func OpenFunction(name string, fn func(ofctx.Context, []byte) (ofctx.Out, error), options ...FunctionOption) {
	if err := registry.Default().RegisterOpenFunction(name, fn, options...); err != nil {
		log.Fatalf("failure to register function: %s", err)
	}
}
