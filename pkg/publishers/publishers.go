package publishers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	// Supported publisher types.
	TypeSQS  = "sqs"
	TypeHTTP = "http"

	httpDefaultMethod         = "POST"
	httpDefaultTimeoutSeconds = 5
)

// configFile represents the structure of the publishers configuration file.
type configFile struct {
	Publishers []PublisherConfig `json:"publishers" yaml:"publishers"`
}

// PublisherConfig represents a single publisher entry declared in config files.
type PublisherConfig struct {
	ID      string               `json:"id" yaml:"id"`
	Type    string               `json:"type" yaml:"type"`
	Enabled *bool                `json:"enabled" yaml:"enabled"`
	SQS     *SQSPublisherConfig  `json:"sqs" yaml:"sqs"`
	HTTP    *HTTPPublisherConfig `json:"http" yaml:"http"`
}

// SQSPublisherConfig holds AWS SQS specific settings.
type SQSPublisherConfig struct {
	QueueURL string `json:"uri" yaml:"uri"`
	Region   string `json:"region" yaml:"region"`
}

// HTTPPublisherConfig holds generic HTTP sink settings.
type HTTPPublisherConfig struct {
	URL            string            `json:"url" yaml:"url"`
	Method         string            `json:"method" yaml:"method"`
	Headers        map[string]string `json:"headers" yaml:"headers"`
	TimeoutSeconds int               `json:"timeout_seconds" yaml:"timeout_seconds"`
}

// ConfigRegistry materializes publisher definitions loaded from config files.
type ConfigRegistry struct {
	mu         sync.RWMutex
	publishers []PublisherConfig
	idx        map[string]PublisherConfig
}

// LoadRegistry loads the publisher registry from a YAML/JSON file.
func LoadRegistry(path string) (*ConfigRegistry, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("publishers file path is empty")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open publishers file: %w", err)
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read publishers file: %w", err)
	}

	fileReg, err := parsePublisherRegistry(raw, filepath.Ext(path))
	if err != nil {
		return nil, err
	}
	if len(fileReg.Publishers) == 0 {
		return nil, errors.New("publishers file contains no publishers entries")
	}

	reg := &ConfigRegistry{
		publishers: make([]PublisherConfig, len(fileReg.Publishers)),
		idx:        make(map[string]PublisherConfig, len(fileReg.Publishers)),
	}

	for i := range fileReg.Publishers {
		cfg := sanitizePublisherConfig(fileReg.Publishers[i])
		if err := validatePublisherConfig(cfg); err != nil {
			return nil, fmt.Errorf("publishers[%d]: %w", i, err)
		}
		if _, exists := reg.idx[cfg.ID]; exists {
			return nil, fmt.Errorf("duplicate publisher id %q", cfg.ID)
		}
		reg.publishers[i] = cfg
		reg.idx[cfg.ID] = cfg
	}

	return reg, nil
}

// parsePublisherRegistry attempts to decode the publishers file content.
func parsePublisherRegistry(data []byte, ext string) (configFile, error) {
	ext = strings.ToLower(strings.TrimSpace(ext))
	decoders := []struct {
		name string
		ext  string
		fn   func([]byte, any) error
	}{
		{name: "yaml", ext: ".yaml", fn: yaml.Unmarshal},
		{name: "yaml", ext: ".yml", fn: yaml.Unmarshal},
		{name: "json", ext: ".json", fn: json.Unmarshal},
	}

	for _, d := range decoders {
		if ext != "" && ext != d.ext {
			continue
		}
		if reg, err := unmarshalPublisherRegistry(d.name, data, d.fn); err == nil {
			return reg, nil
		}
	}

	return configFile{}, errors.New("publishers file format not recognized (expected YAML or JSON)")
}

// unmarshalPublisherRegistry decodes the publishers file using the provided function.
func unmarshalPublisherRegistry(name string, data []byte, fn func([]byte, any) error) (configFile, error) {
	var reg configFile
	if err := fn(data, &reg); err != nil {
		return configFile{}, fmt.Errorf("decode %s publishers: %w", name, err)
	}
	return reg, nil
}

// sanitizePublisherConfig trims and normalizes the publisher config fields.
func sanitizePublisherConfig(cfg PublisherConfig) PublisherConfig {
	cfg.ID = strings.TrimSpace(cfg.ID)
	cfg.Type = strings.ToLower(strings.TrimSpace(cfg.Type))

	if cfg.Enabled == nil {
		def := true
		cfg.Enabled = &def
	}
	if cfg.SQS != nil {
		c := *cfg.SQS
		c.QueueURL = strings.TrimSpace(c.QueueURL)
		c.Region = strings.TrimSpace(c.Region)
		cfg.SQS = &c
	}
	if cfg.HTTP != nil {
		c := *cfg.HTTP
		c.URL = strings.TrimSpace(c.URL)
		c.Method = strings.ToUpper(strings.TrimSpace(c.Method))
		if c.Method == "" {
			c.Method = httpDefaultMethod
		}
		c.Headers = sanitizeHeaders(c.Headers)
		if c.TimeoutSeconds <= 0 {
			c.TimeoutSeconds = httpDefaultTimeoutSeconds
		}
		cfg.HTTP = &c
	}

	return cfg
}

// sanitizeHeaders trims and removes empty headers.
func sanitizeHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// validatePublisherConfig checks that required fields are present.
func validatePublisherConfig(cfg PublisherConfig) error {
	if cfg.ID == "" {
		return errors.New("id is required")
	}
	if cfg.Type == "" {
		return fmt.Errorf("type is required for publisher %q", cfg.ID)
	}
	if cfg.Type == TypeSQS {
		if cfg.SQS == nil {
			return fmt.Errorf("sqs config required for publisher %q", cfg.ID)
		}
		if cfg.SQS.QueueURL == "" {
			return fmt.Errorf("sqs.uri is required for publisher %q", cfg.ID)
		}
		if cfg.SQS.Region == "" {
			return fmt.Errorf("sqs.region is required for publisher %q", cfg.ID)
		}
	}
	if cfg.Type == TypeHTTP {
		if cfg.HTTP == nil {
			return fmt.Errorf("http config required for publisher %q", cfg.ID)
		}
		if cfg.HTTP.URL == "" {
			return fmt.Errorf("http.url is required for publisher %q", cfg.ID)
		}
	}
	return nil
}

// PublisherByID returns the publisher config by id.
func (r *ConfigRegistry) ByID(id string) (PublisherConfig, bool) {
	if r == nil {
		return PublisherConfig{}, false
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return PublisherConfig{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.idx[id]
	return cfg, ok
}

// All returns all configured publishers.
func (r *ConfigRegistry) All() []PublisherConfig {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]PublisherConfig, len(r.publishers))
	copy(out, r.publishers)
	return out
}

// Enabled returns publishers that are enabled.
func (r *ConfigRegistry) Enabled() []PublisherConfig {
	if r == nil {
		return nil
	}

	all := r.All()
	if len(all) == 0 {
		return nil
	}

	out := make([]PublisherConfig, 0, len(all))
	for _, cfg := range all {
		if cfg.EnabledValue() {
			out = append(out, cfg)
		}
	}
	return out
}

// EnabledValue returns enabled flag defaulting to true.
func (cfg PublisherConfig) EnabledValue() bool {
	if cfg.Enabled == nil {
		return true
	}
	return *cfg.Enabled
}
