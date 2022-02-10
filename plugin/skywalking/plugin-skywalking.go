package skywalking

import (
	"context"
	"sync"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"k8s.io/klog/v2"

	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

const (
	name    = "skywalking"
	version = "v1"

	componentIDOpenFunction = 5013 // https://github.com/apache/skywalking/blob/master/oap-server/server-starter/src/main/resources/component-libraries.yml#L515
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

func initGo2sky(ofCtx ofctx.Context, p *PluginSkywalking) {
	initGo2skyOnce.Do(func() {
		r, err := reporter.NewGRPCReporter(ofCtx.PluginsTracing.Provider.OapServer, reporter.WithLog(&klogWrapper{}))
		if err != nil {
			klog.Errorf("new go2sky grpc reporter error\n", err)
			return
		}
		tracer, err := go2sky.NewTracer(ofCtx.Name, go2sky.WithReporter(r), go2sky.WithInstance(ofCtx.PluginsTracing.Tags["instance"]))
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
		return preSyncRequestLogic(&ctx, p.tracer)
	}
	return nil
}

func (p *PluginSkywalking) ExecPostHook(ctx ofctx.Context, plugins map[string]plugin.Plugin) error {
	if p.tracer == nil {
		return nil
	}
	if ctx.SyncRequestMeta.Request != nil {
		return postSyncRequestLogic(&ctx)
	}
	return nil
}

func (p PluginSkywalking) Get(fieldName string) (interface{}, bool) {
	return nil, false
}

func setPublicAttrs(ctx context.Context, ofCtx *ofctx.Context, span go2sky.Span) {
	span.SetSpanLayer(agentv3.SpanLayer_FAAS)
	span.SetComponent(componentIDOpenFunction)

	// tags
	for key, value := range ofCtx.PluginsTracing.Tags {
		span.Tag(go2sky.Tag(key), value)
	}
	// baggage
	for key, value := range ofCtx.PluginsTracing.Baggage {
		go2sky.PutCorrelation(ctx, key, value)
	}

}
