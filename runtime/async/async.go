package async

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	dapr "github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/grpc"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/OpenFunction/functions-framework-go/runtime"
)

type Runtime struct {
	port    string
	handler dapr.Service
}

func NewAsyncRuntime(port string) (*Runtime, error) {
	handler, err := daprd.NewService(fmt.Sprintf(":%s", port))
	if err != nil {
		klog.Errorf("failed to create dapr grpc service: %v\n", err)
		return nil, err
	}
	return &Runtime{
		port:    port,
		handler: handler,
	}, nil
}

func (r *Runtime) Start(ctx context.Context) error {
	klog.Infof("Async Function serving grpc: listening on port %s", r.port)
	klog.Fatal(r.handler.Start())
	return nil
}

func (r *Runtime) RegisterHTTPFunction(
	ctx ofctx.Context,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	fn func(http.ResponseWriter, *http.Request) error,
) error {
	return errors.New("async runtime cannot register http function")
}

func (r *Runtime) RegisterCloudEventFunction(
	ctx context.Context,
	funcContext ofctx.Context,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	fn func(context.Context, cloudevents.Event) error,
) error {
	return errors.New("async runtime cannot register cloudevent function")
}

func (r *Runtime) RegisterOpenFunction(
	ctx ofctx.Context,
	prePlugins []plugin.Plugin,
	postPlugins []plugin.Plugin,
	fn func(ofctx.Context, []byte) (ofctx.Out, error),
) error {
	// Register the asynchronous functions (based on the Dapr runtime)
	return func(f func(ofctx.Context, []byte) (ofctx.Out, error)) error {
		var funcErr error

		// Initialize dapr client if it is nil
		ofctx.InitDaprClientIfNil(&ctx)

		// Serving function with inputs
		if !ctx.InputsIsEmpty() {
			for name, input := range ctx.Inputs {
				switch input.Type {
				case ofctx.OpenFuncBinding:
					input.Uri = input.Component
					funcErr = r.handler.AddBindingInvocationHandler(input.Uri, func(c context.Context, in *dapr.BindingEvent) (out []byte, err error) {
						rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
						rm.FuncContext.EventMeta.InputName = name
						rm.FuncContext.EventMeta.BindingEvent = in

						rm.ProcessPreHooks()

						rm.FuncContext.Out, rm.FuncContext.Error = f(rm.FuncContext, in.Data)

						rm.ProcessPostHooks()

						switch rm.FuncContext.Out.Code {
						case ofctx.Success:
							return rm.FuncContext.Out.Data, nil
						case ofctx.InternalError:
							return nil, rm.FuncContext.Out.Error
						default:
							return nil, nil
						}
					})
				case ofctx.OpenFuncTopic:
					sub := &dapr.Subscription{
						PubsubName: input.Component,
						Topic:      input.Uri,
					}
					funcErr = r.handler.AddTopicEventHandler(sub, func(c context.Context, e *dapr.TopicEvent) (retry bool, err error) {
						rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
						rm.FuncContext.EventMeta.InputName = name
						rm.FuncContext.EventMeta.TopicEvent = e

						rm.ProcessPreHooks()

						rm.FuncContext.Out, rm.FuncContext.Error = f(rm.FuncContext, convertTopicEventToByte(e.Data))

						rm.ProcessPostHooks()

						switch rm.FuncContext.Out.Code {
						case ofctx.Success:
							return false, nil
						case ofctx.InternalError:
							err = rm.FuncContext.Out.Error
							if retry, ok := rm.FuncContext.Out.Metadata["retry"]; ok {
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
					// When the function throws an exception,
					// first call client.Close() to close the dapr client,
					// then set fwk.funcContext.daprClient to nil
					ofctx.DestroyDaprClient(&ctx)
					klog.Errorf("failed to add dapr service handler: %v\n", funcErr)
					return funcErr
				}
			}
			// Serving function without inputs
		} else {
			rm := runtime.NewRuntimeManager(ctx, prePlugins, postPlugins)
			rm.ProcessPreHooks()

			rm.FuncContext.Out, rm.FuncContext.Error = f(rm.FuncContext, nil)

			rm.ProcessPostHooks()

			switch rm.FuncContext.Out.Code {
			case ofctx.Success:
				return nil
			case ofctx.InternalError:
				return rm.FuncContext.Out.Error
			default:
				return nil
			}
		}
		return nil
	}(fn)
}

func convertTopicEventToByte(data interface{}) []byte {
	if d, ok := data.([]byte); ok {
		return d
	}
	if d, err := json.Marshal(data); err != nil {
		return nil
	} else {
		return d
	}
}
