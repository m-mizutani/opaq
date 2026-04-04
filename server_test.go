package opaq_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
)

func TestQueryToPath(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    string
		wantErr bool
	}{
		{
			name:  "data.authz.allow",
			query: "data.authz.allow",
			want:  "/v1/data/authz/allow",
		},
		{
			name:  "data.authz",
			query: "data.authz",
			want:  "/v1/data/authz",
		},
		{
			name:  "data.example.policy.main",
			query: "data.example.policy.main",
			want:  "/v1/data/example/policy/main",
		},
		{
			name:  "data with trailing dot",
			query: "data.",
			want:  "/v1/data",
		},
		{
			name:  "data root",
			query: "data",
			want:  "/v1/data",
		},
		{
			name:    "no data prefix",
			query:   "authz.allow",
			wantErr: true,
		},
		{
			name:    "empty string",
			query:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// queryToPath is unexported, so test through server round-trip
			// This test verifies the path conversion via the OPA server mock
			_ = tt // path conversion is tested indirectly through integration tests
		})
	}

	// Direct test via server round-trip: verify the correct path is called
	for _, tt := range tests {
		if tt.wantErr {
			continue
		}
		t.Run("roundtrip/"+tt.name, func(t *testing.T) {
			var calledPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"allow": true}})
			}))
			defer srv.Close()

			client, err := opaq.New(opaq.Server(srv.URL))
			gt.NoError(t, err)

			var output struct {
				Allow bool `json:"allow"`
			}
			err = client.Query(context.Background(), tt.query, nil, &output)
			gt.NoError(t, err)
			gt.Value(t, calledPath).Equal(tt.want)
		})
	}
}

func TestServerQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Method).Equal(http.MethodPost)
		gt.Value(t, r.Header.Get("Content-Type")).Equal("application/json")

		var req struct {
			Input map[string]any `json:"input"`
		}
		gt.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		user, _ := req.Input["user"].(string)
		allow := user == "admin"

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"allow": allow},
		})
	}))
	defer srv.Close()

	client, err := opaq.New(opaq.Server(srv.URL))
	gt.NoError(t, err)

	t.Run("allow admin", func(t *testing.T) {
		var output struct {
			Allow bool `json:"allow"`
		}
		err := client.Query(context.Background(), "data.authz", map[string]any{"user": "admin"}, &output)
		gt.NoError(t, err)
		gt.Value(t, output.Allow).Equal(true)
	})

	t.Run("deny non-admin", func(t *testing.T) {
		var output struct {
			Allow bool `json:"allow"`
		}
		err := client.Query(context.Background(), "data.authz", map[string]any{"user": "bob"}, &output)
		gt.NoError(t, err)
		gt.Value(t, output.Allow).Equal(false)
	})
}

func TestServerQuery_InvalidQuery(t *testing.T) {
	client, err := opaq.New(opaq.Server("http://localhost:9999"))
	gt.NoError(t, err)

	var output struct{}

	t.Run("no data prefix", func(t *testing.T) {
		err := client.Query(context.Background(), "authz.allow", nil, &output)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("must start with \"data.\"")
	})

	t.Run("empty query", func(t *testing.T) {
		err := client.Query(context.Background(), "", nil, &output)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("must not be empty")
	})
}

func TestServerQuery_StaticHeaders(t *testing.T) {
	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": true})
	}))
	defer srv.Close()

	client, err := opaq.New(
		opaq.Server(srv.URL),
		opaq.WithHeader("X-Custom", "test-value"),
		opaq.WithHeader("Authorization", "Bearer static-token"),
	)
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.NoError(t, err)
	gt.Value(t, receivedHeaders.Get("X-Custom")).Equal("test-value")
	gt.Value(t, receivedHeaders.Get("Authorization")).Equal("Bearer static-token")
}

func TestServerQuery_DynamicHeaders(t *testing.T) {
	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": true})
	}))
	defer srv.Close()

	callCount := 0
	client, err := opaq.New(
		opaq.Server(srv.URL),
		opaq.WithHeader("Authorization", "Bearer static-token"),
		opaq.WithHeaderFunc(func(ctx context.Context) (http.Header, error) {
			callCount++
			return http.Header{
				"Authorization": []string{"Bearer dynamic-token"},
				"X-Request-Id":  []string{"req-123"},
			}, nil
		}),
	)
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.NoError(t, err)
	gt.Value(t, callCount).Equal(1)
	// Dynamic header should override static
	gt.Value(t, receivedHeaders.Get("Authorization")).Equal("Bearer dynamic-token")
	gt.Value(t, receivedHeaders.Get("X-Request-Id")).Equal("req-123")

	// Second call should invoke headerFunc again
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.NoError(t, err)
	gt.Value(t, callCount).Equal(2)
}

func TestServerQuery_DynamicHeaderError(t *testing.T) {
	client, err := opaq.New(
		opaq.Server("http://localhost:9999"),
		opaq.WithHeaderFunc(func(ctx context.Context) (http.Header, error) {
			return nil, context.DeadlineExceeded
		}),
	)
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("failed to get dynamic headers")
}

func TestServerQuery_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer srv.Close()

	client, err := opaq.New(opaq.Server(srv.URL))
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("status 500")
}

func TestServerQuery_InvalidResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client, err := opaq.New(opaq.Server(srv.URL))
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("failed to decode response")
}

func TestServerQuery_NoResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	client, err := opaq.New(opaq.Server(srv.URL))
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("no evaluation result")
}

func TestServerQuery_PrintHookError(t *testing.T) {
	client, err := opaq.New(opaq.Server("http://localhost:9999"))
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output,
		opaq.WithPrintHook(func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
			return nil
		}),
	)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("WithPrintHook is only available with local source")
}

func TestOptionCompatibility(t *testing.T) {
	t.Run("WithHTTPClient with local source", func(t *testing.T) {
		_, err := opaq.New(
			opaq.DataMap(map[string]string{"test.rego": "package test\nallow = true"}),
			opaq.WithHTTPClient(&http.Client{}),
		)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("WithHTTPClient is only available with Server source")
	})

	t.Run("WithHeader with local source", func(t *testing.T) {
		_, err := opaq.New(
			opaq.DataMap(map[string]string{"test.rego": "package test\nallow = true"}),
			opaq.WithHeader("X-Custom", "value"),
		)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("WithHeader is only available with Server source")
	})

	t.Run("WithHeaderFunc with local source", func(t *testing.T) {
		_, err := opaq.New(
			opaq.DataMap(map[string]string{"test.rego": "package test\nallow = true"}),
			opaq.WithHeaderFunc(func(ctx context.Context) (http.Header, error) {
				return nil, nil
			}),
		)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("WithHeaderFunc is only available with Server source")
	})

	t.Run("WithRegoVersion with server source", func(t *testing.T) {
		_, err := opaq.New(
			opaq.Server("http://localhost:9999"),
			opaq.WithRegoVersion(1),
		)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("WithRegoVersion is only available with local source")
	})

	t.Run("WithRelPath with server source", func(t *testing.T) {
		_, err := opaq.New(
			opaq.Server("http://localhost:9999"),
			opaq.WithRelPath("testdata"),
		)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("WithRelPath is only available with local source")
	})
}

func TestServerQuery_CustomHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": true})
	}))
	defer srv.Close()

	customClient := &http.Client{}
	client, err := opaq.New(
		opaq.Server(srv.URL),
		opaq.WithHTTPClient(customClient),
	)
	gt.NoError(t, err)

	var output bool
	err = client.Query(context.Background(), "data.test", nil, &output)
	gt.NoError(t, err)
	gt.Value(t, output).Equal(true)
}
