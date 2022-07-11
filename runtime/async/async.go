package async

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dapr "github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/grpc"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/internal/functions"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/runtime"
)

const (
	defaultPattern = "/"
)

type Runtime struct {
	port       string
	pattern    string
	handler    dapr.Service
	grpcHander *FakeServer
}

func NewAsyncRuntime(port string, pattern string) (*Runtime, error) {
	if pattern == "" {
		pattern = defaultPattern
	}
	if testMode := os.Getenv(ofctx.TestModeEnvName); testMode == ofctx.TestModeOn {
		handler, grpcHandler, err := NewFakeService(fmt.Sprintf(":%s", port))
		if err != nil {
			klog.Errorf("failed to create dapr grpc service: %v\n", err)
			return nil, err
		}
		return &Runtime{
			port:       port,
			pattern:    pattern,
			handler:    handler,
			grpcHander: grpcHandler,
		}, nil
	}
	handler, err := daprd.NewService(fmt.Sprintf(":%s", port))
	if err != nil {
		klog.Errorf("failed to create dapr grpc service: %v\n", err)
		return nil, err
	}
	return &Runtime{
		port:       port,
		pattern:    pattern,
		handler:    handler,
		grpcHander: nil,
	}, nil
}

func (r *Runtime) Start(ctx context.Context) error {
	klog.Infof("Async Function serving grpc: listening on port %s", r.port)
	klog.Fatal(r.handler.Start())
	return nil
}

func (r *Runtime) RegisterHTTPFunction(
	ctx ofctx.RuntimeContext,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	rf *functions.RegisteredFunction,
) error {
	return errors.New("async runtime cannot register http function")
}

func (r *Runtime) RegisterCloudEventFunction(
	ctx context.Context,
	funcContext ofctx.RuntimeContext,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	rf *functions.RegisteredFunction,
) error {
	return errors.New("async runtime cannot register cloudevent function")
}

func (r *Runtime) RegisterOpenFunction(
	ctx ofctx.RuntimeContext,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	rf *functions.RegisteredFunction,
) error {
	// Register the asynchronous functions (based on the Dapr runtime)
	return func(f func(ofctx.Context, []byte) (ofctx.Out, error)) error {
		var funcErr error

		// Initialize dapr client if it is nil
		ctx.InitDaprClientIfNil()

		// Serving function with inputs
		if ctx.HasInputs() {
			for name, input := range ctx.GetInputs() {
				n := name
				switch input.GetType() {
				case ofctx.OpenFuncBinding:
					input.Uri = input.ComponentName
					funcErr = r.handler.AddBindingInvocationHandler(input.Uri, func(c context.Context, in *dapr.BindingEvent) (out []byte, err error) {
						rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
						rm.FuncContext.SetEvent(n, in)
						rm.FunctionRunWrapperWithHooks(rf.GetOpenFunctionFunction())

						switch rm.FuncOut.GetCode() {
						case ofctx.Success:
							return rm.FuncOut.GetData(), nil
						case ofctx.InternalError:
							return nil, rm.FuncContext.GetError()
						default:
							return nil, nil
						}
					})
					if funcErr == nil {
						klog.Infof("registered bindings handler: %s", input.Uri)
					}
				case ofctx.OpenFuncTopic:
					sub := &dapr.Subscription{
						PubsubName: input.ComponentName,
						Topic:      input.Uri,
					}
					funcErr = r.handler.AddTopicEventHandler(sub, func(c context.Context, e *dapr.TopicEvent) (retry bool, err error) {
						rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
						rm.FuncContext.SetEvent(name, e)
						rm.FunctionRunWrapperWithHooks(rf.GetOpenFunctionFunction())

						switch rm.FuncOut.GetCode() {
						case ofctx.Success:
							return false, nil
						case ofctx.InternalError:
							err = rm.FuncContext.GetError()
							if retry, ok := rm.FuncOut.GetMetadata()["retry"]; ok {
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
					if funcErr == nil {
						klog.Infof("registered pubsub handler: %s, topic: %s", input.ComponentName, input.Uri)
					}
				default:
					return fmt.Errorf("invalid input type: %s", input.GetType())
				}
				if funcErr != nil {
					// When the function throws an exception,
					// first call client.Close() to close the dapr client,
					// then set fwk.funcContext.daprClient to nil
					ctx.DestroyDaprClient()
					klog.Errorf("failed to add dapr service handler: %v\n", funcErr)
					return funcErr
				}
			}
			// If a function has no input, just return it.
			return nil
		}
		err := errors.New("no inputs defined for the function")
		klog.Errorf("failed to register function: %v\n", err)
		return err
	}(rf.GetOpenFunctionFunction())
}

func (r *Runtime) Name() ofctx.Runtime {
	return ofctx.Async
}

func (r *Runtime) GetHandler() interface{} {
	return r.grpcHander
}
