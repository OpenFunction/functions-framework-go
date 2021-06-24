package openfunctioncontext

const (
	OpenFuncAsync   Runtime      = "OpenFuncAsync"
	Knative         Runtime      = "Knative"
	OpenFuncBinding ResourceType = "bindings"
	OpenFuncService ResourceType = "invoke"
	OpenFuncTopic   ResourceType = "pubsub"
)

type OpenFunctionContext struct {
	Name      string             `json:"name"`
	Version   string             `json:"version"`
	RequestID string             `json:"requestID,omitempty"`
	Input     Input              `json:"input,omitempty"`
	Outputs   map[string]*Output `json:"outputs,omitempty"`
	Runtime   Runtime            `json:"runtime"`
	Port      string             `json:"port,omitempty"`
	State     interface{}        `json:"state,omitempty"`
}

type Input struct {
	Name   string            `json:"name"`
	Uri    string            `json:"uri"`
	Params map[string]string `json:"params,omitempty"`
}

type Output struct {
	Uri    string            `json:"uri"`
	Params map[string]string `json:"params,omitempty"`
}

type Runtime string

type Protocol string

type ResourceType string
