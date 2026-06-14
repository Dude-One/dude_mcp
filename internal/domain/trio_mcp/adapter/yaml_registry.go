package adapter

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/trio/triov2-mcp/internal/domain/trio_mcp/entity"
	"gopkg.in/yaml.v3"
)

// yamlSource mirrors the sources.yaml structure for unmarshalling.
type yamlSource struct {
	Name           string `yaml:"name"`
	PlatformDomain string `yaml:"platform_domain"`
	Description    string `yaml:"description"`
	Type           string `yaml:"type"`
	BaseURL        string `yaml:"base_url"`
	SwaggerPath    string `yaml:"swagger_path"`
	ContextPath    string `yaml:"context_path"`
}

type yamlFile struct {
	Sources []yamlSource `yaml:"sources"`
}

// YAMLRegistry loads sources.yaml at startup.
// Used only for first-run DB seeding — DB is the source of truth after that.
type YAMLRegistry struct {
	mu      sync.RWMutex
	path    string
	sources map[string]entity.Source
}

func NewYAMLRegistry(path string) (*YAMLRegistry, error) {
	r := &YAMLRegistry{path: path}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *YAMLRegistry) Reload() error {
	return r.load()
}

func (r *YAMLRegistry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("reading sources file %q: %w", r.path, err)
	}

	var f yamlFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing sources file: %w", err)
	}

	sources := make(map[string]entity.Source, len(f.Sources))
	for _, s := range f.Sources {
		sources[s.Name] = entity.Source{
			Name:           s.Name,
			PlatformDomain: s.PlatformDomain,
			Description:    s.Description,
			Type:           entity.SourceType(s.Type),
			BaseURL:        s.BaseURL,
			SwaggerPath:    s.SwaggerPath,
			ContextPath:    s.ContextPath,
			IsActive:       true,
		}
	}

	r.mu.Lock()
	r.sources = sources
	r.mu.Unlock()
	return nil
}

func (r *YAMLRegistry) ListSources(_ context.Context) ([]entity.Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]entity.Source, 0, len(r.sources))
	for _, s := range r.sources {
		out = append(out, s)
	}
	return out, nil
}

func (r *YAMLRegistry) GetSource(_ context.Context, name string) (*entity.Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sources[name]
	if !ok {
		return nil, fmt.Errorf("source %q not registered", name)
	}
	return &s, nil
}
