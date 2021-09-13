package openfunctioncontext

import "github.com/dapr/go-sdk/service/common"

const (
	OpenFuncAsync   Runtime      = "OpenFuncAsync"
	Knative         Runtime      = "Knative"
	OpenFuncBinding ResourceType = "bindings"
	OpenFuncTopic   ResourceType = "pubsub"
	Success         ReturnCode   = 200
	InternalError   ReturnCode   = 500
)

type OpenFunctionContext struct {
	Name      string             `json:"name"`
	Version   string             `json:"version"`
	RequestID string             `json:"requestID,omitempty"`
	Inputs    map[string]*Input  `json:"inputs,omitempty"`
	Outputs   map[string]*Output `json:"outputs,omitempty"`
	Runtime   Runtime            `json:"runtime"`
	Port      string             `json:"port,omitempty"`
	State     interface{}        `json:"state,omitempty"`
	Event     *EventMetadata     `json:"event,omitempty"`
}

type EventMetadata struct {
	InputName    string               `json:"inputName,omitempty"`
	BindingEvent *common.BindingEvent `json:"bindingEvent,omitempty"`
	TopicEvent   *common.TopicEvent   `json:"topicEvent,omitempty"`
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

type Runtime string
type ReturnCode int
type ResourceType string

type Return struct {
	Code     ReturnCode        `json:"code"`
	Data     []byte            `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    string            `json:"error,omitempty"`
}
