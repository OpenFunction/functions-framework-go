package functions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	HTTPType         = "http"
	CloudEventType   = "cloudevent"
	OpenFunctionType = "openfunction"
	defaultPath      = "/"
)

// RegisteredFunction represents a function that has been
// registered with the registry.
type RegisteredFunction struct {
	functionName   string                                         // The name of the function
	functionPath   string                                         // The path of the function, default is '/'
	functionType   string                                         // The type of the function, not using it currently
	httpFn         func(http.ResponseWriter, *http.Request)       // Optional: The user's HTTP function
	cloudEventFn   func(context.Context, cloudevents.Event) error // Optional: The user's CloudEvent function
	openFunctionFn func(ofctx.Context, []byte) (ofctx.Out, error) // Optional: The user's OpenFunction function
}

type FunctionOption func() (func(*RegisteredFunction), error)

func (rf *RegisteredFunction) setup(options ...FunctionOption) error {
	if rf == nil {
		return nil
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		setter, err := option()
		if err != nil {
			return err
		}
		if setter != nil {
			setter(rf)
		}
	}

	if rf.GetName() == "" {
		return errors.New("No function name is registered")
	}

	if rf.GetFunctionType() == "" {
		return errors.New("No function is registered")
	}

	return nil
}

func (rf *RegisteredFunction) GetName() string {
	return rf.functionName
}

func (rf *RegisteredFunction) GetPath() string {
	return rf.functionPath
}

func (rf *RegisteredFunction) GetFunctionType() string {
	return rf.functionType
}

func (rf *RegisteredFunction) GetHTTPFunction() func(http.ResponseWriter, *http.Request) {
	return rf.httpFn
}

func (rf *RegisteredFunction) GetCloudEventFunction() func(context.Context, cloudevents.Event) error {
	return rf.cloudEventFn
}

func (rf *RegisteredFunction) GetOpenFunctionFunction() func(ofctx.Context, []byte) (ofctx.Out, error) {
	return rf.openFunctionFn
}

// failedOption - helper to expose error from option builder
func failedOption(err error) FunctionOption {
	return func() (func(*RegisteredFunction), error) {
		return nil, err
	}
}

// properOption - helper to expose valid setter from option builder
func properOption(setter func(*RegisteredFunction)) FunctionOption {
	return func() (func(*RegisteredFunction), error) {
		return setter, nil
	}
}

func New(options ...FunctionOption) (*RegisteredFunction, error) {
	rf := &RegisteredFunction{functionPath: defaultPath}

	if err := rf.setup(options...); err != nil {
		return nil, err
	}

	return rf, nil
}

func WithFunctionName(name string) FunctionOption {
	if !isValidFunctionName(name) {
		return failedOption(fmt.Errorf("Invalid function name: %s", name))
	}
	return properOption(func(rf *RegisteredFunction) {
		rf.functionName = name
	})
}

// Returns true if the function name is valid
// - must contain only alphanumeric, numbers, or dash characters
// - must be <= 63 characters
// - must start with a letter
// - must end with a letter or number
func isValidFunctionName(name string) bool {
	match, _ := regexp.MatchString("^[A-Za-z](?:[-_A-Za-z0-9]{0,61}[A-Za-z0-9])?$", name)
	return match
}

func WithFunctionPath(path string) FunctionOption {
	if len(path) == 0 {
		return failedOption(errors.New("Empty function path"))
	}

	if path[0] != '/' {
		return failedOption(fmt.Errorf("Function path must start with '/': %s", path))
	}

	return properOption(func(rf *RegisteredFunction) {
		rf.functionPath = path
	})
}

func WithHTTP(fn func(http.ResponseWriter, *http.Request)) FunctionOption {
	if fn == nil {
		return failedOption(errors.New("Function is nil"))
	}

	return properOption(func(rf *RegisteredFunction) {
		rf.functionType = HTTPType
		rf.httpFn = fn
	})
}

func WithCloudEvent(fn func(context.Context, cloudevents.Event) error) FunctionOption {
	if fn == nil {
		return failedOption(errors.New("Function is nil"))
	}

	return properOption(func(rf *RegisteredFunction) {
		rf.functionType = CloudEventType
		rf.cloudEventFn = fn
	})
}

func WithOpenFunction(fn func(ofctx.Context, []byte) (ofctx.Out, error)) FunctionOption {
	if fn == nil {
		return failedOption(errors.New("Function is nil"))
	}

	return properOption(func(rf *RegisteredFunction) {
		rf.functionType = OpenFunctionType
		rf.openFunctionFn = fn
	})
}
