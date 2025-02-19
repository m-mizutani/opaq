package opaq

import (
	"context"

	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown/print"
)

// Source is a function that returns a map of policy data.
//
// Example:
//
//	opaq.Files("./some/dir/policy")
type Source func() (map[string]string, error)

// Client is a client for the opaq.
type Client struct {
	policy   map[string]string
	compiler *ast.Compiler
	cfg      *config
}

type config struct {
	logger  *slog.Logger
	version ast.RegoVersion
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
//		client, err := opaq.New(opaq.Files("./some/dir/policy"))
//		if err != nil {
//			log.Fatal(err)
//		}
//
//	 var resp struct {
//			Allow bool `json:"allow"`
//		}
//		client.Query(context.Background(), "data.your_policy", map[string]any{"input": "input"}, &resp)
func New(src Source, options ...Option) (*Client, error) {
	policy, err := src()
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	cfg := &config{
		logger:  slog.New(slog.NewTextHandler(&noopWriter{}, nil)),
		version: ast.RegoV1,
	}
	for _, opt := range options {
		opt(cfg)
	}

	compiler, err := ast.CompileModulesWithOpt(policy, ast.CompileOpts{
		EnablePrintStatements: true,
		ParserOptions: ast.ParserOptions{
			ProcessAnnotation: true,
			RegoVersion:       cfg.version,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compile policy: %w", err)
	}

	return &Client{
		policy:   policy,
		compiler: compiler,
		cfg:      cfg,
	}, nil
}

// Metadata returns the annotation set of the policy data. It works only for local policy data (File or Data).
func (c *Client) Metadata() ast.FlatAnnotationsRefSet {
	as := c.compiler.GetAnnotationSet()
	return as.Flatten()
}

// Sources returns the policy data. It works only for local policy data (File or Data).
func (c *Client) Sources() map[string]string {
	return c.policy
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
	logger := c.cfg.logger.With("query_id", rand.Text())
	regoOptions := []func(r *rego.Rego){
		rego.Query(query),
		rego.Compiler(c.compiler),
		rego.Input(input),
	}

	var cfg queryCfg
	for _, opt := range options {
		opt(&cfg)
	}

	if cfg.printHook != nil {
		c.cfg.logger.Debug("Setting print hook")
		regoOptions = append(regoOptions, rego.PrintHook(cfg.printHook))
	}

	q := rego.New(regoOptions...)

	logger.Debug("Evaluating query", "query", query, "input", input)
	rs, err := q.Eval(ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate query: %w", err)
	}
	logger.Debug("Query evaluated", "result", rs)

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return ErrNoEvalResult
	}

	raw, err := json.Marshal(rs[0].Expressions[0].Value)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}
	logger.Debug("Unmarshaled result", "output", output)

	return nil
}

// WithPrintHook sets the print hook for the query. The print hook is used to capture the print statements in the policy evaluation.
func WithPrintHook(h print.Hook) QueryOption {
	return func(o *queryCfg) {
		o.printHook = h
	}
}

type queryCfg struct {
	printHook print.Hook
}

type QueryOption func(*queryCfg)
