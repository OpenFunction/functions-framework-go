package framework

import (
	"context"
	"errors"
	"net/http"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/internal/functions"
	"github.com/OpenFunction/functions-framework-go/internal/registry"
	"github.com/OpenFunction/functions-framework-go/plugin"
	plgExample "github.com/OpenFunction/functions-framework-go/plugin/plugin-example"
	"github.com/OpenFunction/functions-framework-go/plugin/skywalking"
	"github.com/OpenFunction/functions-framework-go/runtime"
	"github.com/OpenFunction/functions-framework-go/runtime/async"
	"github.com/OpenFunction/functions-framework-go/runtime/knative"
)

type functionsFrameworkImpl struct {
	funcContext ofctx.RuntimeContext
	prePlugins  []plugin.Plugin
	postPlugins []plugin.Plugin
	pluginMap   map[string]plugin.Plugin
	runtime     runtime.Interface
	registry    *registry.Registry
}

// Framework is the interface for the function conversion.
type Framework interface {
	Register(ctx context.Context, fn interface{}) error
	RegisterPlugins(customPlugins map[string]plugin.Plugin)
	Start(ctx context.Context) error
	GetRuntime() runtime.Interface
}

func NewFramework() (*functionsFrameworkImpl, error) {
	fwk := &functionsFrameworkImpl{}

	// Set the function registry
	fwk.registry = registry.Default()

	// Parse OpenFunction FunctionContext
	if ctx, err := ofctx.GetRuntimeContext(); err != nil {
		klog.Errorf("failed to parse OpenFunction FunctionContext: %v\n", err)
		return nil, err
	} else {
		fwk.funcContext = ctx
	}

	// Scan the local directory and register the plugins if exist
	// Register the framework default plugins under `plugin` directory
	fwk.pluginMap = map[string]plugin.Plugin{}

	// Create runtime
	if err := createRuntime(fwk); err != nil {
		klog.Errorf("failed to create runtime: %v\n", err)
		return nil, err
	}

	return fwk, nil
}

func (fwk *functionsFrameworkImpl) Register(ctx context.Context, fn interface{}) error {
	if fnHTTP, ok := fn.(func(http.ResponseWriter, *http.Request)); ok {
		rf, err := functions.New(functions.WithFunctionName(fwk.funcContext.GetName()), functions.WithHTTP(fnHTTP))
		if err != nil {
			klog.Errorf("failed to register function: %v", err)
		}
		if err := fwk.runtime.RegisterHTTPFunction(fwk.funcContext, fwk.prePlugins, fwk.postPlugins, rf); err != nil {
			klog.Errorf("failed to register function: %v", err)
			return err
		}
	} else if fnOpenFunction, ok := fn.(func(ofctx.Context, []byte) (ofctx.Out, error)); ok {
		rf, err := functions.New(functions.WithFunctionName(fwk.funcContext.GetName()), functions.WithOpenFunction(fnOpenFunction))
		if err != nil {
			klog.Errorf("failed to register function: %v", err)
		}
		if err := fwk.runtime.RegisterOpenFunction(fwk.funcContext, fwk.prePlugins, fwk.postPlugins, rf); err != nil {
			klog.Errorf("failed to register function: %v", err)
			return err
		}
	} else if fnCloudEvent, ok := fn.(func(context.Context, cloudevents.Event) error); ok {
		rf, err := functions.New(functions.WithFunctionName(fwk.funcContext.GetName()), functions.WithCloudEvent(fnCloudEvent))
		if err != nil {
			klog.Errorf("failed to register function: %v", err)
		}
		if err := fwk.runtime.RegisterCloudEventFunction(ctx, fwk.funcContext, fwk.prePlugins, fwk.postPlugins, rf); err != nil {
			klog.Errorf("failed to register function: %v", err)
			return err
		}
	} else {
		err := errors.New("unrecognized function")
		klog.Errorf("failed to register function: %v", err)
		return err
	}
	return nil
}

