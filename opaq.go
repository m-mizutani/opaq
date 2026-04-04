package opaq

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/topdown/print"
)

type sourceKind int

const (
	sourceKindLocal sourceKind = iota
	sourceKindServer
)

// Source configures how the Client evaluates queries.
type Source interface {
	setup(cfg *config) (*setupResult, error)
	kind() sourceKind
}

type setupResult struct {
	querier  querier
	policy   map[string]string
	compiler *ast.Compiler
}

type querier interface {
	query(ctx context.Context, queryStr string, input, output any, cfg *queryCfg) error
}

// Client is a client for the opaq.
type Client struct {
	policy   map[string]string
	compiler *ast.Compiler
	cfg      *config
	querier  querier
}

type config struct {
	logger      *slog.Logger
	version     ast.RegoVersion
	relBasePath string

	// server-specific
	httpClient *http.Client
	headers    map[string]string
	headerFunc func(ctx context.Context) (http.Header, error)

	// tracking explicitly set options for compatibility validation
	versionSet bool
}

type Option func(*config)

func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

func WithRegoVersion(version ast.RegoVersion) Option {
	return func(c *config) {
		c.version = version
		c.versionSet = true
	}
}

// WithRelPath sets the relative path to the policy data. If the relative path is set, the filepath of the policy data will be renamed to the relative path from the current working directory.
//
// Example:
//
//		opaq.Files("./some/dir/policy/example.rego", opaq.WithRelPath("./some/dir"))
//	 // the policy data will be renamed to "policy/example.rego"
func WithRelPath(relBasePath string) Option {
	return func(c *config) {
		c.relBasePath = relBasePath
	}
}

// WithHTTPClient sets the HTTP client for server queries. Only available with Server source.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.httpClient = client
	}
}

// WithHeader sets a static HTTP header for server queries. Can be called multiple times. Only available with Server source.
func WithHeader(key, value string) Option {
	return func(c *config) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

// WithHeaderFunc sets a dynamic header provider for server queries. The function is called on every Query invocation. Dynamic headers override static headers with the same key. Only available with Server source.
func WithHeaderFunc(fn func(ctx context.Context) (http.Header, error)) Option {
	return func(c *config) {
		c.headerFunc = fn
	}
}

type noopWriter struct{}

func (w *noopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// New creates a new opaq client.
//
// Example:
//
//	// Local policy evaluation
//	client, err := opaq.New(opaq.Files("./some/dir/policy"))
//
//	// Remote OPA server
//	client, err := opaq.New(opaq.Server("http://localhost:8181"))
func New(src Source, options ...Option) (*Client, error) {
	cfg := &config{
		logger:  slog.New(slog.NewTextHandler(&noopWriter{}, nil)),
		version: ast.RegoV1,
	}
	for _, opt := range options {
		opt(cfg)
	}

	if err := validateOptions(src.kind(), cfg); err != nil {
		return nil, err
	}

	result, err := src.setup(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Client{
		policy:   result.policy,
		compiler: result.compiler,
		cfg:      cfg,
		querier:  result.querier,
	}, nil
}

func validateOptions(kind sourceKind, cfg *config) error {
	switch kind {
	case sourceKindLocal:
		if cfg.httpClient != nil {
			return fmt.Errorf("WithHTTPClient is only available with Server source")
		}
		if cfg.headers != nil {
			return fmt.Errorf("WithHeader is only available with Server source")
		}
		if cfg.headerFunc != nil {
			return fmt.Errorf("WithHeaderFunc is only available with Server source")
		}
	case sourceKindServer:
		if cfg.versionSet {
			return fmt.Errorf("WithRegoVersion is only available with local source (Files, Data, DataMap)")
		}
		if cfg.relBasePath != "" {
			return fmt.Errorf("WithRelPath is only available with local source (Files, Data, DataMap)")
		}
	}
	return nil
}

// Metadata returns the annotation set of the policy data. It works only for local policy data (File or Data).
func (c *Client) Metadata() ast.FlatAnnotationsRefSet {
	if c.compiler == nil {
		return nil
	}
	as := c.compiler.GetAnnotationSet()
	return as.Flatten()
}

// Sources returns the policy data. It works only for local policy data (File or Data).
func (c *Client) Sources() map[string]string {
	if c.policy == nil {
		return nil
	}
	copied := make(map[string]string)
	maps.Copy(copied, c.policy)
	return copied
}

// Query evaluates the given query with the provided input and output. The query is evaluated against the policy data provided during client creation.
//
// Example:
//
//	input := map[string]any{
//		"input": "input",
//	}
//
//	var output struct {
//		Allow bool `json:"allow"`
//	}
//
//	if err := client.Query(context.Background(), "data.your_policy.allow", input, &output); err != nil {
//		log.Fatal(err)
//	}
func (c *Client) Query(ctx context.Context, query string, input, output any, options ...QueryOption) error {
	var cfg queryCfg
	for _, opt := range options {
		opt(&cfg)
	}
	return c.querier.query(ctx, query, input, output, &cfg)
}

type printHook struct {
	hook Hook
}

type PrintLocation struct {
	Text   []byte `json:"-"`    // The original text fragment from the source.
	File   string `json:"file"` // The name of the source file (which may be empty).
	Row    int    `json:"row"`  // The line in the source.
	Col    int    `json:"col"`  // The column in the row.
	Offset int    `json:"-"`    // The byte offset for the location in the source.

	Tabs []int `json:"-"` // The column offsets of tabs in the source.
}

func (h *printHook) Print(ctx print.Context, msg string) error {
	loc := PrintLocation{
		Text:   ctx.Location.Text,
		File:   ctx.Location.File,
		Row:    ctx.Location.Row,
		Col:    ctx.Location.Col,
		Offset: ctx.Location.Offset,
		Tabs:   ctx.Location.Tabs,
	}
	return h.hook(ctx.Context, loc, msg)
}

type Hook func(ctx context.Context, loc PrintLocation, msg string) error

// WithPrintHook sets the print hook for the query. The print hook is used to capture the print statements in the policy evaluation. Only available with local source.
//
// Example:
//
//	hook := func(ctx print.Context, msg string) error {
//		fmt.Println("📣", msg) // Show print statements from Rego policy
//		return nil
//	}
func WithPrintHook(h Hook) QueryOption {
	return func(o *queryCfg) {
		o.printHook = h
	}
}

type queryCfg struct {
	printHook Hook
}

type QueryOption func(*queryCfg)
