package runtime

import (
	"context"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

type Interface interface {
	Start(ctx context.Context) error
	RegisterHTTPFunction(
		ctx ofctx.Context,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		fn func(http.ResponseWriter, *http.Request) error,
	) error
	RegisterOpenFunction(
		ctx ofctx.Context,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		fn func(ofctx.Context, []byte) (ofctx.Out, error),
	) error
	RegisterCloudEventFunction(
		ctx context.Context,
		funcContex ofctx.Context,
		prePlugins []plugin.Plugin,
		postPlugins []plugin.Plugin,
		fn func(context.Context, cloudevents.Event) error,
	) error
}

type RuntimeManager struct {
	FuncContext ofctx.Context
	prePlugins  []plugin.Plugin
	postPlugins []plugin.Plugin
	pluginState map[string]plugin.Plugin
}

func NewRuntimeManager(funcContext ofctx.Context, prePlugin []plugin.Plugin, postPlugin []plugin.Plugin) *RuntimeManager {
	ctx := funcContext
	rm := &RuntimeManager{
		FuncContext: ctx,
		prePlugins:  prePlugin,
		postPlugins: postPlugin,
	}
	rm.init()
	return rm
}

func (rm *RuntimeManager) init() {
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
