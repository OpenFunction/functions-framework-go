package openfunctioncontext

const (
	GRPC        Protocol     = "gRPC"
	HTTP        Protocol     = "HTTP"
	Dapr        Runtime      = "Dapr"
	Knative     Runtime      = "Knative"
	DaprBinding ResourceType = "bindings"
	DaprService ResourceType = "invoke"
	DaprTopic   ResourceType = "pubsub"
)

type OpenFunctionContext struct {
	Name      string      `json:"name"`
	Version   string      `json:"version"`
	RequestID string      `json:"request_id,omitempty"`
	Input     *Input      `json:"input,omitempty"`
	Outputs   *Outputs    `json:"outputs,omitempty"`
	Runtime   Runtime     `json:"runtime"`
	Protocol  Protocol    `json:"protocol,omitempty"`
	Port      string      `json:"port,omitempty"`
	State     interface{} `json:"state,omitempty"`
}

type Outputs struct {
	Enabled       *bool              `json:"enabled"`
	OutputObjects map[string]*Output `json:"output_objects,omitempty"`
}

type Input struct {
	Name    string       `json:"name"`
	Enabled *bool        `json:"enabled"`
	Pattern string       `json:"pattern"`
	InType  ResourceType `json:"in_type,omitempty"`
}

type Output struct {
	Pattern string            `json:"pattern"`
	OutType ResourceType      `json:"out_type,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
}

type Runtime string

type OutputMethod string

type Protocol string

type ResourceType string
