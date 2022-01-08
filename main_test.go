package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	opaq "github.com/m-mizutani/opaq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stub struct {
	do func(*http.Request) (*http.Response, error)
}

func (x *stub) Do(req *http.Request) (*http.Response, error) {
	return x.do(req)
}

type opaRequest struct {
	Input interface{} `json:"input"`
}
type opaResponse struct {
	Result interface{} `json:"result"`
}
type sampleInput struct {
	User string `json:"user"`
}
type sampleResult struct {
	Allow bool `json:"allow"`
}

func toRespBody(t *testing.T, result interface{}) io.ReadCloser {
	raw, err := json.Marshal(&opaResponse{Result: result})
	require.NoError(t, err)

	return ioutil.NopCloser(bytes.NewReader(raw))
}

func toInput(t *testing.T, input interface{}) io.Reader {
	raw, err := json.Marshal(input)
	require.NoError(t, err)

	return bytes.NewReader(raw)
}

func args(argv ...string) []string {
	return append([]string{"opaq"}, argv...)
}

func bindRequest(t *testing.T, body io.Reader, out interface{}) {
	raw, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var req opaRequest
	require.NoError(t, json.Unmarshal(raw, &req))

	input, err := json.Marshal(req.Input)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(input, out))
}

func TestQuery(t *testing.T) {
	ctx := context.Background()

	t.Run("basic query", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++
				assert.Equal(t, "https://opa.example.com/xxx", r.URL.String())
				assert.Equal(t, "ABC123", r.Header.Get("X-Token"))
				assert.Equal(t, "Five", r.Header.Get("X-Sign"))

				var input sampleInput
				bindRequest(t, r.Body, &input)
				assert.Equal(t, "blue", input.User)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &sampleResult{Allow: true}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"-H", "X-Token: ABC123", // With custom header1
			"-H", "X-Sign: Five", // With custom header2
		))
		require.NoError(t, err)
		assert.Equal(t, 1, called)
	})

	t.Run("read input from file", func(t *testing.T) {
		tmp, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		defer os.Remove(tmp.Name())

		_, err = io.Copy(tmp, toInput(t, sampleInput{User: "orange"}))
		require.NoError(t, err)
		require.NoError(t, tmp.Close())

		var called int
		require.NoError(t, opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++
				var input sampleInput
				bindRequest(t, r.Body, &input)
				assert.Equal(t, "orange", input.User)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &sampleResult{Allow: true}),
				}, nil
			}}),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"-i", tmp.Name(),
		)))
		require.NoError(t, err)
		assert.Equal(t, 1, called)
	})
}

func TestExit(t *testing.T) {
	ctx := context.Background()

	t.Run("fail by fail-defined with defined response", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &sampleResult{Allow: true}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"--fail-defined",
		))
		assert.ErrorIs(t, err, opaq.ErrExitWithNonZero)
		assert.Equal(t, 1, called)
	})

	t.Run("exit normally by fail-defined with undefined response", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &struct{}{}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"--fail-defined",
		))
		assert.NoError(t, err)
		assert.Equal(t, 1, called)
	})

	t.Run("fail by fail-undefined with undefined response", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &struct{}{}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"--fail-undefined",
		))
		assert.ErrorIs(t, err, opaq.ErrExitWithNonZero)
		assert.Equal(t, 1, called)
	})

	t.Run("exit normally by fail-undefined with defined response", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &sampleResult{Allow: true}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"--fail-undefined",
		))
		assert.NoError(t, err)
		assert.Equal(t, 1, called)
	})
}

func TestMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("inject metadata", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++
				var input map[string]interface{}

				bindRequest(t, r.Body, &input)
				assert.Equal(t, "blue", input["user"])
				metadata, ok := input["metadata"].(map[string]interface{})
				require.True(t, ok)
				require.NotNil(t, metadata)
				assert.Equal(t, "five.json", metadata["filename"])

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &sampleResult{Allow: true}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"-m", "filename=five.json",
		))
		require.NoError(t, err)
		assert.Equal(t, 1, called)
	})

	t.Run("change metadata field name", func(t *testing.T) {
		var called int
		err := opaq.New(
			opaq.WithHTTPClient(&stub{do: func(r *http.Request) (*http.Response, error) {
				called++
				var input map[string]interface{}

				bindRequest(t, r.Body, &input)

				// not found by default name
				_, ok := input["metadata"].(map[string]interface{})
				assert.False(t, ok)

				// found in specified field name
				metadata, ok := input["metaverse"].(map[string]interface{})
				require.True(t, ok)
				require.NotNil(t, metadata)
				assert.Equal(t, "five.json", metadata["filename"])

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       toRespBody(t, &sampleResult{Allow: true}),
				}, nil
			}}),
			opaq.WithStdin(toInput(t, sampleInput{User: "blue"})),
		).Cmd(ctx, args(
			"-u", "https://opa.example.com/xxx", // URL
			"-m", "filename=five.json",
			"--metadata-field", "metaverse",
		))
		require.NoError(t, err)
		assert.Equal(t, 1, called)
	})

}

func TestInvalidOption(t *testing.T) {
	testCases := []struct {
		desc string
		args []string
		err  error
	}{
		{
			desc: "No URL must fail",
			args: args(),
		},
		{
			desc: "Invalid URL must fail",
			args: args("-u", "invalid_url"),
			err:  opaq.ErrInvalidConfiguration,
		},
		{
			desc: "Invalid header must fail",
			args: args("-u", "https://example.com", "-H", "invalid header"),
			err:  opaq.ErrInvalidConfiguration,
		},
		{
			desc: "Invalid metadata fails",
			args: args("-u", "https://example.com", "-m", "foo"),
			err:  opaq.ErrInvalidConfiguration,
		},
		{
			desc: "No metadata field name fails",
			args: args("-u", "https://example.com", "-m", "foo=baa", "--metadata-field="),
			err:  opaq.ErrInvalidConfiguration,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := opaq.New().Cmd(context.Background(), tC.args)
			assert.Error(t, err)
			if tC.err != nil {
				assert.ErrorIs(t, tC.err, err)
			}
		})
	}
}
