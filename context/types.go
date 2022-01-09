package context

import (
	"context"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
)

const (
	Async               Runtime      = "Async"
	Knative             Runtime      = "Knative"
	OpenFuncBinding     ResourceType = "bindings"
	OpenFuncTopic       ResourceType = "pubsub"
	Success             ReturnCode   = 200
	InternalError       ReturnCode   = 500
	defaultPort                      = "8080"
	daprSidecarGRPCPort              = "50001"
)

type Runtime string
type ReturnCode int
type ResourceType string

type Context struct {
	Name            string               `json:"name"`
	Version         string               `json:"version"`
	RequestID       string               `json:"requestID,omitempty"`
	Ctx             context.Context      `json:"ctx,omitempty"`
	Inputs          map[string]*Input    `json:"inputs,omitempty"`
	Outputs         map[string]*Output   `json:"outputs,omitempty"`
	Runtime         Runtime              `json:"runtime"`
	Port            string               `json:"port,omitempty"`
	State           interface{}          `json:"state,omitempty"`
	EventMeta       *EventMetadata       `json:"event,omitempty"`
	SyncRequestMeta *SyncRequestMetadata `json:"syncRequest,omitempty"`
	PrePlugins      []string             `json:"prePlugins,omitempty"`
	PostPlugins     []string             `json:"postPlugins,omitempty"`
	Out             Out                  `json:"out,omitempty"`
	Error           error                `json:"error,omitempty"`
	HttpPattern     string               `json:"httpPattern,omitempty"`
	daprClient      dapr.Client
}

type EventMetadata struct {
	InputName    string               `json:"inputName,omitempty"`
	BindingEvent *common.BindingEvent `json:"bindingEvent,omitempty"`
	TopicEvent   *common.TopicEvent   `json:"topicEvent,omitempty"`
	CloudEvent   *cloudevents.Event   `json:"cloudEventnt,omitempty"`
}

type SyncRequestMetadata struct {
	ResponseWriter http.ResponseWriter `json:"responseWriter,omitempty"`
	Request        *http.Request       `json:"request,omitempty"`
}

type Input struct {
	Uri       string            `json:"uri,omitempty"`
	Component string            `json:"component,omitempty"`
	Type      ResourceType      `json:"type"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Output struct {
	Uri       string            `json:"uri,omitempty"`
	Component string            `json:"component,omitempty"`
	Type      ResourceType      `json:"type"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Operation string            `json:"operation,omitempty"`
}

type Out struct {
	Code     ReturnCode        `json:"code"`
	Data     []byte            `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    error             `json:"error,omitempty"`
}
