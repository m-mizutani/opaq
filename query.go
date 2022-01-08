package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/m-mizutani/goerr"
)

type queryConfig struct {
	URL           string
	FailDefined   bool
	FailUndefined bool
	Input         string
	Output        string
	Headers       []string
}

func (x *queryConfig) Validate() error {
	if err := validation.Validate(x.URL,
		validation.Required,
		is.URL,
	); err != nil {
		return ErrInvalidConfiguration.Wrap(err)
	}

	for _, hdr := range x.Headers {
		if err := validation.Validate(hdr,
			validation.Required,
			validation.Match(regexp.MustCompile(`^[\w-]+:.+$`)),
		); err != nil {
			return ErrInvalidConfiguration.Wrap(err)
		}
	}

	return nil
}

func (x *Proc) query(ctx context.Context, cfg *queryConfig) error {
	logger.With("config", cfg).Debug("Starting inquiry")

	if err := cfg.Validate(); err != nil {
		return err
	}

	var dataInput io.Reader = x.stdin
	if cfg.Input != "-" {
		f, err := os.Open(cfg.Input)
		if err != nil {
			return goerr.Wrap(err).With("path", cfg.Input)
		}
		dataInput = f
		defer func() {
			if err := f.Close(); err != nil {
				logger.Err(err).Error(err.Error())
			}
		}()
	}

	var data interface{}
	if err := json.NewDecoder(dataInput).Decode(&data); err != nil {
		return goerr.Wrap(err).With("path", cfg.Input)
	}

	input := &QueryInput{
		URL:     cfg.URL,
		Data:    data,
		Headers: make(http.Header),
	}

	for _, hdr := range cfg.Headers {
		h := strings.Split(hdr, ":")
		input.Headers.Add(strings.TrimSpace(h[0]), strings.TrimSpace(h[1]))
	}

	var out interface{}
	client := Client{httpClient: x.httpClient}
	if err := client.Query(ctx, input, &out); err != nil {
		return err
	}

	var dataOutput io.Writer = x.stdout
	if cfg.Input != "-" {
		f, err := os.Open(cfg.Input)
		if err != nil {
			return goerr.Wrap(err).With("path", cfg.Input)
		}
		dataOutput = f
		defer func() {
			if err := f.Close(); err != nil {
				logger.Err(err).Error(err.Error())
			}
		}()
	}
	encoder := json.NewEncoder(dataOutput)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		return goerr.Wrap(err)
	}

	logger.Debug("Exiting inquiry")

	if cfg.FailDefined && !isEmpty(out) {
		return ErrExitWithNonZero
	}
	if cfg.FailUndefined && isEmpty(out) {
		return ErrExitWithNonZero
	}

	return nil
}

func isEmpty(out interface{}) bool {
	if out == nil {
		return true
	}
	switch reflect.TypeOf(out).Kind() {
	case reflect.Ptr:
		return reflect.ValueOf(out).IsNil()
	case reflect.Map, reflect.Array, reflect.Slice:
		return reflect.ValueOf(out).Len() == 0
	}
	return false
}
