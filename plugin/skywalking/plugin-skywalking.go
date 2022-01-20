package skywalking

import (
	"sync"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"k8s.io/klog/v2"
)

const (
	name    = "skywalking"
	version = "v1"
)

var (
	initGo2skyOnce sync.Once
)

type klogWrapper struct {
}

func (k klogWrapper) Info(args ...interface{}) {
	klog.Info(args)
}

func (k klogWrapper) Infof(format string, args ...interface{}) {
	klog.Infof(format, args)
}

func (k klogWrapper) Warn(args ...interface{}) {
	klog.Warning(args)
}

func (k klogWrapper) Warnf(format string, args ...interface{}) {
	klog.Warningf(format, args)
}

func (k klogWrapper) Error(args ...interface{}) {
	klog.Error(args)
}

func (k klogWrapper) Errorf(format string, args ...interface{}) {
	klog.Errorf(format, args)
}

func initGo2sky(ctx ofctx.Context, p *PluginSkywalking) {
	// SW_AGENT_COLLECTOR_BACKEND_SERVICES
	initGo2skyOnce.Do(func() {
		//backend := os.Getenv("SW_AGENT_COLLECTOR_BACKEND_SERVICES")
		//if backend == "" {
		//	return
		//}
		//r, err := reporter.NewGRPCReporter(backend, reporter.WithLog(&klogWrapper{}))
		//if err != nil {
		//	klog.Errorf("new go2sky grpc reporter error\n", err)
		//	return
		//}
		r, err := reporter.NewLogReporter()
		if err != nil {
			return
		}
		tracer, err := go2sky.NewTracer(ctx.Name, go2sky.WithReporter(r))
		if err != nil {
			klog.Errorf("new go2sky tracer error\n", err)
			return
		}
		go2sky.SetGlobalTracer(tracer)

		p.tracer = tracer
	})
}

type PluginSkywalking struct {
	tracer *go2sky.Tracer
}

func (p *PluginSkywalking) Init() plugin.Plugin {
	return p
}

func (p PluginSkywalking) Name() string {
	return name
}

func (p PluginSkywalking) Version() string {
	return version
}

func (p *PluginSkywalking) ExecPreHook(ctx ofctx.Context, plugins map[string]plugin.Plugin) error {
	initGo2sky(ctx, p)
	if p.tracer == nil {
		return nil
	}

	if ctx.SyncRequestMeta.Request != nil {
		// SyncRequest
		return preSyncRequestLogic(ctx, p.tracer)
	} else if ctx.EventMeta.BindingEvent != nil {

	} else if ctx.EventMeta.CloudEvent != nil {
		// Cloud event

	}
	return nil
}

func (p *PluginSkywalking) ExecPostHook(ctx ofctx.Context, plugins map[string]plugin.Plugin) error {
	if p.tracer == nil {
		return nil
	}
	if ctx.SyncRequestMeta.Request != nil {
		return postSyncRequestLogic(ctx)
	}

	return nil
}

func (p PluginSkywalking) Get(fieldName string) (interface{}, bool) {
	return nil, false
}
