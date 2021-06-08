package functionframeworks

import (
	"context"
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
	handler     = http.DefaultServeMux
	grpcHandler dapr.Service
)

func RegisterHTTPFunction(ctx context.Context, fn func(http.ResponseWriter, *http.Request)) error {
	return registerHTTPFunction("/", fn, handler)
}

func RegisterOpenFunction(ctx context.Context, fn func(*ofctx.OpenFunctionContext, interface{}) int) error {
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

func registerOpenFunction(fn func(*ofctx.OpenFunctionContext, interface{}) int, h *http.ServeMux) error {
	ctx, err := ofctx.GetOpenFunctionContext()
	if err != nil {
		return err
	}

	if *ctx.Input.Enabled {
		if ctx.Protocol == ofctx.HTTP {
			h.HandleFunc(ctx.Input.Pattern, func(w http.ResponseWriter, r *http.Request) {
				defer recoverPanicHTTP(w, "Function panic")
				code := fn(ctx, r)
				w.WriteHeader(code)
			})
		}

		if ctx.Protocol == ofctx.GRPC && ctx.Runtime == ofctx.Dapr {
			grpcHandler, err = daprd.NewService(fmt.Sprintf(":%s", ctx.Port))
			if err != nil {
				return err
			}

			switch ctx.Input.InType {
			case ofctx.DaprBinding:
				err = grpcHandler.AddBindingInvocationHandler(ctx.Input.Pattern, func(c context.Context, in *dapr.BindingEvent) (out []byte, err error) {
					code := fn(ctx, in)
					if code == 200 {
						return nil, nil
					} else {
						return nil, errors.New("error")

					}
				})
			case ofctx.DaprTopic:
				sub := &dapr.Subscription{
					PubsubName: ctx.Input.Name,
					Topic:      ctx.Input.Pattern,
				}
				err = grpcHandler.AddTopicEventHandler(sub, func(c context.Context, e *dapr.TopicEvent) (retry bool, err error) {
					code := fn(ctx, e)
					if code == 200 {
						return false, nil
					} else {
						return true, errors.New("error")
					}
				})
			case ofctx.DaprService:
				err = grpcHandler.AddServiceInvocationHandler(ctx.Input.Pattern, func(c context.Context, in *dapr.InvocationEvent) (out *dapr.Content, err error) {
					code := fn(ctx, in)
					if code == 200 {
						return nil, nil
					} else {
						return nil, errors.New("error")
					}
				})
			default:
				return errors.New("invalid input type")
			}
			if err != nil {
				return err
			}
		}

	} else {
		if ctx.Protocol == ofctx.HTTP {
			h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				defer recoverPanicHTTP(w, "Function panic")
				code := fn(ctx, r)
				w.WriteHeader(code)
			})
		}

		if ctx.Protocol == ofctx.GRPC {
			grpcHandler, err = daprd.NewService(fmt.Sprintf(":%s", ctx.Port))

			if err != nil {
				return err
			}
			code := fn(ctx, nil)
			if code == 200 {
				return nil
			} else {
				return errors.New("error")
			}
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
		err = startHTTP(port)
	} else {
		switch ctx.Protocol {
		case ofctx.HTTP:
			port := "8080"
			if ctx.Port != "" {
				port = ctx.Port
			}
			err = startHTTP(port)
		case ofctx.GRPC:
			port := "50001"
			if ctx.Port != "" {
				port = ctx.Port
			}
			err = startGRPC(port)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func startHTTP(port string) error {
	log.Printf("Function serving http: listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
	return nil
}

func startGRPC(port string) error {
	log.Printf("Function serving grpc: listening on port %s", port)
	log.Fatal(grpcHandler.Start())
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
