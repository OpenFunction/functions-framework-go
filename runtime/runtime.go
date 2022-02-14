package runtime

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

type Interface interface {
	Start(ctx context.Context) error
	RegisterHTTPFunction(
		ctx ofctx.RuntimeContext,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		fn func(http.ResponseWriter, *http.Request),
	) error
	RegisterOpenFunction(
		ctx ofctx.RuntimeContext,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		fn func(ofctx.Context, []byte) (ofctx.Out, error),
	) error
	RegisterCloudEventFunction(
		ctx context.Context,
		funcContex ofctx.RuntimeContext,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		fn func(context.Context, cloudevents.Event) error,
	) error
	Name() ofctx.Runtime
	GetHandler() interface{}
}

type RuntimeManager struct {
	FuncContext ofctx.RuntimeContext
	FuncOut     ofctx.Out
	prePlugins  []plugin.Plugin
	postPlugins []plugin.Plugin
	pluginState map[string]plugin.Plugin
}

func NewRuntimeManager(funcContext ofctx.RuntimeContext, prePlugin []plugin.Plugin, postPlugin []plugin.Plugin) *RuntimeManager {
	ctx := funcContext
	rm := &RuntimeManager{
		FuncContext: ctx,
		prePlugins:  prePlugin,
		postPlugins: postPlugin,
	}
	if ctx.GetOut() != nil {
		rm.FuncOut = ctx.GetOut()
	} else {
		rm.FuncOut = ofctx.NewFunctionOut()
	}
	rm.init()
	return rm
}

func (rm *RuntimeManager) init() {
	rm.FuncContext.SetNativeContext(context.Background())
	rm.pluginState = map[string]plugin.Plugin{}

	var newPrePlugins []plugin.Plugin
	for _, plg := range rm.prePlugins {
		if existPlg, ok := rm.pluginState[plg.Name()]; !ok {
			p := plg.Init()
			rm.pluginState[plg.Name()] = p
			newPrePlugins = append(newPrePlugins, p)
		} else {
			newPrePlugins = append(newPrePlugins, existPlg)
		}
	}
	rm.prePlugins = newPrePlugins

	var newPostPlugins []plugin.Plugin
	for _, plg := range rm.postPlugins {
		if existPlg, ok := rm.pluginState[plg.Name()]; !ok {
			p := plg.Init()
			rm.pluginState[plg.Name()] = p
			newPostPlugins = append(newPostPlugins, p)
		} else {
			newPostPlugins = append(newPostPlugins, existPlg)
		}
	}
	rm.postPlugins = newPostPlugins
}

func (rm *RuntimeManager) ProcessPreHooks() {
	for _, plg := range rm.prePlugins {
		if err := plg.ExecPreHook(rm.FuncContext, rm.pluginState); err != nil {
			klog.Warningf("plugin %s failed in pre phase: %s", plg.Name(), err.Error())
		}
	}
}

func (rm *RuntimeManager) ProcessPostHooks() {
	for _, plg := range rm.postPlugins {
		if err := plg.ExecPostHook(rm.FuncContext, rm.pluginState); err != nil {
			klog.Warningf("plugin %s failed in post phase: %s", plg.Name(), err.Error())
		}
	}
}

func (rm *RuntimeManager) FunctionRunWrapperWithHooks(fn interface{}) {
	functionContext := rm.FuncContext.GetContext()

	rm.ProcessPreHooks()

	if function, ok := fn.(func(http.ResponseWriter, *http.Request)); ok {
		srMeta := rm.FuncContext.GetSyncRequest()
		rww := ofctx.NewResponseWriterWrapper(srMeta.ResponseWriter, 200)
		function(rww, srMeta.Request)
		rm.FuncContext.WithOut(rm.FuncOut.WithCode(rww.Status()))
	} else if function, ok := fn.(func(ofctx.Context, []byte) (ofctx.Out, error)); ok {
		if rm.FuncContext.GetBindingEvent() != nil {
			out, err := function(functionContext, rm.FuncContext.GetBindingEvent().Data)
			rm.FuncContext.WithOut(out.GetOut())
			rm.FuncContext.WithError(err)
		} else if rm.FuncContext.GetTopicEvent() != nil {
			out, err := function(functionContext, convertTopicEventToByte(rm.FuncContext.GetTopicEvent().Data))
			rm.FuncContext.WithOut(out.GetOut())
			rm.FuncContext.WithError(err)
		} else if rm.FuncContext.GetSyncRequest().Request != nil {
			body, _ := ioutil.ReadAll(rm.FuncContext.GetSyncRequest().Request.Body)
			out, err := function(functionContext, body)
			rm.FuncContext.WithOut(out.GetOut())
			rm.FuncContext.WithError(err)
		}
	} else if function, ok := fn.(func(context.Context, cloudevents.Event) error); ok {
		ce := cloudevents.Event{}
		if rm.FuncContext.GetCloudEvent() != nil {
			ce = *rm.FuncContext.GetCloudEvent()
		}
		rm.FuncContext.WithError(function(rm.FuncContext.GetNativeContext(), ce))
	}

	rm.ProcessPostHooks()
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
