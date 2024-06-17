package context

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SkyAPM/go2sky"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	"github.com/go-chi/chi/v5"
	"k8s.io/klog/v2"
	agentv3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

var (
	daprGRPCHost           string
	daprGRPCPort           string
	bindingQueueComponents = map[string]bool{
		"bindings.kafka":                  true,
		"bindings.rabbitmq":               true,
		"bindings.aws.sqs":                true,
		"bindings.aws.kinesis":            true,
		"bindings.gcp.pubsub":             true,
		"bindings.azure.eventgrid":        true,
		"bindings.azure.eventhubs":        true,
		"bindings.azure.servicebusqueues": true,
		"bindings.azure.storagequeues":    true,
	}
)

const (
	TestModeEnvName                           = "TEST_MODE"
	FunctionContextEnvName                    = "FUNC_CONTEXT"
	PodNameEnvName                            = "POD_NAME"
	PodNamespaceEnvName                       = "POD_NAMESPACE"
	ModeEnvName                               = "CONTEXT_MODE"
	Async                        Runtime      = "Async"
	Knative                      Runtime      = "Knative"
	OpenFuncBinding              ResourceType = "bindings"
	OpenFuncTopic                ResourceType = "pubsub"
	Success                                   = 200
	InternalError                             = 500
	defaultPort                               = "8080"
	defaultHttpPattern                        = "/"
	defaultDaprHost                           = "127.0.0.1"
	defaultDaprGRPCPort                       = "50001"
	TracingProviderSkywalking                 = "skywalking"
	TracingProviderOpentelemetry              = "opentelemetry"
	KubernetesMode                            = "kubernetes"
	SelfHostMode                              = "self-host"
	TestModeOn                                = "on"
	innerEventTypePrefix                      = "io.openfunction.function"
	tracingProviderSkywalking                 = "skywalking"
	RawData                                   = Option("RawData") // This option controls the Send() function to send raw data
)

type Runtime string
type ResourceType string
type Option string

type NativeContext interface {
	// GetNativeContext returns the Go native context object.
	GetNativeContext() context.Context

	// SetNativeContext set the Go native context object.
	SetNativeContext(context.Context)
}

type RuntimeContext interface {
	NativeContext

	// GetName returns the function's name.
	GetName() string

	// GetMode returns the operating environment mode of the function.
	GetMode() string

	// GetContext returns the pointer of raw OpenFunction FunctionContext object.
	GetContext() *FunctionContext

	// GetOut returns the pointer of raw OpenFunction FunctionOut object.
	GetOut() Out

	// HasInputs detects if the function has any input sources.
	HasInputs() bool

	// HasOutputs detects if the function has any output targets.
	HasOutputs() bool

	// InitDaprClientIfNil detects whether the dapr client in the current FunctionContext has been initialized,
	// and initializes it if it has not been initialized.
	InitDaprClientIfNil()

	// DestroyDaprClient destroys the dapr client when the function is executed with an exception.
	DestroyDaprClient()

	// GetPrePlugins returns a list of plugin names for the previous phase of function execution.
	GetPrePlugins() []string

	// GetPostPlugins returns a list of plugin names for the post phase of function execution.
	GetPostPlugins() []string

	// GetRuntime returns the Runtime.
	GetRuntime() Runtime

	// GetPort returns the port that the function service is listening on.
	GetPort() string

	// GetError returns the error status of the function.
	GetError() error

	// GetHttpPattern returns the path of the server listening in Knative runtime mode.
	GetHttpPattern() string

	// SetSyncRequest sets the native http.ResponseWriter and *http.Request when an http request is received.
	SetSyncRequest(w http.ResponseWriter, r *http.Request)

	// SetEvent sets the name of the input source and the native event when an event request is received.
	SetEvent(inputName string, event interface{})

	// GetInputs returns the mapping relationship of *Input.
	GetInputs() map[string]*Input

	// GetOutputs returns the mapping relationship of *Output.
	GetOutputs() map[string]*Output

	// GetSyncRequest returns the pointer of SyncRequest.
	GetSyncRequest() *SyncRequest

	// GetBindingEvent returns the pointer of common.BindingEvent.
	GetBindingEvent() *common.BindingEvent

	// GetTopicEvent returns the pointer of common.TopicEvent.
	GetTopicEvent() *common.TopicEvent

	// GetCloudEvent returns the pointer of v2.Event.
	GetCloudEvent() *cloudevents.Event

	// GetInnerEvent returns the InnerEvent.
	GetInnerEvent() InnerEvent

	// WithOut adds the FunctionOut object to the RuntimeContext.
	WithOut(out *FunctionOut) RuntimeContext

	// WithError adds the error state to the RuntimeContext.
	WithError(err error) RuntimeContext

	// GetPodName returns the name of the pod the function is running on.
	GetPodName() string

	// GetPodNamespace returns the namespace of the pod the function is running on.
	GetPodNamespace() string

	// GetPluginsTracingCfg returns the TracingConfig interface.
	GetPluginsTracingCfg() TracingConfig

	// HasPluginsTracingCfg returns nil if there is no TracingConfig.
	HasPluginsTracingCfg() bool
}

