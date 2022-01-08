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
	MetaData      []string
	MetaDataField string
}

func (x *queryConfig) Validate() error {
	if err := validation.Validate(x.URL,
		validation.Required,
		is.URL,
	); err != nil {
		return ErrInvalidConfiguration.Wrap(err).With("target", "--url")
	}

	for _, hdr := range x.Headers {
		if err := validation.Validate(hdr,
			validation.Required,
			validation.Match(regexp.MustCompile(`^[\w-]+:.+$`)),
		); err != nil {
			return ErrInvalidConfiguration.Wrap(err).
				With("NOTE: Expected format", "HeaderName: Value").
				With("target", "--header")
		}
	}

	if len(x.MetaData) > 0 {
		if err := validation.Validate(x.MetaDataField,
			validation.Required,
		); err != nil {
			return ErrInvalidConfiguration.Wrap(err).With("target", "--metadata-field")
		}

		for _, meta := range x.MetaData {
			if err := validation.Validate(meta,
				validation.Required,
				validation.Match(regexp.MustCompile(`^[\w-_]+=.+$`)),
			); err != nil {
				return ErrInvalidConfiguration.Wrap(err).
					With("target", "--metadata").
					With("NOTE: Expected format", "Key=Value")
			}
		}
	}

	return nil
}

func (x *Proc) query(ctx context.Context, cfg *queryConfig) error {
	logger.With("config", cfg).Debug("Starting inquiry")

	if err := cfg.Validate(); err != nil {
		return err
	}

	var data interface{}
	if len(cfg.MetaData) == 0 {
		if err := x.readData(cfg.Input, &data); err != nil {
			return err
		}
	} else {
		var tmp map[string]interface{}
		if err := x.readData(cfg.Input, &tmp); err != nil {
			return goerr.Wrap(err).With("NOTE", "input must be key-value map for metadata injection")
		}

		metadata := make(map[string]string)
		for _, meta := range cfg.MetaData {
			p := strings.Index(meta, "=")
			if p < 0 {
				panic("validation does not work for metadata")
			}
			key := meta[:p]
			value := meta[(p + 1):]
			metadata[key] = value
		}
		tmp[cfg.MetaDataField] = metadata
		data = tmp
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

	if err := x.writeData(cfg.Output, out); err != nil {
		return err
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

func (x *Proc) readData(input string, out interface{}) error {
	var dataInput io.Reader = x.stdin
	if input != "-" {
		f, err := os.Open(input)
		if err != nil {
			return goerr.Wrap(err).With("path", input)
		}
		dataInput = f
		defer func() {
			if err := f.Close(); err != nil {
				logger.Err(err).Error(err.Error())
			}
		}()
	}

	if err := json.NewDecoder(dataInput).Decode(out); err != nil {
		return goerr.Wrap(err).With("path", input)
	}
	return nil
}

func (x *Proc) writeData(output string, out interface{}) error {
	var dataOutput io.Writer = x.stdout
	if output != "-" {
		f, err := os.Create(output)
		if err != nil {
			return goerr.Wrap(err).With("path", output)
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
