package publishers

import (
	"fmt"
	"strings"
	"sync"
)

// Builder creates a Publisher from a config entry.
type Builder func(cfg PublisherConfig) (Publisher, error)

// Registry maps publisher types to builders.
type Registry interface {
	Register(typ string, builder Builder)
	PublisherFor(cfg PublisherConfig) (Publisher, error)
}

type registry struct {
	mu       sync.RWMutex
	builders map[string]Builder
}

// NewRegistry returns a registry with optional pre-registered builders.
func NewRegistry(builders map[string]Builder) Registry {
	r := &registry{
		builders: make(map[string]Builder),
	}
	for typ, b := range builders {
		r.Register(typ, b)
	}
	return r
}

// Register associates a builder with a publisher type.
func (r *registry) Register(typ string, builder Builder) {
	if typ = strings.TrimSpace(strings.ToLower(typ)); typ == "" || builder == nil {
		return
	}

	r.mu.Lock()
	r.builders[typ] = builder
	r.mu.Unlock()
}

// PublisherFor returns the publisher built for the provided config.
func (r *registry) PublisherFor(cfg PublisherConfig) (Publisher, error) {
	if cfg.Type == "" {
		return nil, fmt.Errorf("publisher %q has no type configured", cfg.ID)
	}

	r.mu.RLock()
	builder := r.builders[strings.ToLower(cfg.Type)]
	r.mu.RUnlock()

	if builder == nil {
		return nil, fmt.Errorf("no publisher registered for type %q", cfg.Type)
	}
	return builder(cfg)
}

// DefaultRegistry wires up known publishers.
func DefaultRegistry() Registry {
	builders := map[string]Builder{
		TypeHTTP: newHTTPPublisher,
		TypeSQS:  newSQSPublisher,
	}
	return NewRegistry(builders)
}

// BuildAll instantiates publishers for configs using the registry.
func BuildAll(reg Registry, cfgs []PublisherConfig) ([]Publisher, error) {
	if reg == nil || len(cfgs) == 0 {
		return nil, nil
	}

	var pubs []Publisher
	for _, cfg := range cfgs {
		pub, err := reg.PublisherFor(cfg)
		if err != nil {
			return nil, err
		}
		pubs = append(pubs, pub)
	}
	return pubs, nil
}