type Context interface {
	NativeContext

	// Send provides the ability to allow the user to send data to a specified output target.
	Send(outputName string, data []byte) ([]byte, error)

	// ReturnOnSuccess returns the Out with a success state.
	ReturnOnSuccess() Out

	// ReturnOnInternalError returns the Out with an error state.
	ReturnOnInternalError() Out

	// GetSyncRequest returns the pointer of SyncRequest.
	GetSyncRequest() *SyncRequest

	// GetBindingEvent returns the pointer of common.BindingEvent.
	GetBindingEvent() *common.BindingEvent

	// GetTopicEvent returns the pointer of common.TopicEvent.
	GetTopicEvent() *common.TopicEvent

	// GetCloudEvent returns the pointer of v2.Event.
	GetCloudEvent() *cloudevents.Event

	// GetInnerEvent returns the InnerEvent.
	GetInnerEvent() InnerEvent

	// ContextOptions returns the context's options.
	ContextOptions() ContextOption

	// GetInputName return the inputName of the event
	GetInputName() string

	GetDaprClient() dapr.Client
}

type Out interface {

	// GetOut returns the pointer of raw FunctionOut object.
	GetOut() *FunctionOut

	// GetCode returns the return code in FunctionOut.
	GetCode() int

	// GetData returns the return data in FunctionOut.
	GetData() []byte

	// GetMetadata returns the metadata in FunctionOut.
	GetMetadata() map[string]string

	// WithCode sets the FunctionOut with new return code.
	WithCode(code int) *FunctionOut

	// WithData sets the FunctionOut with new return data.
	WithData(data []byte) *FunctionOut
}

type TracingConfig interface {

	// IsEnabled detects if the tracing configuration is enabled.
	IsEnabled() bool

	// ProviderName returns the name of tracing provider.
	ProviderName() string

	// ProviderOapServer returns the oap server of the tracing provider.
	ProviderOapServer() string

	// GetTags returns the tags of the tracing configuration.
	GetTags() map[string]string

	// GetBaggage returns the baggage of the tracing configuration.
	GetBaggage() map[string]string
}

type ContextOption interface {
	SetRawData(condition bool) ContextOption
	IsRawDataEnabled() bool
}

type FunctionContext struct {
	mu             sync.Mutex
	Name           string             `json:"name"`
	Version        string             `json:"version"`
	RequestID      string             `json:"requestID,omitempty"`
	Ctx            context.Context    `json:"ctx,omitempty"`
	Inputs         map[string]*Input  `json:"inputs,omitempty"`
	Outputs        map[string]*Output `json:"outputs,omitempty"`
	Runtime        Runtime            `json:"runtime"`
	Port           string             `json:"port,omitempty"`
	State          interface{}        `json:"state,omitempty"`
	Event          *EventRequest      `json:"event,omitempty"`
	SyncRequest    *SyncRequest       `json:"syncRequest,omitempty"`
	PrePlugins     []string           `json:"prePlugins,omitempty"`
	PostPlugins    []string           `json:"postPlugins,omitempty"`
	PluginsTracing *PluginsTracing    `json:"pluginsTracing,omitempty"`
	Out            Out                `json:"out,omitempty"`
	Error          error              `json:"error,omitempty"`
	HttpPattern    string             `json:"httpPattern,omitempty"`
	podName        string
	podNamespace   string
	daprClient     dapr.Client
	mode           string
	options        map[Option]string
}

