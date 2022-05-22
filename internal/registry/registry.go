package registry

import (
	"context"
	"fmt"
	"net/http"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/internal/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Registry is a registry of functions.
type Registry struct {
	functions map[string]*functions.RegisteredFunction
	paths     map[string]string
}

var defaultInstance = New()

// Default returns the default, singleton registry instance.
func Default() *Registry {
	return defaultInstance
}

func New() *Registry {
	return &Registry{
		functions: map[string]*functions.RegisteredFunction{},
		paths:     map[string]string{},
	}
}

func (r *Registry) IsEmpty() bool {
	return len(r.functions) == 0
}

func (r *Registry) GetFunctionNames() []string {
	funcNames := []string{}
	for k := range r.functions {
		funcNames = append(funcNames, k)
	}
	return funcNames
}

// RegisterHTTP a HTTP function with a given name
func (r *Registry) RegisterHTTP(name string, fn func(http.ResponseWriter, *http.Request), options ...functions.FunctionOption) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}

	// append at the end to overwrite any option from user
	options = append(options, functions.WithFunctionName(name))
	options = append(options, functions.WithHTTP(fn))

	function, err := functions.New(options...)
	if err != nil {
		return err
	}

	path := function.GetPath()
	if _, ok := r.paths[path]; ok {
		return fmt.Errorf("function path already registered: %s", path)
	}

	r.functions[name] = function
	r.paths[path] = name
	return nil
}

// RegistryCloudEvent a CloudEvent function with a given name
func (r *Registry) RegisterCloudEvent(name string, fn func(context.Context, cloudevents.Event) error, options ...functions.FunctionOption) error {
	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}

	// append at the end to overwrite any option from user
	options = append(options, functions.WithFunctionName(name))
	options = append(options, functions.WithCloudEvent(fn))

	function, err := functions.New(options...)
	if err != nil {
		return err
	}

	path := function.GetPath()
	if _, ok := r.paths[path]; ok {
		return fmt.Errorf("function path already registered: %s", path)
	}

	r.functions[name] = function
	r.paths[path] = name
	return nil
}

// RegisterOpenFunction a OpenFunction function with a given name
func (r *Registry) RegisterOpenFunction(name string, fn func(ofctx.Context, []byte) (ofctx.Out, error), options ...functions.FunctionOption) error {

	if _, ok := r.functions[name]; ok {
		return fmt.Errorf("function name already registered: %s", name)
	}

	// append at the end to overwrite any option from user
	options = append(options, functions.WithFunctionName(name))
	options = append(options, functions.WithOpenFunction(fn))

	function, err := functions.New(options...)
	if err != nil {
		return err
	}

	path := function.GetPath()
	if _, ok := r.paths[path]; ok {
		return fmt.Errorf("function path already registered: %s", path)
	}

	r.functions[name] = function
	r.paths[path] = name

	return nil
}

// GetRegisteredFunction a registered function by name
func (r *Registry) GetRegisteredFunction(name string) (*functions.RegisteredFunction, bool) {
	fn, ok := r.functions[name]
	return fn, ok
}
