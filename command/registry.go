package command

import (
	"github.com/celestix/gotgproto/dispatcher"
)

// Registry manages module and command registration
type Registry struct {
	modules       []Module
	defaultPrefix string
}

// NewRegistry creates a new registry with the given default prefix
func NewRegistry(defaultPrefix string) *Registry {
	return &Registry{
		modules:       make([]Module, 0),
		defaultPrefix: defaultPrefix,
	}
}

// AddModule adds a module to the registry
func (r *Registry) AddModule(module Module) {
	r.modules = append(r.modules, module)
}

// RegisterAll registers all modules with the given dispatcher
func (r *Registry) RegisterAll(d dispatcher.Dispatcher) {
	for _, module := range r.modules {
		module.Load(d, r.defaultPrefix)
	}
}

// GetModules returns all registered modules
func (r *Registry) GetModules() []Module {
	return r.modules
}