type EventRequest struct {
	InputName    string               `json:"inputName,omitempty"`
	BindingEvent *common.BindingEvent `json:"bindingEvent,omitempty"`
	TopicEvent   *common.TopicEvent   `json:"topicEvent,omitempty"`
	CloudEvent   *cloudevents.Event   `json:"cloudEventnt,omitempty"`
	innerEvent   InnerEvent
}

type SyncRequest struct {
	ResponseWriter http.ResponseWriter `json:"responseWriter,omitempty"`
	Request        *http.Request       `json:"request,omitempty"`
}

type Input struct {
	Uri           string            `json:"uri,omitempty"`
	ComponentName string            `json:"componentName"`
	ComponentType string            `json:"componentType"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// GetType will be called after the context has been parsed correctly,
// therefore we do not have to handle the error return of getBuildingBlockType()
func (i *Input) GetType() ResourceType {
	bbt, _ := getBuildingBlockType(i.ComponentType)
	return bbt
}

type Output struct {
	Uri           string            `json:"uri,omitempty"`
	ComponentName string            `json:"componentName"`
	ComponentType string            `json:"componentType"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Operation     string            `json:"operation,omitempty"`
}

// GetType will be called after the context has been parsed correctly,
// therefore we do not have to handle the error return of getBuildingBlockType()
func (o *Output) GetType() ResourceType {
	bbt, _ := getBuildingBlockType(o.ComponentType)
	return bbt
}