func (fwk *functionsFrameworkImpl) Start(ctx context.Context) error {

	target := os.Getenv("FUNCTION_TARGET")

	// if FUNCTION_TARGET is provided
	if len(target) > 0 {
		if fn, ok := fwk.registry.GetRegisteredFunction(target); ok {
			klog.Infof("registering function: %s on path: %s", target, fn.GetPath())
			switch fn.GetFunctionType() {
			case functions.HTTPType:
				fwk.Register(ctx, fn.GetHTTPFunction())
			case functions.CloudEventType:
				fwk.Register(ctx, fn.GetCloudEventFunction())
			case functions.OpenFunctionType:
				fwk.Register(ctx, fn.GetOpenFunctionFunction())
			}
		} else {
			klog.Errorf("function not found: %s", target)
		}
	} else {
		// if FUNCTION_TARGET is not provided but user uses declarative function, by default all registered functions will be deployed.
		funcNames := fwk.registry.GetFunctionNames()
		if len(funcNames) > 1 && fwk.funcContext.GetRuntime() == ofctx.Async {
			return errors.New("only one function is allowed in async runtime")
		} else if len(funcNames) > 0 {
			klog.Info("no 'FUNCTION_TARGET' is provided, register all the functions in the registry")
			for _, name := range funcNames {
				if rf, ok := fwk.registry.GetRegisteredFunction(name); ok {
					klog.Infof("registering function: %s on path: %s", rf.GetName(), rf.GetPath())
					switch rf.GetFunctionType() {
					case functions.HTTPType:
						if err := fwk.runtime.RegisterHTTPFunction(fwk.funcContext, fwk.prePlugins, fwk.postPlugins, rf); err != nil {
							klog.Errorf("failed to register function: %v", err)
							return err
						}
					case functions.CloudEventType:
						if err := fwk.runtime.RegisterCloudEventFunction(ctx, fwk.funcContext, fwk.prePlugins, fwk.postPlugins, rf); err != nil {
							klog.Errorf("failed to register function: %v", err)
							return err
						}
					case functions.OpenFunctionType:
						if err := fwk.runtime.RegisterOpenFunction(fwk.funcContext, fwk.prePlugins, fwk.postPlugins, rf); err != nil {
							klog.Errorf("failed to register function: %v", err)
							return err
						}
					}
				}
			}
		}
	}

	err := fwk.runtime.Start(ctx)
	if err != nil {
		klog.Error("failed to start runtime service")
		return err
	}
	return nil
}

func (fwk *functionsFrameworkImpl) RegisterPlugins(customPlugins map[string]plugin.Plugin) {
	// Register default plugins
	fwk.pluginMap = map[string]plugin.Plugin{
		plgExample.Name: plgExample.New(),
		skywalking.Name: skywalking.New(),
	}

	// Register custom plugins
	if customPlugins != nil {
		for name, plg := range customPlugins {
			if _, ok := fwk.pluginMap[name]; !ok {
				fwk.pluginMap[name] = plg
			} else {
				// Skip the registration of plugin with name that already exist
				continue
			}
		}
	}

	klog.Infoln("Plugins for pre-hook stage:")
	for _, plgName := range fwk.funcContext.GetPrePlugins() {
		if plg, ok := fwk.pluginMap[plgName]; ok {
			klog.Infof("- %s", plg.Name())
			fwk.prePlugins = append(fwk.prePlugins, plg)
		}
	}

	klog.Infoln("Plugins for post-hook stage:")
	for _, plgName := range fwk.funcContext.GetPostPlugins() {
		if plg, ok := fwk.pluginMap[plgName]; ok {
			klog.Infof("- %s", plg.Name())
			fwk.postPlugins = append(fwk.postPlugins, plg)
		}
	}
}

func (fwk *functionsFrameworkImpl) GetRuntime() runtime.Interface {
	return fwk.runtime
}

func createRuntime(fwk *functionsFrameworkImpl) error {
	var err error

	rt := fwk.funcContext.GetRuntime()
	port := fwk.funcContext.GetPort()
	pattern := fwk.funcContext.GetHttpPattern()
	switch rt {
	case ofctx.Knative:
		fwk.runtime = knative.NewKnativeRuntime(port, pattern)
		return nil
	case ofctx.Async:
		fwk.runtime, err = async.NewAsyncRuntime(port, pattern)
		if err != nil {
			return err
		}
	}

	if fwk.runtime == nil {
		errMsg := "runtime is nil"
		klog.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}
