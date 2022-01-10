package skywalking

import (
	"fmt"
	"log"
	"net/http"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/plugin"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

const (
	name    = "skywalking"
	version = "v1"
)

func New() plugin.Plugin {
	r, err := reporter.NewLogReporter()
	if err != nil {
		log.Fatalf("new reporter error %v \n", err)
	}
	tracer, err := go2sky.NewTracer("example", go2sky.WithReporter(r))
	return &PluginSkywalking{
		tracer: tracer,
	}
}

type PluginSkywalking struct {
	tracer *go2sky.Tracer
}

func (p PluginSkywalking) Name() string {
	return name
}

func (p PluginSkywalking) Version() string {
	return version
}

func (p PluginSkywalking) ExecPreHook(ctx ofctx.Context, plugins map[string]plugin.Plugin) error {
	if ctx.SyncRequestMeta.Request == nil {
		return nil
	}
	request := ctx.SyncRequestMeta.Request

	span, nCtx, err := p.tracer.CreateEntrySpan(ctx.SyncRequestMeta.Request.Context(), getOperationName(request), func(key string) (string, error) {
		return request.Header.Get(key), nil
	})
	if err != nil {
		return err
	}
	ctx.SyncRequestMeta.Request = request.WithContext(nCtx)

	span.SetComponent(1)
	span.Tag(go2sky.TagHTTPMethod, request.Method)
	span.Tag(go2sky.TagURL, fmt.Sprintf("%s%s", request.Host, request.URL.Path))
	span.SetSpanLayer(agentv3.SpanLayer_Http)
	return nil
}

func (p PluginSkywalking) ExecPostHook(ctx ofctx.Context, plugins map[string]plugin.Plugin) error {
	if ctx.SyncRequestMeta.Request == nil {
		return nil
	}
	request := ctx.SyncRequestMeta.Request

	span := go2sky.ActiveSpan(request.Context())
	if span == nil {
		return nil
	}

	span.End()
	return nil
}

func (p PluginSkywalking) Get(fieldName string) (interface{}, bool) {
	return nil, false
}

func getOperationName(r *http.Request) string {
	return fmt.Sprintf("/%s%s", r.Method, r.URL.Path)
}

type responseWriterWrapper struct {
	w          http.ResponseWriter
	statusCode int
}

func (rww *responseWriterWrapper) Header() http.Header {
	return rww.w.Header()
}

func (rww *responseWriterWrapper) Write(bytes []byte) (int, error) {
	return rww.w.Write(bytes)
}

func (rww *responseWriterWrapper) WriteHeader(statusCode int) {
	rww.statusCode = statusCode
	rww.w.WriteHeader(statusCode)
}
