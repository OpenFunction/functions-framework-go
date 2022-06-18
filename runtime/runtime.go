package runtime

import (
	"context"
	"io/ioutil"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
    cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/internal/functions"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

type Interface interface {
	Start(ctx context.Context) error
	RegisterHTTPFunction(
		ctx ofctx.RuntimeContext,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		rf *functions.RegisteredFunction,
	) error
	RegisterOpenFunction(
		ctx ofctx.RuntimeContext,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		rf *functions.RegisteredFunction,
	) error
	RegisterCloudEventFunction(
		ctx context.Context,
		funcContex ofctx.RuntimeContext,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		rf *functions.RegisteredFunction,
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

		// get the sync request
		sr := rm.FuncContext.GetSyncRequest()

		// wrap the response writer
		rww := ofctx.NewResponseWriterWrapper(sr.ResponseWriter, 200)

		function(rww, sr.Request)
		rm.FuncContext.WithOut(rm.FuncOut.WithCode(rww.Status()))

	} else if function, ok := fn.(func(ofctx.Context, []byte) (ofctx.Out, error)); ok {
		if rm.FuncContext.GetBindingEvent() != nil || rm.FuncContext.GetTopicEvent() != nil {
			// get the user data from inner event
			userData := rm.FuncContext.GetInnerEvent().GetUserData()

			// pass user data to user function
			out, err := function(functionContext, userData)

			rm.FuncContext.WithOut(out.GetOut())
			rm.FuncContext.WithError(err)

		} else if rm.FuncContext.GetSyncRequest().Request != nil {
			var body []byte
			// if it is a cloud event, we extra the cloudevent data as user data, and pass the raw cloud event in ctx
			// if it is a http request, we pass the request body as user data and create a dummy cloud event with user data
			r := rm.FuncContext.GetSyncRequest().Request
			msg := cehttp.NewMessageFromHttpRequest(r)
			event, err := binding.ToEvent(r.Context(), msg)
			if err == nil { // if it is a cloud event
				body = event.Data()
				rm.FuncContext.SetEvent("", event)
			} else { // not a cloud event
				body, _ = ioutil.ReadAll(r.Body)
				ce := cloudevents.NewEvent()
				_ = ce.SetData(cloudevents.ApplicationJSON, ofctx.ConvertUserDataToBytes(body))
				// have to reset the cloudevent here other wise a http call can get the cloud event of the last cloud event call from ctx
				rm.FuncContext.SetEvent("", &ce)
			}
			out, err := function(functionContext, body)
			rm.FuncContext.WithOut(out.GetOut())
			rm.FuncContext.WithError(err)
			// overwrite the result
			rm.FuncOut = rm.FuncContext.GetOut()
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
