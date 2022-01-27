package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/m-mizutani/goerr"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient HTTPClient
}

type opaRequest struct {
	Input interface{} `json:"input"`
}

type opaResponse struct {
	Result interface{} `json:"result"`
}

type QueryInput struct {
	Data    interface{}
	URL     string
	Headers http.Header
}

func (x *Client) Query(ctx context.Context, input *QueryInput, out interface{}) error {
	logger.With("input", input).Debug("sending query")

	inputData, err := json.Marshal(&opaRequest{Input: input.Data})
	if err != nil {
		return goerr.Wrap(err).With("input", input)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, input.URL, bytes.NewReader(inputData))
	if err != nil {
		return ErrInvalidInput.Wrap(err).With("input", input)
	}

	httpReq.Header = input.Headers
	httpReq.Header.Add("Content-Type", "application/json")

	httpResp, err := x.httpClient.Do(httpReq)
	if err != nil {
		return ErrRequestFailed.Wrap(err)
	}

	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(httpResp.Body)
		return goerr.Wrap(ErrRequestFailed, "status code is not OK").
			With("code", httpResp.StatusCode).
			With("body", string(body))
	}

	raw, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return ErrUnexpectedResp.Wrap(err).With("body", string(raw))
	}

	var opaResp opaResponse
	if err := json.Unmarshal(raw, &opaResp); err != nil {
		return ErrUnexpectedResp.Wrap(err).With("body", string(raw))
	}

	result, err := json.Marshal(opaResp.Result)
	if err != nil {
		return ErrUnexpectedResp.Wrap(err).With("opaResp", opaResp)
	}
	if err := json.Unmarshal(result, out); err != nil {
		return ErrUnexpectedResp.Wrap(err).With("result data", string(raw))
	}

	return nil
}
