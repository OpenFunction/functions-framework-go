package knative

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/internal/functions"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/runtime"
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
	pattern string
	handler *mux.Router
}

func NewKnativeRuntime(port string, pattern string) *Runtime {
	if pattern == "" {
		pattern = defaultPattern
	}
	return &Runtime{
		port:    port,
		pattern: pattern,
		handler: mux.NewRouter(),
	}
}

func (r *Runtime) Start(ctx context.Context) error {
	klog.Infof("Knative Function serving http: listening on port %s", r.port)
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", r.port), r.handler))
	return nil
}

func (r *Runtime) RegisterOpenFunction(
	ctx ofctx.RuntimeContext,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	rf *functions.RegisteredFunction,
) error {
	// Initialize dapr client if FuncContext defined inputs or outputs
	if ctx.HasInputs() || ctx.HasOutputs() {
		ctx.InitDaprClientIfNil()
	}

	// Register the synchronous function (based on Knaitve runtime)
	route := r.handler.HandleFunc(rf.GetPath(), func(w http.ResponseWriter, r *http.Request) {
		rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
		// save the Vars into the context
		_ctx := ofctx.CtxWithVars(r.Context(), ofctx.Vars(r))
		rm.FuncContext.SetNativeContext(_ctx)
		rm.FuncContext.SetSyncRequest(w, r.WithContext(_ctx))
		defer RecoverPanicHTTP(w, "Function panic")
		rm.FunctionRunWrapperWithHooks(rf.GetOpenFunctionFunction())

		switch rm.FuncOut.GetCode() {
		case ofctx.Success:
			w.Header().Set(functionStatusHeader, successStatus)
			w.WriteHeader(rm.FuncOut.GetCode())
			w.Write(rm.FuncOut.GetData())
			return
		case ofctx.InternalError:
			w.Header().Set(functionStatusHeader, errorStatus)
			w.WriteHeader(rm.FuncOut.GetCode())
			return
		default:
			return
		}
	})

	// add methods matcher if provided
	methods := rf.GetFunctionMethods()
	if len(methods) > 0 {
		route.Methods(methods...)
	}

	return nil
}

func (r *Runtime) RegisterHTTPFunction(
	ctx ofctx.RuntimeContext,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	rf *functions.RegisteredFunction,
) error {
	route := r.handler.HandleFunc(rf.GetPath(), func(w http.ResponseWriter, r *http.Request) {
		rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
		// save the Vars into the context
		_ctx := ofctx.CtxWithVars(r.Context(), ofctx.Vars(r))
		rm.FuncContext.SetNativeContext(_ctx)
		rm.FuncContext.SetSyncRequest(w, r.WithContext(_ctx))
		defer RecoverPanicHTTP(w, "Function panic")
		rm.FunctionRunWrapperWithHooks(rf.GetHTTPFunction())
	})

	// add methods matcher if any
	methods := rf.GetFunctionMethods()
	if len(methods) > 0 {
		route.Methods(methods...)
	}

	return nil
}

func (r *Runtime) RegisterCloudEventFunction(
	ctx context.Context,
	funcContext ofctx.RuntimeContext,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	rf *functions.RegisteredFunction,
) error {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		klog.Errorf("failed to create protocol: %v\n", err)
		return err
	}

	handleFn, err := cloudevents.NewHTTPReceiveHandler(ctx, p, func(ctx context.Context, ce cloudevents.Event) error {
		rm := runtime.NewRuntimeManager(funcContext, prePlugins, postPlugins)
		// save the native ctx
		rm.FuncContext.SetNativeContext(ctx)
		rm.FuncContext.SetEvent("", &ce)
		rm.FunctionRunWrapperWithHooks(rf.GetCloudEventFunction())
		return rm.FuncContext.GetError()
	})

	if err != nil {
		klog.Errorf("failed to create handler: %v\n", err)
		return err
	}

	// function to extract Vars and add into ctx
	withVars := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ofctx.CtxWithVars(r.Context(), ofctx.Vars(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	r.handler.Handle(rf.GetPath(), withVars(handleFn))
	return nil
}

func (r *Runtime) Name() ofctx.Runtime {
	return ofctx.Knative
}

func (r *Runtime) GetHandler() interface{} {
	return r.handler
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
