package opaq

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
)

// localQuerier evaluates queries against locally compiled policies.
type localQuerier struct {
	compiler *ast.Compiler
	logger   *slog.Logger
}

func (q *localQuerier) query(ctx context.Context, queryStr string, input, output any, cfg *queryCfg) error {
	logger := q.logger.With("query_id", rand.Text())
	regoOptions := []func(r *rego.Rego){
		rego.Query(queryStr),
		rego.Compiler(q.compiler),
		rego.Input(input),
	}

	if cfg.printHook != nil {
		q.logger.Debug("Setting print hook")
		regoOptions = append(regoOptions, rego.PrintHook(&printHook{hook: cfg.printHook}))
	}

	r := rego.New(regoOptions...)

	logger.Debug("Evaluating query", "query", queryStr, "input", input)
	rs, err := r.Eval(ctx)
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

// compileLocal compiles policy data and returns a setupResult with a localQuerier.
func compileLocal(policy map[string]string, cfg *config) (*setupResult, error) {
	if cfg.relBasePath != "" {
		newPolicy := make(map[string]string)
		for k, v := range policy {
			rel, err := filepath.Rel(cfg.relBasePath, k)
			if err != nil {
				continue
			}
			newPolicy[rel] = v
		}
		policy = newPolicy
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

	return &setupResult{
		querier:  &localQuerier{compiler: compiler, logger: cfg.logger},
		policy:   policy,
		compiler: compiler,
	}, nil
}

// fileSource loads policy files from directories.
type fileSource struct {
	paths []string
}

func (s *fileSource) kind() sourceKind { return sourceKindLocal }

func (s *fileSource) setup(cfg *config) (*setupResult, error) {
	policy := make(map[string]string)

	for _, file := range s.paths {
		err := filepath.WalkDir(file, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".rego" {
				return nil
			}

			fpath := filepath.Clean(path)
			raw, err := os.ReadFile(fpath)
			if err != nil {
				return fmt.Errorf("failed to read policy file: %w", err)
			}

			policy[fpath] = string(raw)

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	return compileLocal(policy, cfg)
}

// Files returns a Source that reads policy files. The method gets the policy files that have `.rego` extension, and gets the policy files recursively from the given paths.
func Files(files ...string) Source {
	return &fileSource{paths: files}
}

// dataSource provides inline policy data.
type dataSource struct {
	args []string
}

func (s *dataSource) kind() sourceKind { return sourceKindLocal }

func (s *dataSource) setup(cfg *config) (*setupResult, error) {
	data := make(map[string]string)
	for i := 0; i < len(s.args); i += 2 {
		if i+1 >= len(s.args) {
			return nil, fmt.Errorf("invalid number of arguments")
		}
		data[s.args[i]] = s.args[i+1]
	}
	return compileLocal(data, cfg)
}

// Data returns a Source that have given data. The arguments are pairs of key and value.
// Example:
//
//	opaq.Data("pkg1", "package pkg1\n\nallow := true\n")
func Data(args ...string) Source {
	return &dataSource{args: args}
}

// dataMapSource provides inline policy data from a map.
type dataMapSource struct {
	data map[string]string
}

func (s *dataMapSource) kind() sourceKind { return sourceKindLocal }

func (s *dataMapSource) setup(cfg *config) (*setupResult, error) {
	policy := make(map[string]string, len(s.data))
	for k, v := range s.data {
		policy[k] = v
	}
	return compileLocal(policy, cfg)
}

// DataMap returns a Source that have given data. The argument is a map of key and value.
// Example:
//
//	opaq.DataMap(map[string]string{
//		"pkg1": "package pkg1\n\nallow := true\n",
//	})
func DataMap(data map[string]string) Source {
	return &dataMapSource{data: data}
}
