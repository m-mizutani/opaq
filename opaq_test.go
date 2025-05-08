package opaq_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/open-policy-agent/opa/v1/ast"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		source      opaq.Source
		options     []opaq.Option
		wantErr     bool
		errContains string
	}{
		{
			name: "valid policy",
			source: func() (map[string]string, error) {
				return map[string]string{
					"test.rego": `package test

					allow = true`,
				}, nil
			},
		},
		{
			name: "invalid policy",
			source: func() (map[string]string, error) {
				return map[string]string{
					"test.rego": `package test

					allow = invalid_syntax`,
				}, nil
			},
			wantErr:     true,
			errContains: "failed to compile policy",
		},
		{
			name: "source error",
			source: func() (map[string]string, error) {
				return nil, &testError{"test error"}
			},
			wantErr:     true,
			errContains: "failed to create client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := opaq.New(tt.source, tt.options...)
			if tt.wantErr {
				gt.Error(t, err)
				gt.S(t, err.Error()).Contains(tt.errContains)
				return
			}
			gt.NoError(t, err)
			gt.NotNil(t, client)
		})
	}
}

func TestClient_Query(t *testing.T) {
	policy := `package test

	default allow = false

	allow if {
		input.user == "admin"
		print("admin access granted")
	}`

	client, err := opaq.New(func() (map[string]string, error) {
		return map[string]string{"test.rego": policy}, nil
	})
	gt.NoError(t, err)

	tests := []struct {
		name        string
		query       string
		input       map[string]any
		want        bool
		wantErr     bool
		errContains string
	}{
		{
			name:  "allow admin",
			query: "data.test",
			input: map[string]any{
				"user": "admin",
			},
			want: true,
		},
		{
			name:  "deny non-admin",
			query: "data.test",
			input: map[string]any{
				"user": "user",
			},
			want: false,
		},
		{
			name:        "invalid query",
			query:       "invalid.query",
			wantErr:     true,
			errContains: "failed to evaluate query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Allow bool `json:"allow"`
			}
			err := client.Query(context.Background(), tt.query, tt.input, &result)
			if tt.wantErr {
				gt.Error(t, err)
				gt.S(t, err.Error()).Contains(tt.errContains)
				return
			}
			gt.NoError(t, err)
			gt.Value(t, result.Allow).Equal(tt.want)
		})
	}
}

func TestClient_Options(t *testing.T) {
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	policy := `package test

	allow if {
		print("testing print hook")
		true
	}`

	client, err := opaq.New(
		func() (map[string]string, error) {
			return map[string]string{"test.rego": policy}, nil
		},
		opaq.WithLogger(logger),
		opaq.WithRegoVersion(ast.RegoV1),
	)
	gt.NoError(t, err)

	var result struct {
		Allow bool `json:"allow"`
	}
	var printHookCalled bool
	var printHookMsg string
	err = client.Query(
		context.Background(),
		"data.test",
		nil,
		&result,
		opaq.WithPrintHook(func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
			printHookCalled = true
			printHookMsg = msg
			return nil
		}),
	)
	gt.NoError(t, err)
	gt.Value(t, result.Allow).Equal(true)
	gt.Value(t, printHookCalled).Equal(true)
	gt.S(t, logOutput.String()).Contains("Setting print hook")
	gt.S(t, printHookMsg).Equal("testing print hook")
}

func TestClient_Metadata(t *testing.T) {
	policy := `package test

# METADATA
# title: Test Policy
# description: A test policy
allow = true`

	client, err := opaq.New(opaq.Data("test.rego", policy))
	gt.NoError(t, err).Must()

	metadata := client.Metadata()
	gt.Array(t, metadata).Longer(0)
}

func TestClient_Sources(t *testing.T) {
	policy := `package test
	allow = true`

	source := map[string]string{"test.rego": policy}
	client, err := opaq.New(func() (map[string]string, error) {
		return source, nil
	})
	gt.NoError(t, err)

	sources := client.Sources()
	gt.Map(t, sources).Equal(source)
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestClientWithAuthzPolicy(t *testing.T) {
	client, err := opaq.New(opaq.Files("testdata/server"))
	gt.NoError(t, err)

	tests := []struct {
		name     string
		query    string
		input    map[string]any
		expected bool
	}{
		{
			name:  "allow alice user",
			query: "data.authz",
			input: map[string]any{
				"user": "alice",
			},
			expected: true,
		},
		{
			name:  "deny bob user",
			query: "data.authz",
			input: map[string]any{
				"user": "bob",
			},
			expected: false,
		},
		{
			name:  "allow admin role",
			query: "data.authz",
			input: map[string]any{
				"role": "admin",
			},
			expected: true,
		},
		{
			name:  "deny user role",
			query: "data.authz",
			input: map[string]any{
				"role": "user",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Allow bool `json:"allow"`
			}
			err := client.Query(context.Background(), tt.query, tt.input, &result)
			gt.NoError(t, err)
			gt.Value(t, result.Allow).Equal(tt.expected)
		})
	}
}

func TestMetadata(t *testing.T) {
	p, err := opaq.New(opaq.Files("testdata/metadata/pkg.rego"))
	gt.NoError(t, err)
	meta := p.Metadata()
	gt.A(t, meta).
		Longer(0).
		At(0, func(t testing.TB, v *ast.AnnotationsRef) {
			gt.Equal(t, v.Annotations.Title, "my package")
			gt.Equal(t, v.Annotations.Scope, "package")
			gt.Equal(t, v.Annotations.Custom["key"], "value")
		})
}

func TestMetadataConflict(t *testing.T) {
	_, err := opaq.New(
		opaq.Files("testdata/metadata/conflict1.rego", "testdata/metadata/conflict2.rego"),
	)
	gt.Error(t, err)
}

func TestWithRelPath(t *testing.T) {
	client, err := opaq.New(opaq.Files("testdata/server"), opaq.WithRelPath("testdata"))
	gt.NoError(t, err)

	sources := client.Sources()
	gt.Map(t, sources).HaveKey("server/policy.rego").NotHaveKey("testdata/server/policy.rego")
}
