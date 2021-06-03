package openfunctioncontext

const (
	HTTPPut     OutputMethod = "PUT"
	HTTPPost    OutputMethod = "POST"
	HTTPGet     OutputMethod = "GET"
	HTTPDelete  OutputMethod = "DELETE"
	GRPC        Kind         = "gRPC"
	HTTP        Kind         = "HTTP"
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
	State     interface{} `json:"state,omitempty"`
	Out       interface{} `json:"out,omitempty"`
}

type Outputs struct {
	Enabled       *bool              `json:"enabled"`
	Kind          Kind               `json:"kind"`
	OutputObjects map[string]*Output `json:"output_objects"`
}

type Input struct {
	Name    string       `json:"name"`
	Enabled *bool        `json:"enabled"`
	Pattern string       `json:"pattern"`
	Port    string       `json:"port"`
	Kind    Kind         `json:"kind"`
	InType  ResourceType `json:"in_type,omitempty"`
}

type Output struct {
	Pattern   string       `json:"pattern"`
	ReqMethod OutputMethod `json:"request_method,omitempty"`
}

type Runtime string

type OutputMethod string

type Kind string

type ResourceType string
