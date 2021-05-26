package functionframeworks

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	functionStatusHeader = "X-Status"
	crashStatus          = "crash"
	errorStatus          = "error"
)

var (
	handler = http.DefaultServeMux
)

func RegisterHTTPFunction(ctx context.Context, fn func(http.ResponseWriter, *http.Request)) error {
	return registerHTTPFunction("/", fn, handler)
}

func RegisterOpenFunction(fn func(*OpenFunctionContext, *http.Request) int) error {
	return registerOpenFunction(fn, handler)
}

func RegisterCloudEventFunction(ctx context.Context, fn func(context.Context, cloudevents.Event) error) error {
	return registerCloudEventFunction(ctx, fn, handler)
}

func registerHTTPFunction(path string, fn func(http.ResponseWriter, *http.Request), h *http.ServeMux) error {
	h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		defer recoverPanicHTTP(w, "Function panic")
		fn(w, r)
	})
	return nil
}

func registerOpenFunction(fn func(*OpenFunctionContext, *http.Request) int, h *http.ServeMux) error {
	ctx, err := GetOpenFunctionContext()
	if err != nil {
		return err
	}

	h.HandleFunc(ctx.Input.Url, func(w http.ResponseWriter, r *http.Request) {
		defer recoverPanicHTTP(w, "Function panic")
		code := fn(ctx, r)
		w.WriteHeader(code)
	})

	return nil
}

func registerCloudEventFunction(ctx context.Context, fn func(context.Context, cloudevents.Event) error, h *http.ServeMux) error {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		return fmt.Errorf("failed to create protocol: %v", err)
	}

	handleFn, err := cloudevents.NewHTTPReceiveHandler(ctx, p, fn)

	if err != nil {
		return fmt.Errorf("failed to create handler: %v", err)
	}

	h.Handle("/", handleFn)
	return nil
}

func Start(port string) error {
	log.Printf("Function serving: listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
	return nil
}

func recoverPanicHTTP(w http.ResponseWriter, msg string) {
	if r := recover(); r != nil {
		writeHTTPErrorResponse(w, http.StatusInternalServerError, crashStatus, fmt.Sprintf("%s: %v\n\n%s", msg, r, debug.Stack()))
	}
}

func writeHTTPErrorResponse(w http.ResponseWriter, statusCode int, status, msg string) {
	// Ensure logs end with a newline otherwise they are grouped incorrectly in SD.
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprintf(os.Stderr, msg)

	w.Header().Set(functionStatusHeader, status)
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, msg)
}
