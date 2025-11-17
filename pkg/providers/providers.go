package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Package providers contains pluggable provider configs (YAML/JSON) helpers.

type Provider struct {
	ID             string         `json:"id" yaml:"id"`
	Name           string         `json:"name" yaml:"name"`
	Type           string         `json:"type" yaml:"type"`
	SourceURL      string         `json:"source_url" yaml:"source_url"`
	ResponseFormat string         `json:"response_format" yaml:"response_format"`
	RequestDelayMs int            `json:"request_delay_ms" yaml:"request_delay_ms"`
	Config         map[string]any `json:"config" yaml:"config"`
}

type registry struct {
	Providers []Provider `json:"providers" yaml:"providers"`
}

var (
	regMu                 sync.RWMutex
	currentReg            registry
	providersIdx          map[string]Provider
	defaultRequestDelayMs = 500
)

// Providers returns a copy of the currently loaded providers registry.
func Providers() []Provider {
	regMu.RLock()
	defer regMu.RUnlock()

	if len(currentReg.Providers) == 0 {
		return nil
	}

	out := make([]Provider, len(currentReg.Providers))
	copy(out, currentReg.Providers)
	return out
}

// ProviderByID returns the provider entry for the given id, if loaded.
func ProviderByID(id string) (Provider, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Provider{}, false
	}

	regMu.RLock()
	defer regMu.RUnlock()

	if providersIdx == nil {
		return Provider{}, false
	}

	p, ok := providersIdx[id]
	return p, ok
}

// LoadProviders loads provider registry from file.
func LoadProviders(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("providers file path is empty")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open providers file: %w", err)
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read providers file: %w", err)
	}

	reg, err := parseRegistry(raw, filepath.Ext(path))
	if err != nil {
		return err
	}

	if len(reg.Providers) == 0 {
		return errors.New("providers file contains no providers entries")
	}

	idx := make(map[string]Provider, len(reg.Providers))
	for i := range reg.Providers {
		p := sanitizeProvider(reg.Providers[i])
		if err := validateProvider(p); err != nil {
			return fmt.Errorf("provider[%d]: %w", i, err)
		}
		if _, exists := idx[p.ID]; exists {
			return fmt.Errorf("duplicate provider id %q", p.ID)
		}
		reg.Providers[i] = p
		idx[p.ID] = p
	}

	regMu.Lock()
	currentReg = reg
	providersIdx = idx
	regMu.Unlock()

	return nil
}

func parseRegistry(data []byte, ext string) (registry, error) {
	ext = strings.ToLower(strings.TrimSpace(ext))

	decoders := []struct {
		name string
		ext  string
		fn   unmarshalFn
	}{
		{name: "yaml", ext: ".yaml", fn: yaml.Unmarshal},
		{name: "yaml", ext: ".yml", fn: yaml.Unmarshal},
		{name: "json", ext: ".json", fn: json.Unmarshal},
	}

	for _, d := range decoders {
		if ext != "" && ext != d.ext {
			continue
		}
		if reg, err := unmarshalRegistry(d.name, data, d.fn); err == nil {
			return reg, nil
		}
	}

	return registry{}, errors.New("providers file format not recognized (expected YAML or JSON)")
}

type unmarshalFn func([]byte, any) error

func unmarshalRegistry(name string, data []byte, fn unmarshalFn) (registry, error) {
	var reg registry
	if err := fn(data, &reg); err != nil {
		return registry{}, fmt.Errorf("decode %s providers: %w", name, err)
	}
	return reg, nil
}

func sanitizeProvider(p Provider) Provider {
	p.ID = strings.TrimSpace(p.ID)
	p.Name = strings.TrimSpace(p.Name)
	p.Type = strings.TrimSpace(p.Type)
	p.SourceURL = strings.TrimSpace(p.SourceURL)
	p.ResponseFormat = strings.TrimSpace(p.ResponseFormat)

	if p.Config == nil {
		p.Config = map[string]any{}
	}
	if p.RequestDelayMs <= 0 {
		p.RequestDelayMs = defaultRequestDelayMs
	}

	return p
}

func validateProvider(p Provider) error {
	if p.ID == "" {
		return errors.New("id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("name is required for provider %q", p.ID)
	}
	if p.Type == "" {
		return fmt.Errorf("type is required for provider %q", p.ID)
	}
	if p.SourceURL == "" {
		return fmt.Errorf("source_url is required for provider %q", p.ID)
	}
	if p.ResponseFormat == "" {
		return fmt.Errorf("response_format is required for provider %q", p.ID)
	}
	return nil
}

// RequestDelay returns the per-request throttle duration for the provider.
func (p Provider) RequestDelay() time.Duration {
	if p.RequestDelayMs <= 0 {
		return time.Duration(defaultRequestDelayMs) * time.Millisecond
	}
	return time.Duration(p.RequestDelayMs) * time.Millisecond
}
