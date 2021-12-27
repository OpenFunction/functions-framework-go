package functionframeworks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	daprClient "github.com/dapr/go-sdk/client"
	dapr "github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/grpc"

	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
)

const (
	functionStatusHeader = "X-Status"
	crashStatus          = "crash"
	errorStatus          = "error"
)

var (
	handler                  = http.DefaultServeMux
	openFuncAsyncServHandler dapr.Service
	mu                       sync.RWMutex
)

func RegisterHTTPFunction(ctx context.Context, fn func(http.ResponseWriter, *http.Request)) error {
	return registerHTTPFunction("/", fn, handler)
}

func RegisterOpenFunction(ctx context.Context, fn func(*ofctx.OpenFunctionContext, []byte) ofctx.RetValue) error {
	return func(f func(*ofctx.OpenFunctionContext, []byte) ofctx.RetValue) error {
		fctx, err := ofctx.GetOpenFunctionContext()
		if err != nil {
			return err
		}

		if fctx.Runtime == ofctx.OpenFuncAsync {
			openFuncAsyncServHandler, err = daprd.NewService(fmt.Sprintf(":%s", fctx.Port))
			if err != nil {
				return err
			}
		} else {
			return errors.New("cannot use non-OpenFuncAsync runtime for OpenFunction registration")
		}

		var funcErr error

		if fctx.DaprClient == nil {
			fctx.DaprClient = getDaprClient(fctx.ClientPort)
		}

		// Serving function with inputs
		if !fctx.InputsIsEmpty() {
			for name, input := range fctx.Inputs {
				switch input.Type {
				case ofctx.OpenFuncBinding:
					input.Uri = input.Component
					funcErr = openFuncAsyncServHandler.AddBindingInvocationHandler(input.Uri, func(c context.Context, in *dapr.BindingEvent) (out []byte, err error) {
						currentContext := fctx
						currentContext.Event.InputName = name
						currentContext.Event.BindingEvent = in
						ret := f(currentContext, in.Data)
						switch ret.Code {
						case ofctx.Success:
							return ret.Data, nil
						case ofctx.InternalError:
							return nil, errors.New(fmt.Sprint(ret.Error))
						default:
							return nil, nil
						}
					})
				case ofctx.OpenFuncTopic:
					sub := &dapr.Subscription{
						PubsubName: input.Component,
						Topic:      input.Uri,
					}
					funcErr = openFuncAsyncServHandler.AddTopicEventHandler(sub, func(c context.Context, e *dapr.TopicEvent) (retry bool, err error) {
						currentContext := fctx
						currentContext.Event.InputName = name
						currentContext.Event.TopicEvent = e
						ret := f(currentContext, e.Data.([]byte))
						switch ret.Code {
						case ofctx.Success:
							return false, nil
						case ofctx.InternalError:
							err = errors.New(fmt.Sprint(ret.Error))
							if retry, ok := ret.Metadata["retry"]; ok {
								if strings.EqualFold(retry, "true") {
									return true, err
								} else if strings.EqualFold(retry, "false") {
									return false, err
								} else {
									return false, err
								}
							}
							return false, err
						default:
							return false, nil
						}
					})
				default:
					return fmt.Errorf("invalid input type: %s", input.Type)
				}
				if funcErr != nil {
					destroyDaprClient(fctx.DaprClient)
					return err
				}
			}
			// Serving function without inputs
		} else {
			ret := fn(fctx, []byte{})
			switch ret.Code {
			case ofctx.Success:
				return nil
			case ofctx.InternalError:
				err = errors.New(fmt.Sprint(ret.Error))
				return err
			default:
				return nil
			}
		}
		return nil
	}(fn)
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
			port = ofctx.DefaultKnaitvePort
		}
		err = startKnative(port)
	} else {
		if ctx.Runtime == ofctx.OpenFuncAsync {
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

func getDaprClient(port string) daprClient.Client {
	mu.Lock()
	defer mu.Unlock()
	c, e := daprClient.NewClientWithPort(port)
	if e != nil {
		panic(e)
	}
	return c
}

func destroyDaprClient(client daprClient.Client) {
	mu.Lock()
	defer mu.Unlock()
	client.Close()
	client = nil
}