type FunctionOut struct {
	mu       sync.Mutex
	Code     int               `json:"code"`
	Data     []byte            `json:"data,omitempty"`
	Error    error             `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type PluginsTracing struct {
	Enabled  bool              `json:"enabled" yaml:"enabled"`
	Provider *TracingProvider  `json:"provider" yaml:"provider"`
	Tags     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Baggage  map[string]string `json:"baggage" yaml:"baggage"`
}

type TracingProvider struct {
	Name      string `json:"name" yaml:"name"`
	OapServer string `json:"oapServer" yaml:"oapServer"`
}

type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rww *ResponseWriterWrapper) Status() int {
	return rww.statusCode
}

func (rww *ResponseWriterWrapper) Header() http.Header {
	return rww.ResponseWriter.Header()
}

func (rww *ResponseWriterWrapper) Write(bytes []byte) (int, error) {
	return rww.ResponseWriter.Write(bytes)
}

func (rww *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rww.statusCode = statusCode
	rww.ResponseWriter.WriteHeader(statusCode)
}

func NewResponseWriterWrapper(w http.ResponseWriter, statusCode int) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		w,
		statusCode,
	}
}

func (ctx *FunctionContext) Send(outputName string, data []byte) ([]byte, error) {
	if !ctx.HasOutputs() {
		return nil, errors.New("no output")
	}

	var err error
	var output *Output
	var response *dapr.BindingEvent
	var payload []byte

	if v, ok := ctx.Outputs[outputName]; ok {
		output = v
	} else {
		return nil, fmt.Errorf("output %s not found", outputName)
	}

	payload = data

	if IsTracingProviderSkyWalking(ctx) && traceable(output.ComponentType) && !ctx.IsRawDataEnabled() {
		ie := NewInnerEvent(ctx)
		ie.MergeMetadata(ctx.GetInnerEvent())
		ie.SetUserData(data)

		// Set the exit span for tracing
		if err := setExitSpan(ctx, ie, outputName); err != nil {
			klog.Warningf("failed to set exit span: %v", err)
		}

		payload = ie.GetCloudEventJSON()
	}

	switch output.GetType() {
	case OpenFuncTopic:
		err = ctx.daprClient.PublishEvent(context.Background(), output.ComponentName, output.Uri, payload)
	case OpenFuncBinding:
		in := &dapr.InvokeBindingRequest{
			Name:      output.ComponentName,
			Operation: output.Operation,
			Data:      payload,
			Metadata:  output.Metadata,
		}
		response, err = ctx.daprClient.InvokeBinding(context.Background(), in)
	}

	if err != nil {
		return nil, err
	}

	if response != nil {
		return response.Data, nil
	}
	return nil, nil
}

func (ctx *FunctionContext) GetDaprClient() dapr.Client {
	return ctx.daprClient
}

func (ctx *FunctionContext) HasInputs() bool {
	if len(ctx.GetInputs()) > 0 {
		return true
	}
	return false
}

func (ctx *FunctionContext) HasOutputs() bool {
	if len(ctx.GetOutputs()) > 0 {
		return true
	}
	return false
}

func (ctx *FunctionContext) ReturnOnSuccess() Out {
	return &FunctionOut{
		Code: Success,
	}
}

func (ctx *FunctionContext) ReturnOnInternalError() Out {
	return &FunctionOut{
		Code: InternalError,
	}
}

func (ctx *FunctionContext) InitDaprClientIfNil() {
	if testMode := os.Getenv(TestModeEnvName); testMode == TestModeOn {
		return
	}

	if ctx.daprClient == nil {
		var err error
		ctx.mu.Lock()
		defer ctx.mu.Unlock()

		for attempts := 120; attempts > 0; attempts-- {
			address := net.JoinHostPort(daprGRPCHost, daprGRPCPort)
			c, e := dapr.NewClientWithAddress(address)
			if e == nil {
				ctx.daprClient = c
				break
			}
			err = e
			time.Sleep(500 * time.Millisecond)
		}

		if ctx.daprClient == nil {
			klog.Errorf("failed to init dapr client: %v", err)
			panic(err)
		}
	}
}

func (ctx *FunctionContext) DestroyDaprClient() {
	if testMode := os.Getenv(TestModeEnvName); testMode == TestModeOn {
		return
	}

	if ctx.daprClient != nil {
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		ctx.daprClient.Close()
		ctx.daprClient = nil
	}
}

func (ctx *FunctionContext) GetPrePlugins() []string {
	return ctx.PrePlugins
}

func (ctx *FunctionContext) GetPostPlugins() []string {
	return ctx.PostPlugins
}

func (ctx *FunctionContext) GetRuntime() Runtime {
	return ctx.Runtime
}

func (ctx *FunctionContext) GetPort() string {
	return ctx.Port
}

func (ctx *FunctionContext) GetHttpPattern() string {
	return ctx.HttpPattern
}

func (ctx *FunctionContext) GetError() error {
	return ctx.Error
}

func (ctx *FunctionContext) GetMode() string {
	return ctx.mode
}

func (ctx *FunctionContext) GetNativeContext() context.Context {
	return ctx.Ctx
}

func (ctx *FunctionContext) SetNativeContext(c context.Context) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Ctx = c
}

func (ctx *FunctionContext) SetSyncRequest(w http.ResponseWriter, r *http.Request) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.SyncRequest.ResponseWriter = w
	ctx.SyncRequest.Request = r
}

func (ctx *FunctionContext) SetEvent(inputName string, event interface{}) {
	switch t := event.(type) {
	case *common.BindingEvent:
		be := event.(*common.BindingEvent)
		ie := convertEvent(ctx, inputName, be.Data)
		ctx.setEvent(inputName, be, nil, nil, ie)
	case *common.TopicEvent:
		te := event.(*common.TopicEvent)
		ie := convertEvent(ctx, inputName, ConvertUserDataToBytes(te.Data))
		ctx.setEvent(inputName, nil, te, nil, ie)
	case *cloudevents.Event:
		ce := event.(*cloudevents.Event)
		ie := convertEvent(ctx, inputName, ce.Data())
		ctx.setEvent(inputName, nil, nil, ce, ie)
	default:
		klog.Errorf("failed to resolve event type: %v", t)
	}
}

func (ctx *FunctionContext) setEvent(name string, be *common.BindingEvent, te *common.TopicEvent, ce *cloudevents.Event, ie InnerEvent) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Event.InputName = name
	ctx.Event.BindingEvent = be
	ctx.Event.TopicEvent = te
	ctx.Event.CloudEvent = ce
	ctx.Event.innerEvent = ie
}

func (ctx *FunctionContext) GetName() string {
	return ctx.Name
}

func (ctx *FunctionContext) GetContext() *FunctionContext {
	return ctx
}

func (ctx *FunctionContext) GetInputs() map[string]*Input {
	return ctx.Inputs
}

func (ctx *FunctionContext) GetOutputs() map[string]*Output {
	return ctx.Outputs
}

func (ctx *FunctionContext) GetPodName() string {
	return ctx.podName
}

func (ctx *FunctionContext) GetPodNamespace() string {
	return ctx.podNamespace
}

func (ctx *FunctionContext) GetSyncRequest() *SyncRequest {
	return ctx.SyncRequest
}

func (ctx *FunctionContext) GetBindingEvent() *common.BindingEvent {
	return ctx.Event.BindingEvent
}

func (ctx *FunctionContext) GetTopicEvent() *common.TopicEvent {
	return ctx.Event.TopicEvent
}

func (ctx *FunctionContext) GetCloudEvent() *cloudevents.Event {
	return ctx.Event.CloudEvent
}

func (ctx *FunctionContext) GetInnerEvent() InnerEvent {
	return ctx.Event.innerEvent
}

func (ctx *FunctionContext) GetInputName() string {
	return ctx.Event.InputName
}

func (ctx *FunctionContext) GetPluginsTracingCfg() TracingConfig {
	return ctx.PluginsTracing
}

func (ctx *FunctionContext) HasPluginsTracingCfg() bool {
	return ctx.PluginsTracing != nil
}

func (ctx *FunctionContext) WithOut(out *FunctionOut) RuntimeContext {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Out = out
	return ctx
}

func (ctx *FunctionContext) WithError(err error) RuntimeContext {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Error = err
	return ctx
}

func (ctx *FunctionContext) GetOut() Out {
	return ctx.Out
}

func (ctx *FunctionContext) setContextOption(key Option, value string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.options[key] = value
}

func (ctx *FunctionContext) getContextOption(key Option) string {
	if value, ok := ctx.options[key]; ok {
		return value
	} else {
		return ""
	}
}

func (ctx *FunctionContext) ContextOptions() ContextOption {
	return ctx
}

func (ctx *FunctionContext) SetRawData(enable bool) ContextOption {
	ctx.setContextOption(RawData, strconv.FormatBool(enable))
	return ctx
}

func (ctx *FunctionContext) IsRawDataEnabled() bool {
	if enable, err := strconv.ParseBool(ctx.getContextOption(RawData)); err != nil {
		return false
	} else {
		return enable
	}
}

func (o *FunctionOut) GetOut() *FunctionOut {
	return o
}

func (o *FunctionOut) GetCode() int {
	return o.Code
}

func (o *FunctionOut) GetData() []byte {
	return o.Data
}

func (o *FunctionOut) GetMetadata() map[string]string {
	return o.Metadata
}

func (o *FunctionOut) WithCode(code int) *FunctionOut {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Code = code
	return o
}

func (o *FunctionOut) WithData(data []byte) *FunctionOut {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Data = data
	return o
}

func (tracing *PluginsTracing) IsEnabled() bool {
	return tracing.Enabled
}

func (tracing *PluginsTracing) ProviderName() string {
	if tracing.Provider != nil {
		return tracing.Provider.Name
	} else {
		return ""
	}
}

func (tracing *PluginsTracing) ProviderOapServer() string {
	if tracing.Provider != nil {
		return tracing.Provider.OapServer
	} else {
		return ""
	}
}

func (tracing *PluginsTracing) GetTags() map[string]string {
	return tracing.Tags
}

func (tracing *PluginsTracing) GetBaggage() map[string]string {
	return tracing.Baggage
}

func registerTracingPluginIntoPrePlugins(plugins []string, target string) []string {
	if len(plugins) == 0 {
		plugins = append(plugins, target)
	} else if exist := hasPlugin(plugins, target); !exist {
		plugins = append(plugins, target)
	}
	return plugins
}

func registerTracingPluginIntoPostPlugins(plugins []string, target string) []string {
	if len(plugins) == 0 {
		plugins = append(plugins, target)
	} else if exist := hasPlugin(plugins, target); !exist {
		plugins = append(plugins[:1], plugins[:]...)
		plugins[0] = target
	}
	return plugins
}

func hasPlugin(plugins []string, target string) bool {
	for _, plg := range plugins {
		if plg == target {
			return true
		}
	}
	return false
}

func GetRuntimeContext() (RuntimeContext, error) {
	ctx, err := parseContext()
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

func CloneRuntimeContext(ctx RuntimeContext) RuntimeContext {
	return &FunctionContext{

		Name:    ctx.GetName(),
		Version: ctx.GetContext().Version,
		Inputs:  ctx.GetInputs(),
		Outputs: ctx.GetOutputs(),

		Runtime: ctx.GetRuntime(),
		Port:    ctx.GetPort(),
		State:   ctx.GetContext().State,

		PrePlugins:     ctx.GetPrePlugins(),
		PostPlugins:    ctx.GetPostPlugins(),
		PluginsTracing: ctx.GetContext().PluginsTracing,
		HttpPattern:    ctx.GetHttpPattern(),

		Event:        &EventRequest{},
		SyncRequest:  &SyncRequest{},
		mode:         ctx.GetMode(),
		podName:      ctx.GetPodName(),
		podNamespace: ctx.GetPodNamespace(),
		options:      ctx.GetContext().options,
		daprClient:   ctx.GetContext().daprClient,
	}
}

func parseContext() (*FunctionContext, error) {
	ctx := &FunctionContext{
		Inputs:  make(map[string]*Input),
		Outputs: make(map[string]*Output),
	}

	data := os.Getenv(FunctionContextEnvName)
	if data == "" {
		return nil, fmt.Errorf("env %s not found", FunctionContextEnvName)
	}

	err := json.Unmarshal([]byte(data), ctx)
	if err != nil {
		return nil, err
	}

	switch ctx.Runtime {
	case Async, Knative:
		break
	default:
		return nil, fmt.Errorf("invalid runtime: %s", ctx.Runtime)
	}

	ctx.Event = &EventRequest{}
	ctx.SyncRequest = &SyncRequest{}

	if ctx.HasInputs() {
		for name, in := range ctx.GetInputs() {
			if _, err := getBuildingBlockType(in.ComponentType); err != nil {
				klog.Errorf("failed to get building block type for input %s: %v", name, err)
				return nil, err
			}
		}
	}

	if ctx.HasOutputs() {
		for name, out := range ctx.GetOutputs() {
			if _, err := getBuildingBlockType(out.ComponentType); err != nil {
				klog.Errorf("failed to get building block type for output %s: %v", name, err)
				return nil, err
			}
		}
	}

	switch os.Getenv(ModeEnvName) {
	case SelfHostMode:
		ctx.mode = SelfHostMode
	default:
		ctx.mode = KubernetesMode
	}

	if ctx.mode == KubernetesMode {
		podName := os.Getenv(PodNameEnvName)
		if podName == "" {
			return nil, errors.New("the name of the pod cannot be retrieved from the environment, " +
				"you need to set the POD_NAME environment variable")
		}
		ctx.podName = podName

		podNamespace := os.Getenv(PodNamespaceEnvName)
		if podNamespace == "" {
			return nil, errors.New("the namespace of the pod cannot be retrieved from the environment, " +
				"you need to set the POD_NAMESPACE environment variable")
		}
		ctx.podNamespace = podNamespace
	}

	if ctx.PluginsTracing != nil && ctx.PluginsTracing.IsEnabled() {
		if ctx.PluginsTracing.Provider != nil && ctx.PluginsTracing.Provider.Name != "" {
			switch ctx.PluginsTracing.Provider.Name {
			case TracingProviderSkywalking, TracingProviderOpentelemetry:
				ctx.PrePlugins = registerTracingPluginIntoPrePlugins(ctx.PrePlugins, ctx.PluginsTracing.Provider.Name)
				ctx.PostPlugins = registerTracingPluginIntoPostPlugins(ctx.PostPlugins, ctx.PluginsTracing.Provider.Name)
			default:
				return nil, fmt.Errorf("invalid tracing provider name: %s", ctx.PluginsTracing.Provider.Name)
			}
			if ctx.PluginsTracing.Tags != nil {
				if funcName, ok := ctx.PluginsTracing.Tags["func"]; !ok || funcName != ctx.Name {
					ctx.PluginsTracing.Tags["func"] = ctx.Name
				}
				ctx.PluginsTracing.Tags["instance"] = ctx.podName
				ctx.PluginsTracing.Tags["namespace"] = ctx.podNamespace
			}
		} else {
			return nil, errors.New("the tracing plugin is enabled, but its configuration is incorrect")
		}
	}

	if ctx.Port == "" {
		ctx.Port = defaultPort
	} else {
		if _, err := strconv.Atoi(ctx.Port); err != nil {
			return nil, fmt.Errorf("error parsing port: %s", err.Error())
		}
	}

	if ctx.HttpPattern == "" {
		ctx.HttpPattern = defaultHttpPattern
	}

	// Support one-sidecar-per-function mode
	host := os.Getenv("DAPR_HOST")
	if host == "" {
		daprGRPCHost = defaultDaprHost
	} else {
		daprGRPCHost = host
	}

	// When using self-hosted mode, configure the client port via env,
	// refer to https://docs.dapr.io/reference/environment/
	port := os.Getenv("DAPR_GRPC_PORT")
	if port == "" {
		daprGRPCPort = defaultDaprGRPCPort
	} else {
		daprGRPCPort = port
	}

	// Initialize the context options
	newContextOptions(ctx)

	return ctx, nil
}

func NewFunctionOut() *FunctionOut {
	return &FunctionOut{}
}

func newContextOptions(ctx *FunctionContext) {
	ctx.options = map[Option]string{
		RawData: "false",
	}
}

// Convert queue binding event into cloud event format to add tracing metadata in the cloud event context.
func traceable(t string) bool {

	// All events sent to dapr pubsub components need to be encapsulated
	if strings.HasPrefix(t, "pubsub") {
		return true
	}

	// For dapr binding components, let the mapping conditions of the bindingQueueComponents
	// determine if the tracing metadata can be added.
	return bindingQueueComponents[t]
}

func getBuildingBlockType(componentType string) (ResourceType, error) {
	typeSplit := strings.Split(componentType, ".")
	if len(typeSplit) > 1 {
		t := typeSplit[0]
		switch ResourceType(t) {
		case OpenFuncBinding, OpenFuncTopic:
			return ResourceType(t), nil
		default:
			return "", fmt.Errorf("unknown component type: %s", t)
		}
	}
	return "", errors.New("invalid component type")
}

func setExitSpan(ctx *FunctionContext, innerEvent InnerEvent, target string) error {
	if !ctx.HasPluginsTracingCfg() || !ctx.GetPluginsTracingCfg().IsEnabled() {
		return nil
	}

	switch ctx.GetPluginsTracingCfg().ProviderName() {
	case tracingProviderSkywalking:
		tracer := go2sky.GetGlobalTracer()
		if tracer == nil {
			return errors.New("skywalking is not enabled")
		}

		span, err := tracer.CreateExitSpan(ctx.GetNativeContext(), ctx.GetName(), target, func(headerKey, headerValue string) error {
			innerEvent.SetMetadata(headerKey, headerValue)
			return nil
		})
		if err != nil {
			return err
		}
		defer span.End()

		span.SetSpanLayer(agentv3.SpanLayer_FAAS)
		span.SetComponent(5013)
		return nil
	default:
		return nil
	}
}

func ConvertUserDataToBytes(data interface{}) []byte {
	if d, ok := data.([]byte); ok {
		return d
	}
	if d, ok := data.(string); ok {
		return []byte(d)
	}
	if d, err := json.Marshal(data); err != nil {
		return nil
	} else {
		return d
	}
}

type contextKey int

const (
	varsKey contextKey = iota
)

// CtxWithVars is just for backward compatibility of VarsFromCtx to the previous implementation using gorilla/mux
// CtxWithVars adds URL Parameters into Context and user can get them by VarsFromCtx
// However, the recommended way to read URL Parameter in go-chi/chi is using URLParamFromCtx
func CtxWithVars(ctx context.Context, vars map[string]string) context.Context {
	return context.WithValue(ctx, varsKey, vars)
}

func VarsFromCtx(ctx context.Context) map[string]string {
	if rv := ctx.Value(varsKey); rv != nil {
		return rv.(map[string]string)
	}
	return nil
}

// URLParamFromCtx returns the url parameter from a http.Request Context.
func URLParamFromCtx(ctx context.Context, key string) string {
	return chi.URLParamFromCtx(ctx, key)
}

// URLParamsFromCtx returns all the url parameters from a http.Request Context.
func URLParamsFromCtx(ctx context.Context) map[string]string {
	res := map[string]string{}
	if rctx := chi.RouteContext(ctx); rctx != nil {
		for k := 0; k < len(rctx.URLParams.Keys); k++ {
			key := rctx.URLParams.Keys[k]
			val := rctx.URLParams.Values[k]
			res[key] = val
		}
	}
	return res
}

func IsTracingProviderSkyWalking(ctx RuntimeContext) bool {
	if ctx.HasPluginsTracingCfg() && ctx.GetPluginsTracingCfg().IsEnabled() &&
		ctx.GetPluginsTracingCfg().ProviderName() == TracingProviderSkywalking {
		return true
	}

	return false
}
