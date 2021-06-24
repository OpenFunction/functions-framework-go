package functionframeworks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	dapr "github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/grpc"
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
	handler                  = http.DefaultServeMux
	openFuncAsyncServHandler dapr.Service
)

func RegisterHTTPFunction(ctx context.Context, fn func(http.ResponseWriter, *http.Request)) error {
	return registerHTTPFunction("/", fn, handler)
}

func RegisterOpenFunction(ctx context.Context, fn func(*ofctx.OpenFunctionContext, []byte) int) error {
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

func registerOpenFunction(fn func(*ofctx.OpenFunctionContext, []byte) int, h *http.ServeMux) error {
	ctx, err := ofctx.GetOpenFunctionContext()
	if err != nil {
		return err
	}

	if ctx.Runtime == ofctx.OpenFuncAsync {
		openFuncAsyncServHandler, err = daprd.NewService(fmt.Sprintf(":%s", ctx.Port))
		if err != nil {
			return err
		}
	} else {
		return errors.New(fmt.Sprint("Cannot use non-OpenFuncAsync runtime for function registration."))
	}

	// Serving function with inputs
	if !ctx.InputIsEmpty() {
		inType := ctx.Input.Params["type"]
		switch ofctx.ResourceType(inType) {
		case ofctx.OpenFuncBinding:
			if ctx.Input.Uri == "" {
				ctx.Input.Uri = ctx.Input.Name
			}
			err = openFuncAsyncServHandler.AddBindingInvocationHandler(ctx.Input.Uri, func(c context.Context, in *dapr.BindingEvent) (out []byte, err error) {
				code := fn(ctx, in.Data)
				if code == 200 {
					return nil, nil
				} else {
					return nil, errors.New("error")
				}
			})
		case ofctx.OpenFuncTopic:
			sub := &dapr.Subscription{
				PubsubName: ctx.Input.Name,
				Topic:      ctx.Input.Uri,
			}
			err = openFuncAsyncServHandler.AddTopicEventHandler(sub, func(c context.Context, e *dapr.TopicEvent) (retry bool, err error) {
				in, err := json.Marshal(e.Data)
				if err != nil {
					return true, err
				}
				code := fn(ctx, in)
				if code == 200 {
					return false, nil
				} else {
					return true, errors.New("error")
				}
			})
		case ofctx.OpenFuncService:
			if ctx.Input.Uri == "" {
				ctx.Input.Uri = ctx.Input.Name
			}
			err = openFuncAsyncServHandler.AddServiceInvocationHandler(ctx.Input.Uri, func(c context.Context, in *dapr.InvocationEvent) (out *dapr.Content, err error) {
				code := fn(ctx, in.Data)
				if code == 200 {
					return nil, nil
				} else {
					return nil, errors.New("error")
				}
			})
		default:
			return fmt.Errorf("invalid input type: %s", inType)
		}
		if err != nil {
			return err
		}
		// Serving function without inputs
	} else {
		code := fn(ctx, []byte{})
		if code == 200 {
			return nil
		} else {
			return errors.New("error")
		}
	}
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

func Start() error {
	ctx, err := ofctx.GetOpenFunctionContext()
	if err != nil {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		err = startKnative(port)
	} else {
		if ctx.Runtime == ofctx.OpenFuncAsync {
			port := "50001"
			if ctx.Port == "" {
				ctx.Port = port
			}
			err = startOpenFuncAsync(ctx)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func startKnative(port string) error {
	log.Printf("Knative Function serving http: listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
	return nil
}

func startOpenFuncAsync(ctx *ofctx.OpenFunctionContext) error {
	log.Printf("OpenFuncAsync Function serving grpc: listening on port %s", ctx.Port)
	log.Fatal(openFuncAsyncServHandler.Start())
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
