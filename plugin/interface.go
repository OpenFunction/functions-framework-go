package plugin

import (
	ofctx "github.com/OpenFunction/functions-framework-go/context"
)

type Metadata interface {
	Name() string
	Version() string
}

type Plugin interface {
	Metadata
	Init() Plugin
	ExecPreHook(ctx ofctx.Context, plugins map[string]Plugin) error
	ExecPostHook(ctx ofctx.Context, plugins map[string]Plugin) error
	Get(fieldName string) (interface{}, bool)
}
