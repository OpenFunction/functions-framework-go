package knative

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
)

const (
	functionStatusHeader = "X-OpenFunction-Status"
	crashStatus          = "crash"
	errorStatus          = "error"
	successStatus        = "success"
	defaultPattern       = "/"
)

type Runtime struct {
	port    string
	handler *http.ServeMux
	pattern string
}

func NewKnativeRuntime(port string, pattern string) *Runtime {
	if pattern == "" {
		pattern = defaultPattern
	}
	return &Runtime{
		port:    port,
		handler: http.DefaultServeMux,
		pattern: pattern,
	}
}

func (r *Runtime) Start(ctx context.Context) error {
	klog.Infof("Knative Function serving http: listening on port %s", r.port)
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", r.port), r.handler))
	return nil
}

func (r *Runtime) RegisterOpenFunction(
	ctx ofctx.Context,
	processPreHooksFunc func() error,
	processPostHooksFunc func() error,
	fn func(ofctx.Context, []byte) (ofctx.Out, error),
) error {
	// Register the synchronous function (based on Knaitve runtime)
	return func(f func(ofctx.Context, []byte) (ofctx.Out, error)) error {
		r.handler.HandleFunc(r.pattern, func(w http.ResponseWriter, r *http.Request) {
			ctx.SyncRequestMeta.ResponseWriter = w
			ctx.SyncRequestMeta.Request = r
			defer RecoverPanicHTTP(w, "Function panic")

			if err := processPreHooksFunc(); err != nil {
				// Just logging errors
			}

			ctx.Out, ctx.Error = f(ctx, convertRequestBodyToByte(r))

			if err := processPostHooksFunc; err != nil {
				// Just logging errors
			}

			switch ctx.Out.Code {
			case ofctx.Success:
				w.Header().Set(functionStatusHeader, successStatus)
				return
			case ofctx.InternalError:
				w.Header().Set(functionStatusHeader, errorStatus)
				w.WriteHeader(int(ctx.Out.Code))
				return
			default:
				return
			}
		})
		return nil
	}(fn)
}

func (r *Runtime) RegisterHTTPFunction(
	ctx ofctx.Context,
	processPreHooksFunc func() error,
	processPostHooksFunc func() error,
	fn func(http.ResponseWriter, *http.Request) error,
) error {
	r.handler.HandleFunc(r.pattern, func(w http.ResponseWriter, r *http.Request) {
		defer RecoverPanicHTTP(w, "Function panic")

		if err := processPreHooksFunc(); err != nil {
			// Just logging errors
		}

		ctx.Error = fn(w, r)

		if err := processPostHooksFunc(); err != nil {
			// Just logging errors
		}
	})
	return nil
}

func (r *Runtime) RegisterCloudEventFunction(
	ctx context.Context,
	funcContext ofctx.Context,
	processPreHooksFunc func() error,
	processPostHooksFunc func() error,
	fn func(context.Context, cloudevents.Event) error,
) error {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		klog.Errorf("failed to create protocol: %v\n", err)
		return err
	}

	handleFn, err := cloudevents.NewHTTPReceiveHandler(ctx, p, func(ctx context.Context, ce cloudevents.Event) error {
		if err := processPreHooksFunc(); err != nil {
			// Just logging errors
		}

		funcContext.Error = fn(ctx, ce)

		if err := processPostHooksFunc(); err != nil {
			// Just logging errors
		}

		return funcContext.Error
	})

	if err != nil {
		klog.Errorf("failed to create handler: %v\n", err)
		return err
	}

	r.handler.Handle("/", handleFn)
	return nil
}

func RecoverPanicHTTP(w http.ResponseWriter, msg string) {
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

func convertRequestBodyToByte(r *http.Request) []byte {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}
	return body
}
