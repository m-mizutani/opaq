package opaq

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// serverSource configures a Client to query an OPA server via HTTP.
type serverSource struct {
	baseURL string
}

func (s *serverSource) kind() sourceKind { return sourceKindServer }

func (s *serverSource) setup(cfg *config) (*setupResult, error) {
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &setupResult{
		querier: &serverQuerier{
			baseURL:    strings.TrimRight(s.baseURL, "/"),
			httpClient: httpClient,
			headers:    cfg.headers,
			headerFunc: cfg.headerFunc,
			logger:     cfg.logger,
		},
	}, nil
}

// Server returns a Source that queries an OPA server at the given base URL.
//
// Example:
//
//	client, err := opaq.New(opaq.Server("http://localhost:8181"))
func Server(baseURL string) Source {
	return &serverSource{baseURL: baseURL}
}

// serverQuerier sends HTTP requests to an OPA server.
type serverQuerier struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
	headerFunc func(ctx context.Context) (http.Header, error)
	logger     *slog.Logger
}

type opaRequest struct {
	Input any `json:"input"`
}

type opaResponse struct {
	Result json.RawMessage `json:"result"`
}

func (q *serverQuerier) query(ctx context.Context, queryStr string, input, output any, cfg *queryCfg) error {
	if cfg.printHook != nil {
		return fmt.Errorf("WithPrintHook is only available with local source")
	}

	logger := q.logger.With("query_id", rand.Text())

	path, err := queryToPath(queryStr)
	if err != nil {
		return err
	}

	reqBody, err := json.Marshal(opaRequest{Input: input})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := q.baseURL + path
	logger.Debug("Sending query to OPA server", "url", url, "query", queryStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Apply static headers
	for k, v := range q.headers {
		req.Header.Set(k, v)
	}

	// Apply dynamic headers (overrides static)
	if q.headerFunc != nil {
		dynHeaders, err := q.headerFunc(ctx)
		if err != nil {
			return fmt.Errorf("failed to get dynamic headers: %w", err)
		}
		for k, vals := range dynHeaders {
			for i, v := range vals {
				if i == 0 {
					req.Header.Set(k, v)
				} else {
					req.Header.Add(k, v)
				}
			}
		}
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OPA server returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.Debug("OPA server response", "status", resp.StatusCode, "body", string(body))

	var opaResp opaResponse
	if err := json.Unmarshal(body, &opaResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if opaResp.Result == nil {
		return ErrNoEvalResult
	}

	if err := json.Unmarshal(opaResp.Result, output); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	logger.Debug("Unmarshaled result", "output", output)
	return nil
}

// queryToPath converts a Rego query string to an OPA REST API path.
// Only queries starting with "data." are supported (maps to /v1/data/).
//
//	"data.authz.allow" → "/v1/data/authz/allow"
//	"data.authz"       → "/v1/data/authz"
func queryToPath(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query must not be empty")
	}
	if !strings.HasPrefix(query, "data.") {
		return "", fmt.Errorf("query must start with \"data.\" (only /v1/data endpoint is supported): got %q", query)
	}

	// Remove "data." prefix and replace dots with slashes
	rest := strings.TrimPrefix(query, "data.")
	if rest == "" {
		return "/v1/data", nil
	}

	path := "/v1/data/" + strings.ReplaceAll(rest, ".", "/")
	return path, nil
}
