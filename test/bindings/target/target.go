package main

import (
	"context"

	"github.com/fatih/structs"
	"k8s.io/klog/v2"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
)

func main() {
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		klog.Exit(err)
	}
	fwk.RegisterPlugins(getLocalPlugins())
	if err := fwk.Register(ctx, Target); err != nil {
		klog.Exit(err)
	}
	if err := fwk.Start(ctx); err != nil {
		klog.Exit(err)
	}
}

func getLocalPlugins() map[string]plugin.Plugin {
	localPlugins := map[string]plugin.Plugin{
		Name: New(),
	}

	if len(localPlugins) == 0 {
		return nil
	} else {
		return localPlugins
	}
}

func Target(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	klog.Infof("bindings - Data: %s", in)
	return ctx.ReturnOnSuccess(), nil
}

// Plugin

const (
	Name    = "plugin-custom"
	Version = "v1"
)

type PluginCustom struct {
	PluginName    string
	PluginVersion string
	StateC        int64
}

var _ plugin.Plugin = &PluginCustom{}

func New() *PluginCustom {
	return &PluginCustom{
		StateC: int64(0),
	}
}

func (p *PluginCustom) Name() string {
	return Name
}

func (p *PluginCustom) Version() string {
	return Version
}

func (p *PluginCustom) Init() plugin.Plugin {
	return New()
}

func (p *PluginCustom) ExecPreHook(ctx ofctx.RuntimeContext, plugins map[string]plugin.Plugin) error {
	p.StateC++
	return nil
}

func (p *PluginCustom) ExecPostHook(ctx ofctx.RuntimeContext, plugins map[string]plugin.Plugin) error {
	return nil
}

func (p *PluginCustom) Get(fieldName string) (interface{}, bool) {
	plgMap := structs.Map(p)
	value, ok := plgMap[fieldName]
	return value, ok
}
