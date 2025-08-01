package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	stdpath "path"
	"strings"
	"sync"

	"github.com/oaswrap/spec/openapi"
	"github.com/oaswrap/spec/option"
	"github.com/oaswrap/spec/pkg/util"
)

// generator implements the Generator interface for creating OpenAPI specifications.
type generator struct {
	reflector reflector
	spec      spec
	cfg       *openapi.Config

	prefix string
	groups []*generator
	routes []*route
	opts   []option.GroupOption
	once   sync.Once
}

var _ Generator = (*generator)(nil)

// NewRouter returns a new Router instance using the given OpenAPI options.
//
// It is equivalent to NewGenerator.
//
// See also: NewGenerator.
func NewRouter(opts ...option.OpenAPIOption) Generator {
	return NewGenerator(opts...)
}

// NewGenerator returns a new Generator instance using the given OpenAPI options.
//
// It initializes the OpenAPI reflector and configuration.
func NewGenerator(opts ...option.OpenAPIOption) Generator {
	cfg := option.WithOpenAPIConfig(opts...)

	reflector := newReflector(cfg)

	return &generator{
		reflector: reflector,
		spec:      reflector.Spec(),
		cfg:       cfg,
	}
}

// Config returns the OpenAPI configuration used by the Generator.
func (g *generator) Config() *openapi.Config {
	return g.cfg
}

// Get registers a GET operation for the given path and options.
func (g *generator) Get(path string, opts ...option.OperationOption) Route {
	return g.Add("GET", path, opts...)
}

// Post registers a POST operation for the given path and options.
func (g *generator) Post(path string, opts ...option.OperationOption) Route {
	return g.Add("POST", path, opts...)
}

// Put registers a PUT operation for the given path and options.
func (g *generator) Put(path string, opts ...option.OperationOption) Route {
	return g.Add("PUT", path, opts...)
}

// Delete registers a DELETE operation for the given path and options.
func (g *generator) Delete(path string, opts ...option.OperationOption) Route {
	return g.Add("DELETE", path, opts...)
}

// Patch registers a PATCH operation for the given path and options.
func (g *generator) Patch(path string, opts ...option.OperationOption) Route {
	return g.Add("PATCH", path, opts...)
}

// Options registers an OPTIONS operation for the given path and options.
func (g *generator) Options(path string, opts ...option.OperationOption) Route {
	return g.Add("OPTIONS", path, opts...)
}

// Trace registers a TRACE operation for the given path and options.
func (g *generator) Trace(path string, opts ...option.OperationOption) Route {
	return g.Add("TRACE", path, opts...)
}

// Head registers a HEAD operation for the given path and options.
func (g *generator) Head(path string, opts ...option.OperationOption) Route {
	return g.Add("HEAD", path, opts...)
}

// Add registers an operation for the given HTTP method, path, and options.
func (g *generator) Add(method, path string, opts ...option.OperationOption) Route {
	if g.prefix != "" {
		path = g.cleanPath(path)
	}
	route := &route{
		method: method,
		path:   path,
		opts:   opts,
	}
	g.routes = append(g.routes, route)

	return route
}

// Route registers a nested route under the given pattern.
func (g *generator) Route(pattern string, fn func(router Router), opts ...option.GroupOption) Router {
	subGroup := g.Group(pattern, opts...)
	fn(subGroup)
	return subGroup
}

// Group creates a new sub-router with the given path prefix and group options.
func (g *generator) Group(pattern string, opts ...option.GroupOption) Router {
	group := &generator{
		prefix:    g.cleanPath(pattern),
		reflector: g.reflector,
		cfg:       g.cfg,
		opts:      opts,
	}
	g.groups = append(g.groups, group)
	return group
}

// Use applies one or more group options to the router.
func (g *generator) Use(opts ...option.GroupOption) Router {
	g.opts = append(g.opts, opts...)
	return g
}

// MarshalYAML and MarshalJSON implement the YAML and JSON serialization for the OpenAPI specification.
func (g *generator) MarshalYAML() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g.spec.MarshalYAML()
}

// MarshalJSON implements the JSON serialization for the OpenAPI specification.
func (g *generator) MarshalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	schema, err := g.spec.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err := json.Indent(&buffer, schema, "", "  "); err != nil {
		return nil, fmt.Errorf("failed to indent OpenAPI JSON schema: %w", err)
	}

	return buffer.Bytes(), nil
}

// GenerateSchema generates the OpenAPI schema in the specified format (JSON or YAML).
func (g *generator) GenerateSchema(formats ...string) ([]byte, error) {
	format := util.Optional("yaml", formats...)
	if format != "json" && format != "yaml" && format != "yml" {
		return nil, fmt.Errorf("unsupported format: %s, expected 'json', 'yaml', or 'yml'", format)
	}

	if format == "yaml" || format == "yml" {
		return g.MarshalYAML()
	}

	return g.MarshalJSON()
}

// WriteSchemaTo writes the OpenAPI schema to a file.
func (g *generator) WriteSchemaTo(path string) error {
	format := "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	} else if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		return fmt.Errorf("unsupported file extension: %s, expected '.json' or '.yaml' or '.yml'", path)
	}
	schema, err := g.GenerateSchema(format)
	if err != nil {
		return err
	}
	return os.WriteFile(path, schema, 0644)
}

// Validate checks whether the OpenAPI specification is valid.
func (g *generator) Validate() error {
	g.buildOnce()

	return g.reflector.Validate()
}

func (g *generator) buildOnce() {
	g.once.Do(func() {
		for _, r := range g.build() {
			g.reflector.Add(r.method, r.path, r.opts...)
		}
	})
}

func (g *generator) build() []*route {
	var routes []*route
	for _, r := range g.routes {
		var opts []option.OperationOption

		if len(g.opts) > 0 {
			cfg := &option.GroupConfig{}
			for _, opt := range g.opts {
				opt(cfg)
			}
			if cfg.Hide {
				continue
			}
			if cfg.Deprecated {
				opts = append(opts, option.Deprecated(true))
			}
			if len(cfg.Tags) > 0 {
				opts = append(opts, option.Tags(cfg.Tags...))
			}
			if len(cfg.Security) > 0 {
				for _, sec := range cfg.Security {
					opts = append(opts, option.Security(sec.Name, sec.Scopes...))
				}
			}
		}
		if len(r.opts) > 0 {
			r.opts = append(r.opts, opts...)
		}
		routes = append(routes, r)
	}

	for _, group := range g.groups {
		group.opts = append(g.opts, group.opts...)
		routes = append(routes, group.build()...)
	}
	return routes
}

func (g *generator) cleanPath(path string) string {
	cleaned := stdpath.Join(g.prefix, path)
	cleaned = stdpath.Clean(cleaned)
	return cleaned
}

type route struct {
	method string
	path   string
	opts   []option.OperationOption
}

var _ Route = (*route)(nil)

func (r *route) With(opts ...option.OperationOption) Route {
	r.opts = append(r.opts, opts...)
	return r
}
